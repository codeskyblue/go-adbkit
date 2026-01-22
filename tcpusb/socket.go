package tcpusb

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"log/slog"
	"net"
	"sync"

	"github.com/codeskyblue/go-adbkit/adb"
)

const (
	UINT32_MAX = 0xFFFFFFFF
	UINT16_MAX = 0xFFFF

	AUTH_TOKEN        = 1
	AUTH_SIGNATURE    = 2
	AUTH_RSAPUBLICKEY = 3

	TOKEN_LENGTH = 20
)

// AuthHandler is called when a device attempts to authenticate
type AuthHandler func(publicKey []byte) error

// Socket represents a connection from an ADB client
type Socket struct {
	device        *adb.Device
	conn          net.Conn
	authHandler   AuthHandler
	reader        *PacketReader
	version       uint32
	maxPayload    uint32
	authorized    bool
	syncToken     *RollingCounter
	remoteID      *RollingCounter
	services      *ServiceMap
	remoteAddress string
	token         []byte
	signature     []byte
	ended         bool
	mu            sync.Mutex
	writeMu       sync.Mutex
}

// NewSocket creates a new socket instance
func NewSocket(device *adb.Device, conn net.Conn, authHandler AuthHandler) *Socket {
	if authHandler == nil {
		authHandler = func(publicKey []byte) error {
			// Default: always accept
			return nil
		}
	}

	socket := &Socket{
		device:        device,
		conn:          conn,
		authHandler:   authHandler,
		reader:        NewPacketReader(conn),
		version:       1,
		maxPayload:    4096,
		authorized:    false,
		syncToken:     NewRollingCounter(UINT32_MAX),
		remoteID:      NewRollingCounter(UINT32_MAX),
		services:      NewServiceMap(),
		remoteAddress: conn.RemoteAddr().String(),
		ended:         false,
	}

	// Set TCP_NODELAY
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		tcpConn.SetNoDelay(true)
	}

	return socket
}

// Start begins processing packets
func (s *Socket) Start() error {
	defer s.End()

	for {
		packet, err := s.reader.ReadPacket()
		if err != nil {
			slog.Debug("PacketReader error", "error", err)
			return err
		}

		if err := s.handle(packet); err != nil {
			slog.Error("Packet handling error", "error", err)
			return err
		}
	}
}

// End closes the socket and all services
func (s *Socket) End() {
	s.mu.Lock()
	if s.ended {
		s.mu.Unlock()
		return
	}

	s.ended = true
	services := s.services
	conn := s.conn
	s.mu.Unlock()

	// End services without holding the lock to avoid deadlock
	// Service.End() calls socket.Write() which needs to check s.ended
	services.End()
	conn.Close()
}

// Write writes data to the socket
func (s *Socket) Write(data []byte) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	s.mu.Lock()
	if s.ended {
		s.mu.Unlock()
		return fmt.Errorf("socket ended")
	}
	s.mu.Unlock()

	_, err := s.conn.Write(data)
	return err
}

// MaxPayload returns the maximum payload size
func (s *Socket) MaxPayload() uint32 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.maxPayload
}

func (s *Socket) handle(packet *Packet) error {
	s.mu.Lock()
	if s.ended {
		s.mu.Unlock()
		return nil
	}
	s.mu.Unlock()

	switch packet.Command {
	case A_SYNC:
		return s.handleSyncPacket(packet)
	case A_CNXN:
		return s.handleConnectionPacket(packet)
	case A_OPEN:
		return s.handleOpenPacket(packet)
	case A_OKAY, A_WRTE, A_CLSE:
		return s.forwardServicePacket(packet)
	case A_AUTH:
		return s.handleAuthPacket(packet)
	default:
		return fmt.Errorf("unknown command: 0x%08x", packet.Command)
	}
}

func (s *Socket) handleSyncPacket(packet *Packet) error {
	slog.Debug("I:A_SYNC")
	slog.Debug("O:A_SYNC")

	s.mu.Lock()
	syncPacket := Assemble(A_SYNC, 1, s.syncToken.Next(), nil)
	s.mu.Unlock()

	return s.Write(syncPacket)
}

