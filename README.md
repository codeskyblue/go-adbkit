# go-adbkit

[![Go Reference](https://pkg.go.dev/badge/github.com/codeskyblue/go-adbkit.svg)](https://pkg.go.dev/github.com/codeskyblue/go-adbkit)
[![codecov](https://codecov.io/gh/codeskyblue/go-adbkit/graph/badge.svg?token=YkCKmO9TUs)](https://codecov.io/gh/codeskyblue/go-adbkit)

A pure Go client library for the Android Debug Bridge (ADB) protocol.

## Features

- **Pure Go Implementation**: No external dependencies on ADB binaries
- **Comprehensive API Support**:
  - Host commands: version, devices, connect, disconnect, kill server
  - Device commands: shell, properties, reboot, screenshot, screen orientation
  - Package management: list, install, uninstall, clear data, check installation
  - Intent management: start activities, start services
  - Network operations: TCP/IP mode, USB mode, socket connections
  - File operations: stat, readdir, pull, push (sync service)
  - Port forwarding: forward, reverse, list forwards
  - Screen capture: framebuffer, screencap with fallback
  - Logcat: real-time log streaming
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
// Get a device
device := client.Device("device-serial")

// Run a shell command
output, err := device.RunCommand("getprop ro.product.model")
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Device model: %s\n", output)

// Run with context and timeout
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

output, err = device.RunCommandContext(ctx, "getprop ro.product.model")
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Device model: %s\n", output)
```

### File Operations

```go
device := client.Device("device-serial")

// List files
entries, err := device.Readdir("/sdcard/")
if err != nil {
    log.Fatal(err)
}
for _, entry := range entries {
    fmt.Printf("File: %s\n", entry)
}

// Get file info
stat, err := device.Stat("/sdcard/file.txt")
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Size: %d, Mode: %d\n", stat.Size, stat.Mode)

// Pull file from device
reader, err := device.Pull("/sdcard/file.txt")
if err != nil {
    log.Fatal(err)
}
defer reader.Close()

// Write to local file
localFile, err := os.Create("local_file.txt")
if err != nil {
    log.Fatal(err)
}
defer localFile.Close()
_, err = io.Copy(localFile, reader)

// Push file to device
file, err := os.Open("local_file.txt")
if err != nil {
    log.Fatal(err)
}
defer file.Close()

err = device.Push(file, "/sdcard/file.txt", 0644)
if err != nil {
    log.Fatal(err)
}
```

### Port Forwarding

```go
device := client.Device("device-serial")

// Forward local port to device
success, err := device.Forward("tcp:8080", "tcp:8080")
if err != nil {
    log.Fatal(err)
}

// Reverse forward (device to local)
success, err = device.Reverse("tcp:8080", "tcp:8080")
if err != nil {
    log.Fatal(err)
}

// List reverse forwards
revs, err := device.ListReverses()
if err != nil {
    log.Fatal(err)
}
for _, rev := range revs {
    fmt.Printf("%s -> %s\n", rev.Remote, rev.Local)
}

// List all forwards (from client)
forwards, err := client.ListForwards()
if err != nil {
    log.Fatal(err)
}
for _, fwd := range forwards {
    fmt.Printf("%s: %s -> %s\n", fwd.Serial, fwd.Local, fwd.Remote)
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

### Package Management

```go
// Using Device methods
device := client.Device("device-serial")

// List installed packages
packages, err := device.GetPackages()
if err != nil {
    log.Fatal(err)
}
for _, pkg := range packages {
    fmt.Printf("Package: %s\n", pkg)
}

// Check if package is installed
installed, err := device.IsInstalled("com.example.app")
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Installed: %v\n", installed)

// Clear app data
success, err := device.ClearPackageData("com.example.app")
if err != nil {
    log.Fatal(err)
}

// Install APK
success, err = device.Install("/path/to/app.apk")
if err != nil {
    log.Fatal(err)
}

// Uninstall package
success, err = device.Uninstall("com.example.app")
if err != nil {
    log.Fatal(err)
}
```

### Intent Management

```go
device := client.Device("device-serial")

// Start an activity
err := device.StartActivity(adb.StartActivityOptions{
    Action:    "android.intent.action.VIEW",
    Data:      "https://example.com",
    Component: "com.example.app/.MainActivity",
})
if err != nil {
    log.Fatal(err)
}

// Start a service
err = device.StartService(adb.StartServiceOptions{
    Action:    "com.example.app.ACTION_SYNC",
    Component: "com.example.app/.SyncService",
    Extras: map[string]string{
        "param1": "value1",
        "param2": "value2",
    },
})
if err != nil {
    log.Fatal(err)
}
```

### Network Operations

```go
device := client.Device("device-serial")

// Switch to TCP/IP mode
success, err := device.TcpIp(5555)
if err != nil {
    log.Fatal(err)
}

// Connect to device over TCP
msg, err := client.Connect("192.168.1.100:5555")
if err != nil {
    log.Fatal(err)
}

// Switch back to USB mode
success, err = device.Usb()
if err != nil {
    log.Fatal(err)
}

// Open socket connection
conn, err := device.OpenSocket("tcp:8080")
if err != nil {
    log.Fatal(err)
}
defer conn.Close()

// Open local abstract socket (e.g., for scrcpy)
conn, err = device.OpenSocket("localabstract:scrcpy")
if err != nil {
    log.Fatal(err)
}
defer conn.Close()
```

### Screen Capture

```go
device := client.Device("device-serial")

// Capture screen using screencap (faster, recommended)
img, err := device.Screencap()
if err != nil {
    log.Fatal(err)
}

// Save screenshot to file
f, err := os.Create("screenshot.png")
if err != nil {
    log.Fatal(err)
}
defer f.Close()
err = png.Encode(f, img)

// Or use framebuffer (slower, works on older devices)
img, err = device.Framebuffer()
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
msg, err := client.Connect("192.168.1.100:5555")

// Disconnect from remote device
msg, err := client.Disconnect("192.168.1.100:5555")

// Kill ADB server
ok, err := client.KillServer()
```

#### Device Commands

```go
// Get a device
device := client.Device("device-serial")

// Get device information
serial, err := device.Serial()
state, err := device.State()
props, err := device.Properties()
path, err := device.DevicePath()

// Run shell command (simple version without context)
output, err := device.RunCommand("ls /sdcard")

// Run shell command with context support
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()
output, err = device.RunCommandContext(ctx, "ls /sdcard")
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Output: %s\n", output)
```

#### Port Forwarding

```go
device := client.Device("device-serial")

// Forward local port to device
success, err := device.Forward("tcp:8080", "tcp:8080")

// Reverse forward (device to local)
success, err = device.Reverse("tcp:8080", "tcp:8080")

// List reverse forwards for a device
revs, err := device.ListReverses()

// List all forwards (from client)
forwards, err := client.ListForwards()
```

#### Screen Capture

```go
device := client.Device("device-serial")

// Get framebuffer as image.Image
img, err := device.Framebuffer()
if err != nil {
    log.Fatal(err)
}
// img can be used directly with standard image operations
// For example, save to file:
// f, _ := os.Create("screenshot.png")
// defer f.Close()
// png.Encode(f, img)

// Screencap (returns image.Image)
img, err = device.Screencap()
if err != nil {
    log.Fatal(err)
}
```

### Device

For device-specific operations, use `Device`:

```go
device := client.Device("device-serial")

// Device Information
serial, err := device.Serial()
state, err := device.State()
props, err := device.Properties()
path, err := device.DevicePath()

// Device Control
err := device.Reboot()
err = device.Remount()
ok, err := device.Root()

// Shell Commands
conn, err := device.Shell("ls /sdcard")
defer conn.Close()
data, err := adb.ReadAllFromConn(conn)

// Or run command and get output directly
output, err := device.RunCommand("ls /sdcard")

// File Operations
stat, err := device.Stat("/sdcard/file.txt")
entries, err := device.Readdir("/sdcard/")
reader, err := device.Pull("/sdcard/file.txt")
// Push requires io.Reader and file mode
file, _ := os.Open("local.txt")
err = device.Push(file, "/sdcard/file.txt", 0644)

// Screen Operations
img, err := device.Screencap()
img, err = device.Framebuffer()

// Logcat
conn, err = device.OpenLogcat("main")
defer conn.Close()
// Stream logs from conn

// Package Management
packages, err := device.GetPackages()
installed, err := device.IsInstalled("com.example.app")
success, err := device.ClearPackageData("com.example.app")
success, err = device.Install("/path/to/app.apk")
success, err = device.InstallRemote("/sdcard/app.apk")
success, err = device.Uninstall("com.example.app")

// Intent Management
err = device.StartActivity(adb.StartActivityOptions{
    Action:    "android.intent.action.VIEW",
    Component: "com.example.app/.MainActivity",
})
err = device.StartService(adb.StartServiceOptions{
    Component: "com.example.app/.MyService",
})

// Network Operations
success, err := device.TcpIp(5555)
success, err = device.Usb()
conn, err = device.OpenSocket("tcp:8080")
conn, err = device.OpenLocal("/dev/socket/adb")
conn, err = device.OpenLog("main")
conn, err = device.OpenSocketContext(ctx, "localabstract:scrcpy")
```

### Sync Service

For advanced file operations, use the Sync service:

```go
device := client.Device("device-serial")

// Create a new sync service
sync, err := device.NewSyncService()
if err != nil {
    log.Fatal(err)
}
defer sync.Close()

// List files
entries, err := sync.List("/sdcard/")
if err != nil {
    log.Fatal(err)
}
for _, entry := range entries {
    fmt.Printf("%s %d %s\n", entry.Name, entry.Size, entry.Mode)
}

// Pull file with progress
reader, err := sync.Pull("/sdcard/file.txt")
if err != nil {
    log.Fatal(err)
}
defer reader.Close()

// Push file with options
file, _ := os.Open("local.txt")
defer file.Close()

err = sync.Push(file, "/sdcard/file.txt", adb.SyncPushOptions{
    Mode: 0644,
})
if err != nil {
    log.Fatal(err)
}

// Get file stat
stat, err := sync.Stat("/sdcard/file.txt")
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Size: %d, Mode: %d\n", stat.Size, stat.Mode)
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

- Go 1.22 or later
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
