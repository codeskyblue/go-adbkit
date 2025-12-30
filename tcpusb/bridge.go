package tcpusb

import (
	"fmt"
	"log/slog"

	"github.com/codeskyblue/go-adbkit/adb"
)

// Config holds the configuration for a USB-to-TCP bridge
type Config struct {
	Port        int
	ADBHost     string
	ADBPort     int
	AuthHandler AuthHandler
}

// Bridge represents a USB-to-TCP bridge
type Bridge struct {
	Serial string
	Config Config
	Logger *slog.Logger
}

// DefaultConfig returns a Config with sensible defaults
func DefaultConfig() Config {
	return Config{
		Port:    6174,
		ADBHost: "127.0.0.1",
		ADBPort: 5037,
		AuthHandler: func(publicKey []byte) error {
			// Default: always accept connections
			return nil
		},
	}
}

// NewBridge creates a new USB-to-TCP bridge with default configuration
func NewBridge(serial string) *Bridge {
	return &Bridge{
		Serial: serial,
		Config: DefaultConfig(),
		Logger: slog.Default(),
	}
}

// Start starts the USB-to-TCP bridge server
func (b *Bridge) Start() error {
	client := adb.NewClient(b.Config.ADBHost, b.Config.ADBPort)
	server := NewServer(client, b.Serial, b.Config.AuthHandler)

	address := fmt.Sprintf(":%d", b.Config.Port)
	b.Logger.Info("Starting USB-to-TCP bridge", "device", b.Serial, "port", b.Config.Port)

	return server.Listen(address)
}

// StartWithServer starts the bridge and returns the server instance for manual control
func (b *Bridge) StartWithServer() (*Server, error) {
	client := adb.NewClient(b.Config.ADBHost, b.Config.ADBPort)
	server := NewServer(client, b.Serial, b.Config.AuthHandler)

	address := fmt.Sprintf(":%d", b.Config.Port)
	b.Logger.Info("Starting USB-to-TCP bridge", "device", b.Serial, "port", b.Config.Port)

	go func() {
		if err := server.Listen(address); err != nil {
			b.Logger.Error("Server error", "error", err)
		}
	}()

	return server, nil
}
