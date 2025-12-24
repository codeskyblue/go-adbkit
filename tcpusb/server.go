package tcpusb

import (
	"fmt"
	"log/slog"
	"net"
	"sync"
)

// Server provides USB devices over TCP using a translating proxy
type Server struct {
	client      ADBClient
	serial      string
	authHandler AuthHandler
	listener    net.Listener
	connections []*Socket
	mu          sync.Mutex
	closed      bool
}

// NewServer creates a new TCP-USB bridge server
func NewServer(client ADBClient, serial string, authHandler AuthHandler) *Server {
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

// DefaultADBClient provides a default ADB client implementation
type DefaultADBClient struct {
	host string
	port int
}

// NewDefaultADBClient creates a new default ADB client
func NewDefaultADBClient(host string, port int) *DefaultADBClient {
	if host == "" {
		host = "127.0.0.1"
	}
	if port == 0 {
		port = 5037
	}
	return &DefaultADBClient{
		host: host,
		port: port,
	}
}

// Transport establishes a transport connection to the specified device
func (c *DefaultADBClient) Transport(serial string) (net.Conn, error) {
	addr := fmt.Sprintf("%s:%d", c.host, c.port)
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}

	// Send transport command
	cmd := fmt.Sprintf("host:transport:%s", serial)
	if err := sendADBCommand(conn, cmd); err != nil {
		conn.Close()
		return nil, err
	}

	return conn, nil
}

// sendADBCommand sends a command to the ADB server
func sendADBCommand(conn net.Conn, command string) error {
	// Format: 4-byte hex length + command
	length := fmt.Sprintf("%04x", len(command))
	message := length + command

	if _, err := conn.Write([]byte(message)); err != nil {
		return err
	}

	// Read response (OKAY or FAIL)
	response := make([]byte, 4)
	if _, err := conn.Read(response); err != nil {
		return err
	}

	if string(response) != "OKAY" {
		return fmt.Errorf("ADB command failed: %s", string(response))
	}

	return nil
}
