package tcpusb

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"net"
	"sync"
	"time"
)

// Service handles a single ADB service connection
type Service struct {
	client    ADBClient
	serial    string
	localID   uint32
	remoteID  uint32
	socket    *Socket
	transport net.Conn
	opened    bool
	ended     bool
	needAck   bool
	mu        sync.Mutex
	endCh     chan struct{}
	doneCh    chan struct{} // Signal when service is completely done
}

// ADBClient interface for interacting with ADB
type ADBClient interface {
	Transport(serial string) (net.Conn, error)
}

// NewService creates a new service instance
func NewService(client ADBClient, serial string, localID, remoteID uint32, socket *Socket) *Service {
	return &Service{
		client:   client,
		serial:   serial,
		localID:  localID,
		remoteID: remoteID,
		socket:   socket,
		opened:   false,
		ended:    false,
		needAck:  false,
		endCh:    make(chan struct{}),
		doneCh:   make(chan struct{}),
	}
}

// Wait waits for the service to completely finish
func (s *Service) Wait() {
	<-s.doneCh
}

// End closes the service
func (s *Service) End() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.ended {
		return
	}

	if s.transport != nil {
		s.transport.Close()
	}

	localID := s.localID
	if !s.opened {
		localID = 0 // Zero can only mean a failed open
	}

	// Send close packet
	packet := Assemble(A_CLSE, localID, s.remoteID, nil)
	s.socket.Write(packet)

	s.transport = nil
	s.ended = true
	close(s.endCh)
}

// Handle processes a packet for this service
func (s *Service) Handle(packet *Packet) error {
	switch packet.Command {
	case A_OPEN:
		return s.handleOpenPacket(packet)
	case A_OKAY:
		return s.handleOkayPacket(packet)
	case A_WRTE:
		return s.handleWritePacket(packet)
	case A_CLSE:
		return s.handleClosePacket(packet)
	default:
		return fmt.Errorf("unexpected packet: %s", packet.String())
	}
}

func (s *Service) handleOpenPacket(packet *Packet) error {
	slog.Debug("I:A_OPEN", "packet", packet.String())

	// Get transport connection
	transport, err := s.client.Transport(s.serial)
	if err != nil {
		return err
	}

	s.mu.Lock()
	if s.ended {
		s.mu.Unlock()
		transport.Close()
		return ErrLateTransport
	}
	s.transport = transport
	s.mu.Unlock()

	// Send service name to device (discard null byte at end)
	serviceName := packet.Data[:len(packet.Data)-1]
	if _, err := s.transport.Write(encodeData(serviceName)); err != nil {
		return err
	}

	// Read OKAY or FAIL response
	reply := make([]byte, 4)
	if _, err := io.ReadFull(s.transport, reply); err != nil {
		return err
	}

	if string(reply) == "OKAY" {
		slog.Debug("O:A_OKAY")
		okayPacket := Assemble(A_OKAY, s.localID, s.remoteID, nil)
		s.socket.Write(okayPacket)
		s.opened = true

		// Start forwarding data from device in background
		go s.forwardFromDevice()

		// Wait for the service to complete
		s.Wait()
		return nil
	} else if string(reply) == "FAIL" {
		// Read error message
		lenBuf := make([]byte, 4)
		io.ReadFull(s.transport, lenBuf)
		// Handle error...
		return fmt.Errorf("device returned FAIL")
	}

	return nil
}

func (s *Service) handleOkayPacket(packet *Packet) error {
	slog.Debug("I:A_OKAY", "packet", packet.String())

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.ended {
		return nil
	}

	if s.transport == nil {
		return ErrPrematurePacket
	}

	s.needAck = false
	// Don't call tryPush here - let forwardFromDevice handle it

	return nil
}

func (s *Service) handleWritePacket(packet *Packet) error {
	slog.Debug("I:A_WRTE", "packet", packet.String())

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.ended {
		return nil
	}

	if s.transport == nil {
		return ErrPrematurePacket
	}

	if len(packet.Data) > 0 {
		if _, err := s.transport.Write(packet.Data); err != nil {
			return err
		}
	}

	slog.Debug("O:A_OKAY")
	okayPacket := Assemble(A_OKAY, s.localID, s.remoteID, nil)
	s.socket.Write(okayPacket)

	return nil
}

func (s *Service) handleClosePacket(packet *Packet) error {
	slog.Debug("I:A_CLSE", "packet", packet.String())

	if s.transport == nil {
		return ErrPrematurePacket
	}

	s.End()
	return nil
}

func (s *Service) forwardFromDevice() {
	defer func() {
		s.End()
		close(s.doneCh) // Signal that we're completely done
	}()

	buffer := make([]byte, 65536)
	for {
		s.mu.Lock()
		if s.ended || s.transport == nil {
			s.mu.Unlock()
			return
		}

		if s.needAck {
			s.mu.Unlock()
			// Wait a bit before checking again to avoid busy loop
			select {
			case <-s.endCh:
				return
			case <-time.After(10 * time.Millisecond):
				continue
			}
		}

		maxPayload := s.socket.MaxPayload()
		if maxPayload > uint32(len(buffer)) {
			maxPayload = uint32(len(buffer))
		}

		transport := s.transport
		s.mu.Unlock()

		n, err := transport.Read(buffer[:maxPayload])
		if err != nil {
			if err != io.EOF {
				slog.Debug("Transport read error", "error", err)
			}
			return
		}

		if n > 0 {
			s.mu.Lock()
			if !s.ended {
				slog.Debug("O:A_WRTE")
				chunk := make([]byte, n)
				copy(chunk, buffer[:n])
				writePacket := Assemble(A_WRTE, s.localID, s.remoteID, chunk)
				s.socket.Write(writePacket)
				s.needAck = true
			}
			s.mu.Unlock()
		}
	}
}

func (s *Service) tryPush() {
	// This is called with lock held
	if s.needAck || s.ended || s.transport == nil {
		return
	}

	buffer := make([]byte, s.socket.MaxPayload())
	n, err := s.transport.Read(buffer)
	if err != nil || n == 0 {
		return
	}

	slog.Debug("O:A_WRTE")
	chunk := buffer[:n]
	writePacket := Assemble(A_WRTE, s.localID, s.remoteID, chunk)
	s.socket.Write(writePacket)
	s.needAck = true
}

// encodeData encodes data with length prefix for ADB protocol
func encodeData(data []byte) []byte {
	length := make([]byte, 4)
	// Write length in hex ASCII
	lengthStr := fmt.Sprintf("%04x", len(data))
	copy(length, lengthStr)

	var buf bytes.Buffer
	buf.Write(length)
	buf.Write(data)
	return buf.Bytes()
}
