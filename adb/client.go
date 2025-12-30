package adb

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strconv"
)

// Connector is an interface that abstracts connection creation for testability
type Connector interface {
	Connection() (net.Conn, error)
}

// connectorImpl implements Connector with a specific address
type connectorImpl struct {
	host string
	port int
}

func (c connectorImpl) Connection() (net.Conn, error) {
	return net.Dial("tcp", c.addr())
}

func (c connectorImpl) addr() string {
	return fmt.Sprintf("%s:%d", c.host, c.port)
}

// autoStartConnector wraps a Connector and automatically starts ADB server if needed
type autoStartConnector struct {
	base connectorImpl
	bin  string
}

func (c *autoStartConnector) Connection() (net.Conn, error) {
	// Try to connect first
	conn, err := c.base.Connection()
	if err != nil {
		// Connection failed, try to start the server
		_ = exec.Command(c.bin, "start-server").Run()
		// Try again
		conn, err = c.base.Connection()
	}
	return conn, err
}

// Client is a minimal ADB client for talking to the adb server
type Client struct {
	connector Connector
}

// DeviceInfo represents a device returned by `host:devices`
type DeviceInfo struct {
	Serial string
	State  string
}

// DeviceWithPath represents a device with additional path information
type DeviceWithPath struct {
	Serial string
	State  string
	Model  string
	Device string
}

// ForwardEntry represents a forwarded port
type ForwardEntry struct {
	Serial string
	Local  string
	Remote string
}

// ReverseEntry represents a reverse-forwarded port
type ReverseEntry struct {
	Remote string
	Local  string
}

// ClientOptions holds configuration for creating a new Client
type ClientOptions struct {
	Host string
	Port int
	Bin  string // Path to adb binary (optional, for spawning)
}

// defaultADBName returns the default ADB binary name for the current platform
func defaultADBName() string {
	if runtime.GOOS == "windows" {
		return "adb.exe"
	}
	return "adb"
}

// NewClient creates a new Client with default options
func NewClient() *Client {
	return NewClientWithOptions(ClientOptions{})
}

func getEnvOrDefault(envVar, defaultVal string) string {
	if val := os.Getenv(envVar); val != "" {
		return val
	}
	return defaultVal
}

// NewClientWithOptions creates a new Client with custom options
// It will automatically start the ADB server if connection fails
func NewClientWithOptions(opts ClientOptions) *Client {
	if opts.Host == "" {
		opts.Host = getEnvOrDefault("ANDROID_ADB_SERVER_HOST", "127.0.0.1")
	}
	if opts.Port == 0 {
		opts.Port = 5037
		if portStr := os.Getenv("ANDROID_ADB_SERVER_PORT"); portStr != "" {
			if port, err := strconv.Atoi(portStr); err == nil {
				opts.Port = port
			}
		} 
	}
	if opts.Bin == "" {
		opts.Bin = defaultADBName()
	}

	connector := &autoStartConnector{
		base: connectorImpl{host: opts.Host, port: opts.Port},
		bin:  opts.Bin,
	}

	return &Client{
		connector: connector,
	}
}

// NewClientWithConnector creates a new Client with a custom Connector (useful for testing)
func NewClientWithConnector(connector Connector) *Client {
	return &Client{connector: connector}
}

// Device returns a Device instance for interacting with a specific device
// Use DeviceWithSerial() to create a descriptor for a specific device:
//
//	device := client.Device(adb.DeviceWithSerial("emulator-5554"))
//	device.Shell("ls /sdcard")
func (c *Client) Device(descriptor DeviceDescriptor) *Device {
	return &Device{
		client:     c,
		descriptor: descriptor,
	}
}
