# go-adbkit

A pure Go client library for the Android Debug Bridge (ADB) protocol.

## Features

- **Pure Go Implementation**: No external dependencies on ADB binaries
- **Comprehensive API Support**:
  - Host commands: version, devices, connect, disconnect, kill
  - Device commands: shell, file sync, port forwarding, logcat
  - Transport management
- **Simple & Clean API**: Easy to use and integrate into your Go projects
- **Testable**: Mock-friendly design with connector interface for testing

## Installation

```bash
go get github.com/codeskyblue/go-adbkit
```

## Quick Start

### Basic Usage

```go
package main

import (
    "fmt"
    "log"
    "github.com/codeskyblue/go-adbkit/adb"
)

func main() {
    // Create a new ADB client (connects to localhost:5037 by default)
    client := adb.NewClient()

    // Get ADB server version
    version, err := client.Version()
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("ADB version: %d\n", version)

    // List connected devices
    devices, err := client.ListDevices()
    if err != nil {
        log.Fatal(err)
    }
    for _, device := range devices {
        fmt.Printf("Device: %s (%s)\n", device.Serial, device.State)
    }
}
```

### Running Shell Commands

```go
// Get a device shell client
deviceClient := client.DeviceClient("device-serial")

// Run a shell command
output, err := deviceClient.RunCommand("getprop ro.product.model")
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Device model: %s\n", output)
```

### File Operations

```go
deviceClient := client.DeviceClient("device-serial")

// List files
entries, err := deviceClient.List("/sdcard/")
if err != nil {
    log.Fatal(err)
}
for _, entry := range entries {
    fmt.Printf("%s %d %s\n", entry.Mode, entry.Size, entry.Name)
}

// Pull file from device
err = deviceClient.Pull("/sdcard/file.txt", "/local/path/file.txt")
if err != nil {
    log.Fatal(err)
}

// Push file to device
err = deviceClient.Push("/local/path/file.txt", "/sdcard/file.txt")
if err != nil {
    log.Fatal(err)
}
```

### Port Forwarding

```go
// Forward local port to device
err := client.Forward("device-serial", "tcp:8080", "tcp:8080")
if err != nil {
    log.Fatal(err)
}

// List forwardings
forwards, err := client.ListForward("device-serial")
if err != nil {
    log.Fatal(err)
}
for _, fwd := range forwards {
    fmt.Printf("%s -> %s\n", fwd.Local, fwd.Remote)
}

// Remove forwarding
err = client.ForwardRemove("device-serial", "tcp:8080")
if err != nil {
    log.Fatal(err)
}
```

### Device Tracking

```go
// Track device changes with callback
err := client.TrackDevicesWithCallback(func(devices []adb.Device) {
    fmt.Printf("Devices changed: %d devices\n", len(devices))
    for _, dev := range devices {
        fmt.Printf("  %s: %s\n", dev.Serial, dev.State)
    }
})
if err != nil {
    log.Fatal(err)
}
```

## API Reference

### Client

#### Creating a Client

```go
// Default client (localhost:5037)
client := adb.NewClient()

// With options (auto-starts ADB server if needed)
client := adb.NewClientWithOptions(adb.ClientOptions{
    Host: "127.0.0.1",
    Port: 5037,
    Bin:  "adb", // Path to adb binary
})

// With custom connector (useful for testing)
client := adb.NewClientWithConnector(customConnector)
```

#### Host Commands

```go
// Get ADB server version
version, err := client.Version()

// List connected devices
devices, err := client.ListDevices()
// Returns: []Device{Serial: "xxx", State: "device"}

// List devices with paths
devices, err := client.ListDevicesWithPaths()
// Returns: []DeviceWithPath{Serial, State, Model, Device}

// Connect to remote device
msg, err := client.Connect("192.168.1.100", 5555)

// Disconnect from remote device
msg, err := client.Disconnect("192.168.1.100", 5555)

// Kill ADB server
ok, err := client.Kill()
```

#### Device Commands

```go
// Get device serial
serial, err := client.GetSerialNo("device-serial")

// Get device path
path, err := client.GetDevicePath("device-serial")

// Get device state
state, err := client.GetState("device-serial")

// Get device features
features, err := client.GetFeatures("device-serial")
// Returns: map[string]string
```

#### Port Forwarding

```go
// Forward local port to device
err := client.Forward("device-serial", "tcp:8080", "tcp:8080")

// List all forwards for a device
forwards, err := client.ListForward("device-serial")

// Remove specific forward
err := client.ForwardRemove("device-serial", "tcp:8080")

// Remove all forwards for a device
err := client.ForwardRemoveAll("device-serial")
```

### DeviceClient

For device-specific operations, use `DeviceClient`:

```go
deviceClient := client.DeviceClient("device-serial")

// Run shell command
output, err := deviceClient.RunCommand("ls /sdcard")

// Get screen orientation
orientation, err := deviceClient.GetScreenOrientation()

// Get property value
value, err := deviceClient.GetProperty("ro.product.model")

// File operations
entries, err := deviceClient.List("/sdcard/")
err = deviceClient.Pull("/sdcard/file.txt", "local.txt")
err = deviceClient.Push("local.txt", "/sdcard/file.txt")

// Logcat
logcat, err := deviceClient.Logcat()
// Returns: net.Conn that streams log output
```

## Testing

The library includes comprehensive test coverage using mock connections. See [DEVELOP.md](DEVELOP.md) for how to write tests.

```bash
# Run tests
go test ./adb/...

# Run with coverage
go test -cover ./adb/...
```

## Architecture

The library implements the ADB protocol as specified in [AOSP](https://android.googlesource.com/platform/system/core/+/master/adb/PROTOCOL.TXT):

1. **Connection Layer**: Manages TCP connections to ADB server
2. **Protocol Layer**: Handles ADB wire protocol (length-prefixed messages)
3. **Command Layer**: Implements host and device commands
4. **Transport Layer**: Manages device-specific transports

```
Your App -> Client -> ADB Protocol -> ADB Server -> Device
```

## Requirements

- Go 1.18 or later
- ADB server running (usually started by Android SDK Platform Tools)

## Troubleshooting

**Connection refused:**
- Start ADB server: `adb start-server`
- Check if ADB server is running: `adb devices`

**Device not found:**
- Check if device is connected: `adb devices`
- Verify device serial is correct

**Permission denied:**
- Make sure your user has permission to access ADB
- On Linux, you may need to configure udev rules

## Contributing

Contributions are welcome! Please see [DEVELOP.md](DEVELOP.md) for development guidelines.

## License

MIT License - see LICENSE file for details

## Credits

Based on the Node.js implementation from [openstf/adbkit](https://github.com/openstf/adbkit)
