package adb

// ClientOptions holds configuration for creating a new Client
type ClientOptions struct {
	Host string
	Port int
	Bin  string // Path to adb binary (optional, for spawning)
}

// Client is a minimal ADB client for talking to the adb server
type Client struct {
	Host string
	Port int
	Bin  string
}

// Device represents a device returned by `host:devices`
type Device struct {
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

// NewClient creates a new Client with default options
func NewClient(host string, port int) *Client {
	if host == "" {
		host = "127.0.0.1"
	}
	if port == 0 {
		port = 5037
	}
	return &Client{Host: host, Port: port, Bin: "adb"}
}

// NewClientWithOptions creates a new Client with custom options
func NewClientWithOptions(opts ClientOptions) *Client {
	if opts.Host == "" {
		opts.Host = "127.0.0.1"
	}
	if opts.Port == 0 {
		opts.Port = 5037
	}
	if opts.Bin == "" {
		opts.Bin = "adb"
	}
	return &Client{Host: opts.Host, Port: opts.Port, Bin: opts.Bin}
}
