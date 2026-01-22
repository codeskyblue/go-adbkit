package tcpusb

import (
	"fmt"
	"log/slog"
	"net"
	"sync"

	"github.com/codeskyblue/go-adbkit/adb"
)

// Server provides USB devices over TCP using a translating proxy
type Server struct {
	device      *adb.Device
	authHandler AuthHandler
	listener    net.Listener
	connections []*Socket
	mu          sync.Mutex
	closed      bool
}

// NewServer creates a new TCP-USB bridge server
func NewServer(device *adb.Device, authHandler AuthHandler) *Server {
	return &Server{
		device:      device,
		authHandler: authHandler,
		connections: make([]*Socket, 0),
		closed:      false,
	}
}

// Listen starts listening on the specified address
func (srv *Server) Listen(address string) error {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}

	srv.mu.Lock()
	srv.listener = listener
	srv.mu.Unlock()

	slog.Info("TCP-USB bridge listening", "address", address)
	slog.Info("Connect with", "command", fmt.Sprintf("adb connect %s", address))

	for {
		conn, err := listener.Accept()
		if err != nil {
			srv.mu.Lock()
			closed := srv.closed
			srv.mu.Unlock()

			if closed {
				return nil
			}
			slog.Error("Accept error", "error", err)
			continue
		}

		socket := NewSocket(srv.device, conn, srv.authHandler)

		srv.mu.Lock()
		srv.connections = append(srv.connections, socket)
		srv.mu.Unlock()

		go func() {
			defer func() {
				srv.mu.Lock()
				defer srv.mu.Unlock()
				
				// Skip cleanup if server is already closed
				if srv.closed {
					return
				}
				
				// Remove from connections
				for i, s := range srv.connections {
					if s == socket {
						srv.connections = append(srv.connections[:i], srv.connections[i+1:]...)
						break
					}
				}
			}()

			if err := socket.Start(); err != nil {
				slog.Debug("Socket closed", "error", err)
			}
		}()
	}
}

// Close closes the server
func (srv *Server) Close() error {
	srv.mu.Lock()
	
	if srv.closed {
		srv.mu.Unlock()
		return nil
	}

	srv.closed = true

	// Copy connections to avoid holding lock while closing
	connections := make([]*Socket, len(srv.connections))
	copy(connections, srv.connections)
	srv.connections = nil

	listener := srv.listener
	srv.mu.Unlock()

	// Close all connections without holding the lock
	for _, conn := range connections {
		conn.End()
	}

	// Close listener
	if listener != nil {
		return listener.Close()
	}

	return nil
}

// Addr returns the listener address
func (srv *Server) Addr() net.Addr {
	srv.mu.Lock()
	defer srv.mu.Unlock()

	if srv.listener != nil {
		return srv.listener.Addr()
	}
	return nil
}
