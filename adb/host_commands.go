package adb

import (
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
)

// SendHostCommand sends a "host:*" command to the adb server and returns the payload
func (c *Client) SendHostCommand(cmd string) ([]byte, error) {
	conn, err := net.Dial("tcp", c.addr())
	if err != nil {
		return nil, err
	}
	// keep connection open until we read the payload
	defer conn.Close()

	status, err := sendADBCommand(conn, cmd)
	if err != nil {
		return nil, err
	}
	if status == "FAIL" {
		// read error payload
		payload, _ := readLengthPrefixed(conn)
		return nil, fmt.Errorf("adb server FAIL: %s", string(payload))
	}

	// OKAY — read length-prefixed payload(s). Most host commands send a single length-prefixed block.
	payload, err := readLengthPrefixed(conn)
	if err != nil {
		// If there's no length, try to read raw remainder
		rest, _ := io.ReadAll(conn)
		return rest, nil
	}
	return payload, nil
}

// Version returns the adb server version string (as returned by host:version)
func (c *Client) Version() (string, error) {
	payload, err := c.SendHostCommand("host:version")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(payload)), nil
}

// Connect connects to a remote adb device (host:connect:host:port)
func (c *Client) Connect(host string, port int) (string, error) {
	cmd := fmt.Sprintf("host:connect:%s:%d", host, port)
	payload, err := c.SendHostCommand(cmd)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(payload)), nil
}

// Disconnect disconnects a remote adb device (host:disconnect:host:port)
func (c *Client) Disconnect(host string, port int) (string, error) {
	cmd := fmt.Sprintf("host:disconnect:%s:%d", host, port)
	payload, err := c.SendHostCommand(cmd)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(payload)), nil
}

// ListDevices returns the list of attached devices (host:devices)
func (c *Client) ListDevices() ([]Device, error) {
	payload, err := c.SendHostCommand("host:devices")
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(string(payload)), "\n")
	devices := make([]Device, 0, len(lines))
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			devices = append(devices, Device{Serial: parts[0], State: parts[1]})
		}
	}
	return devices, nil
}

// ListDevicesWithPaths returns the list of devices with their paths (host:devices-l)
func (c *Client) ListDevicesWithPaths() ([]DeviceWithPath, error) {
	payload, err := c.SendHostCommand("host:devices-l")
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(string(payload)), "\n")
	devices := make([]DeviceWithPath, 0, len(lines))
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			device := DeviceWithPath{
				Serial: parts[0],
				State:  parts[1],
			}
			if len(parts) >= 4 {
				// Format: serial state model device:path
				device.Model = parts[2]
				device.Device = parts[3]
			}
			devices = append(devices, device)
		}
	}
	return devices, nil
}

// TrackDevices starts tracking devices (returns a connection that streams device updates)
func (c *Client) TrackDevices() (net.Conn, error) {
	conn, err := c.Connection()
	if err != nil {
		return nil, err
	}

	cmd := "host:track-devices"
	status, err := sendADBCommand(conn, cmd)
	if err != nil {
		conn.Close()
		return nil, err
	}
	if status != "OKAY" {
		payload, _ := readLengthPrefixed(conn)
		conn.Close()
		return nil, fmt.Errorf("trackDevices failed: %s", string(payload))
	}

	return conn, nil
}

// TrackDevicesWithCallback tracks devices and calls the callback for each update
func (c *Client) TrackDevicesWithCallback(callback func([]Device)) error {
	conn, err := c.TrackDevices()
	if err != nil {
		return err
	}
	defer conn.Close()

	scanner := newScanner(conn)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		devices := parseDevicesList(line)
		callback(devices)
	}
	return scanner.Err()
}

// parseDevicesList parses a device list line
func parseDevicesList(line string) []Device {
	parts := strings.Fields(line)
	devices := make([]Device, 0)
	for i := 0; i < len(parts); i += 2 {
		if i+1 < len(parts) {
			devices = append(devices, Device{
				Serial: parts[i],
				State:  parts[i+1],
			})
		}
	}
	return devices
}

// Kill kills the adb server
func (c *Client) Kill() (bool, error) {
	payload, err := c.SendHostCommand("host:kill")
	if err != nil {
		return false, err
	}
	// If we get here, the command was sent successfully
	return len(payload) == 0, nil
}

// ConnectWithHostPort connects with host:port format support
func (c *Client) ConnectWithHostPort(hostPort string) (string, error) {
	host, port, err := ParseHostPort(hostPort, 5555)
	if err != nil {
		return "", err
	}
	return c.Connect(host, port)
}

// DisconnectWithHostPort disconnects with host:port format support
func (c *Client) DisconnectWithHostPort(hostPort string) (string, error) {
	host, port, err := ParseHostPort(hostPort, 5555)
	if err != nil {
		return "", err
	}
	return c.Disconnect(host, port)
}

// ParseHostPort parses a host:port string
func ParseHostPort(hostPort string, defaultPort int) (string, int, error) {
	if strings.Contains(hostPort, ":") {
		parts := strings.SplitN(hostPort, ":", 2)
		port, err := strconv.Atoi(parts[1])
		if err != nil {
			return "", 0, err
		}
		return parts[0], port, nil
	}
	return hostPort, defaultPort, nil
}
