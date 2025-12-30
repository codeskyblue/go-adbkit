# ADB Client - Go Implementation

This is a Go implementation of an ADB (Android Debug Bridge) client, translated from the nodejs-adbkit CoffeeScript implementation.

## File Structure

The codebase has been organized into modular files for better maintainability:

### Core Files

- **client.go** - Core client definitions and types
  - `Client` struct
  - `ClientOptions` struct
  - Device-related types (`Device`, `DeviceWithPath`, `ForwardEntry`, `ReverseEntry`)
  - Client constructors (`NewClient`, `NewClientWithOptions`)

- **connection.go** - Connection management
  - `Connection()` - Creates connection to ADB server
  - `sendADBCommand()` - Sends length-prefixed commands
  - `addr()` - Returns server address

- **transport.go** - Device transport layer
  - `Transport()` - Establishes device transport
  - `readLengthPrefixed()` - Reads ADB protocol payloads
  - `parseHexInt()` - Parses hex integers

### Command Modules

- **host_commands.go** - Host-level commands
  - `SendHostCommand()` - Generic host command sender
  - `Version()` - Get ADB server version
  - `Connect()` / `Disconnect()` - TCP/IP device connections
  - `ListDevices()` / `ListDevicesWithPaths()` - List attached devices
  - `TrackDevices()` / `TrackDevicesWithCallback()` - Monitor device changes
  - `Kill()` - Kill ADB server
  - `ConnectWithHostPort()` / `DisconnectWithHostPort()` - Helper for host:port format

- **device_commands.go** - Device-specific commands
  - `GetSerialNo()` - Get device serial number
  - `GetDevicePath()` - Get device path
  - `GetState()` - Get device state
  - `GetFeatures()` - Get device features
  - `GetProperties()` - Get system properties
  - `GetDHCPIpAddress()` - Get DHCP IP address
  - `Shell()` - Execute shell commands
  - `Reboot()` - Reboot device
  - `Remount()` - Remount filesystem
  - `Root()` - Restart adbd as root
  - `WaitForDevice()` / `WaitForDeviceWithTimeout()` - Wait for device
  - `WaitBootComplete()` - Wait for boot completion
  - `TrackJdwp()` - Track JDWP processes
  - `Framebuffer()` - Capture framebuffer
  - `Screencap()` / `ScreencapWithFallback()` - Screen capture
  - `OpenLogcat()` - Open logcat stream

- **package_commands.go** - Application management
  - `GetPackages()` - List installed packages
  - `Clear()` - Clear app data
  - `Install()` / `InstallRemote()` - Install APK
  - `Uninstall()` - Uninstall package
  - `IsInstalled()` - Check if package is installed
  - `StartActivity()` - Start an activity
  - `StartService()` - Start a service
  - `StartActivityOptions` / `StartServiceOptions` - Option structs

- **forward.go** - Port forwarding
  - `Forward()` - Create port forward
  - `ListForwards()` - List port forwards
  - `Reverse()` - Create reverse port forward
  - `ListReverses()` - List reverse forwards

- **device_network.go** - Network operations
  - `TcpIp()` - Switch to TCP/IP mode
  - `Usb()` - Switch back to USB mode
  - `OpenLocal()` - Open local socket
  - `OpenLog()` - Open log buffer
  - `OpenSocket()` - Open socket connection (supports tcp:, local:, localabstract:, etc.)

- **file_operations.go** - File operations
  - `Stat()` - Get file information
  - `Readdir()` - List directory
  - `Pull()` - Pull file from device
  - `Push()` - Push file to device
  - `syncService()` - Establish sync service

## Usage Example

```go
package main

import (
    "fmt"
    "log"
    "github.com/codeskyblue/go-adbkit/adb"
)

func main() {
    // Create a new client
    client := adb.NewClient("", 0) // Use default host:port

    // List devices
    devices, err := client.ListDevices()
    if err != nil {
        log.Fatal(err)
    }

    for _, device := range devices {
        fmt.Printf("Device: %s (%s)\n", device.Serial, device.State)

        // Get properties
        props, err := client.GetProperties(device.Serial)
        if err != nil {
            log.Printf("Error getting properties: %v", err)
            continue
        }

        fmt.Printf("Model: %s\n", props["ro.product.model"])
    }
}
```

## Design Principles

1. **Modularity** - Code is split into logical modules based on functionality
2. **Simplicity** - Each file has a single, clear responsibility
3. **Go Idioms** - Uses Go conventions (errors, interfaces, etc.)
4. **Compatibility** - Maintains API compatibility with the CoffeeScript version

## Translation Notes

This implementation is translated from the CoffeeScript version in nodejs-adbkit. Key differences:

- Uses Go's error handling instead of promises
- Direct TCP connections instead of Node streams
- Go structs instead of JavaScript objects
- Synchronous API (use goroutines for concurrent operations)