func (s *Socket) handleConnectionPacket(packet *Packet) error {
	slog.Debug("I:A_CNXN", "packet", packet.String())

	_ = Swap32(packet.Arg0) // version

	s.mu.Lock()
	if packet.Arg1 < UINT16_MAX {
		s.maxPayload = packet.Arg1
	} else {
		s.maxPayload = UINT16_MAX
	}
	s.mu.Unlock()

	// Create challenge token
	token := make([]byte, TOKEN_LENGTH)
	if _, err := rand.Read(token); err != nil {
		return err
	}

	s.mu.Lock()
	s.token = token
	s.mu.Unlock()

	slog.Debug("Created challenge", "token", base64.StdEncoding.EncodeToString(token))
	slog.Debug("O:A_AUTH")

	authPacket := Assemble(A_AUTH, AUTH_TOKEN, 0, token)
	return s.Write(authPacket)
}

func (s *Socket) handleAuthPacket(packet *Packet) error {
	slog.Debug("I:A_AUTH", "packet", packet.String())

	switch packet.Arg0 {
	case AUTH_SIGNATURE:
		// Store first signature
		s.mu.Lock()
		if s.signature == nil {
			s.signature = packet.Data
			slog.Debug("Received signature", "signature", base64.StdEncoding.EncodeToString(packet.Data))
		}
		s.mu.Unlock()
		// return s.sendCNXN()

		slog.Debug("O:A_AUTH")
		token := s.token
		authPacket := Assemble(A_AUTH, AUTH_TOKEN, 0, token)
		return s.Write(authPacket)

	case AUTH_RSAPUBLICKEY:
		s.mu.Lock()
		signature := s.signature
		_ = s.token // token
		s.mu.Unlock()

		if signature == nil {
			return ErrAuthFailed
		}

		if len(packet.Data) < 2 {
			return ErrAuthFailed
		}

		slog.Debug("Received RSA public key")

		// For simplicity, we'll skip actual RSA verification in this implementation
		// In production, you should verify the signature properly

		// Call auth handler
		if err := s.authHandler(packet.Data); err != nil {
			return ErrAuthFailed
		}
		s.authorized = true
		return s.sendCNXN()
	default:
		return fmt.Errorf("unknown authentication method: %d", packet.Arg0)
	}
}

// CNXN means "connection"
// When the client is authorized, send a CNXN packet
func (s *Socket) sendCNXN() error {
	s.mu.Lock()
	version := s.version
	maxPayload := s.maxPayload
	deviceID := s.getDeviceID()
	s.mu.Unlock()

	slog.Debug("O:A_CNXN")
	cnxnPacket := Assemble(A_CNXN, Swap32(version), maxPayload, deviceID)
	return s.Write(cnxnPacket)
}

func (s *Socket) handleOpenPacket(packet *Packet) error {
	s.mu.Lock()
	authorized := s.authorized
	s.mu.Unlock()

	if !authorized {
		return ErrUnauthorized
	}

	remoteID := packet.Arg0

	s.mu.Lock()
	localID := s.remoteID.Next()
	s.mu.Unlock()

	if len(packet.Data) < 2 {
		return fmt.Errorf("empty service name")
	}

	serviceName := packet.Data[:len(packet.Data)-1] // Remove null terminator
	slog.Info("Calling service", "name", string(serviceName))

	service := NewService(s.device, localID, remoteID, s)

	if err := s.services.Insert(localID, service); err != nil {
		return err
	}

	slog.Debug("Services active", "count", s.services.Count())

	// Handle in goroutine
	go func() {
		defer func() {
			s.services.Remove(localID)
			slog.Debug("Services active", "count", s.services.Count())
			service.End()
		}()

		if err := service.Handle(packet); err != nil {
			slog.Error("Service error", "error", err)
		}
	}()

	return nil
}

func (s *Socket) forwardServicePacket(packet *Packet) error {
	s.mu.Lock()
	authorized := s.authorized
	s.mu.Unlock()

	if !authorized {
		return ErrUnauthorized
	}

	localID := packet.Arg1

	service := s.services.Get(localID)
	if service != nil {
		return service.Handle(packet)
	}

	slog.Debug("Received packet for closed service")
	return nil
}

func (s *Socket) getDeviceID() []byte {
	// Create a simple device ID
	// In production, you should get actual device properties
	serial, err := s.device.Serial()
	if err != nil {
		serial = "unknown"
	}
	deviceID := fmt.Sprintf("device::ro.product.name=adb-bridge;ro.product.model=%s;ro.product.device=%s;\x00",
		serial, serial)
	return []byte(deviceID)
}

// verifySignature verifies an RSA signature (simplified version)
func verifySignature(publicKey *rsa.PublicKey, token, signature []byte) error {
	hash := sha1.Sum(token)
	return rsa.VerifyPKCS1v15(publicKey, 0, hash[:], signature)
}
