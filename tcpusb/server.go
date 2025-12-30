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
	client      *adb.Client
	serial      string
	authHandler AuthHandler
	listener    net.Listener
	connections []*Socket
	mu          sync.Mutex
	closed      bool
}

// NewServer creates a new TCP-USB bridge server
func NewServer(client *adb.Client, serial string, authHandler AuthHandler) *Server {
	return &Server{
		client:      client,
		serial:      serial,
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

		socket := NewSocket(srv.client, srv.serial, conn, srv.authHandler)

		srv.mu.Lock()
		srv.connections = append(srv.connections, socket)
		srv.mu.Unlock()

		go func() {
			defer func() {
				srv.mu.Lock()
				// Remove from connections
				for i, s := range srv.connections {
					if s == socket {
						srv.connections = append(srv.connections[:i], srv.connections[i+1:]...)
						break
					}
				}
				srv.mu.Unlock()
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
	defer srv.mu.Unlock()

	if srv.closed {
		return nil
	}

	srv.closed = true

	// Close all connections
	for _, conn := range srv.connections {
		conn.End()
	}
	srv.connections = nil

	// Close listener
	if srv.listener != nil {
		return srv.listener.Close()
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
