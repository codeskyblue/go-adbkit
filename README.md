# go-adbkit

Go implementation of USB-to-TCP bridge for Android Debug Bridge (ADB).

## Features

- **USB-to-TCP Bridge**: Expose USB-connected Android devices over TCP network
- **Multiple Devices**: Support for bridging multiple devices simultaneously
- **Authentication**: Customizable authentication handlers
- **Clean API**: Simple and intuitive Go API for easy integration

## Installation

```bash
go get github.com/codeskyblue/adbkit
```

## Quick Start

### Command Line Tool

```bash
# Build the command line tool
cd cmd/usb-bridge
go build

# Run the bridge for a specific device
./usb-bridge -serial <device-serial> -port 6174

# Run with verbose logging (shows ADB protocol debug messages)
./usb-bridge -serial <device-serial> -port 6174 --verbose

# Connect from another machine
adb connect <your-ip>:6174
```

### As a Library

```go
package main

import (
    "log"
    "github.com/codeskyblue/adbkit/tcpusb"
)

func main() {
    // Create a bridge for your device
    bridge := tcpusb.NewBridge("your-device-serial")
    bridge.Config.Port = 6174
    
    // Start the bridge (blocking)
    if err := bridge.Start(); err != nil {
        log.Fatal(err)
    }
}
```

## API Documentation

### Creating a Bridge

```go
// Simple bridge with defaults (port 6174, localhost ADB server)
bridge := tcpusb.NewBridge("device-serial")

// Bridge with custom configuration
bridge := tcpusb.NewBridge("device-serial")
bridge.Config.Port = 7000                // Custom port
bridge.Config.ADBHost = "192.168.1.100"  // Remote ADB server
bridge.Config.ADBPort = 5037              // Custom ADB port
bridge.Config.AuthHandler = myAuthFunc    // Custom authentication
```

### Configuration

- **Port**: TCP port for the bridge server (default: 6174)
- **ADBHost**: ADB server host (default: "127.0.0.1")
- **ADBPort**: ADB server port (default: 5037)
- **AuthHandler**: Custom authentication handler function

### Starting the Bridge

```go
// Blocking call - starts server and handles connections
err := bridge.Start()

// Non-blocking - returns server instance for manual control
server, err := bridge.StartWithServer()
defer server.Close()
```

### Custom Authentication

```go
authHandler := func(publicKey []byte) error {
    // Verify the public key
    // Return nil to accept, error to reject
    log.Printf("Device with key %x attempting to connect", publicKey[:20])
    
    if isAuthorized(publicKey) {
        return nil
    }
    return errors.New("unauthorized device")
}

bridge := tcpusb.NewBridge("serial", 
    tcpusb.WithAuthHandler(authHandler),
)
```

## Use Cases

### 1. Remote Development
Access USB-connected Android devices over the network for remote development and testing.

```go
bridge := tcpusb.NewBridge("device-serial", 
    tcpusb.WithPort(6174),
)
bridge.Start()

// On remote machine:
// adb connect <server-ip>:6174
```

### 2. CI/CD Integration
Integrate physical devices into your CI/CD pipeline by exposing them over TCP.

```go
// In your CI server
server, err := bridge.StartWithServer()
if err != nil {
    log.Fatal(err)
}

// Run your tests
runTests()

// Clean up
server.Close()
```

### 3. Device Farm
Create a device farm by bridging multiple devices on different ports.

```go
devices := []string{"device1", "device2", "device3"}
port := 6174

for _, serial := range devices {
    bridge := tcpusb.NewBridge(serial, 
        tcpusb.WithPort(port),
    )
    go bridge.Start()
    port++
}
```

## Architecture

The implementation follows the ADB protocol specification:

1. **Packet Layer**: Handles ADB protocol packets (SYNC, CNXN, OPEN, OKAY, WRTE, CLSE, AUTH)
2. **Socket Layer**: Manages client connections and authentication
3. **Service Layer**: Forwards data between TCP clients and USB devices
4. **Server Layer**: Accepts TCP connections and creates socket handlers

```
TCP Client (adb) <-> Socket <-> Service <-> ADB Server <-> USB Device
```

## Comparison with Node.js Implementation

This is a port of the `usb-device-to-tcp` feature from [openstf/adbkit](https://github.com/openstf/adbkit).

**Advantages of Go implementation:**
- No runtime dependencies (single binary)
- Better performance and lower memory usage
- Type safety
- Easy deployment
- Built-in concurrency support

## Examples

See the [examples](./examples) directory for more usage examples:

- [basic](./examples/basic) - Basic usage examples
- More examples coming soon...

## Requirements

- Go 1.18 or later
- ADB server running (usually started by Android SDK)
- USB-connected Android device

## Troubleshooting

**Bridge won't start:**
- Make sure ADB server is running: `adb start-server`
- Check if the port is already in use: `lsof -i :<port>`

**Can't connect from remote machine:**
- Ensure firewall allows connections on the bridge port
- Verify the device serial is correct: `adb devices`

**Device not authorized:**
- Accept the authorization prompt on the Android device
- Or implement a custom auth handler to auto-accept

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

MIT License - see LICENSE file for details

## Credits

Based on the Node.js implementation from [openstf/adbkit](https://github.com/openstf/adbkit)
