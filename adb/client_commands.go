package adb

import (
	"context"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"time"
)

// SendHostCommandContext sends a "host:*" command with context support (timeout, cancellation)
func (c *Client) SendHostCommandContext(ctx context.Context, cmd string) ([]byte, error) {
	conn, err := c.ConnectionContext(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	if deadline, ok := ctx.Deadline(); ok {
		conn.SetDeadline(deadline)
		defer conn.SetDeadline(time.Time{})
	}

	status, err := sendADBCommand(conn, cmd)
	if err != nil {
		return nil, err
	}
	if status == StatusFail {
		payload, _ := readLengthPrefixed(conn)
		return nil, fmt.Errorf("adb server FAIL: %s", string(payload))
	}

	payload, err := readLengthPrefixed(conn)
	if err != nil {
		rest, _ := io.ReadAll(conn)
		return rest, nil
	}
	return payload, nil
}

// SendHostCommand sends a "host:*" command to the adb server and returns the payload
// Uses a 30-second timeout by default
func (c *Client) SendHostCommand(cmd string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return c.SendHostCommandContext(ctx, cmd)
}

// Version returns the adb server version as integer (as returned by host:version)
// The version format is "00040029" which means version 4.41
// Returns the integer representation: 0x00040029 = 263305
func (c *Client) Version() (int, error) {
	payload, err := c.SendHostCommand("host:version")
	if err != nil {
		return 0, err
	}
	versionStr := strings.TrimSpace(string(payload))

	version, err := parseHexInt(versionStr)
	if err != nil {
		return 0, fmt.Errorf("invalid version format %q: %w", versionStr, err)
	}

	return int(version), nil
}

// ConnectContext connects to a remote adb device with context support (timeout, cancellation)
// Accepts host:port format (e.g., "192.168.1.100:5555")
func (c *Client) ConnectContext(ctx context.Context, hostPort string) (string, error) {
	host, port, err := ParseHostPort(hostPort, 5555)
	if err != nil {
		return "", err
	}
	cmd := fmt.Sprintf("host:connect:%s:%d", host, port)
	payload, err := c.SendHostCommandContext(ctx, cmd)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(payload)), nil
}

// Connect connects to a remote adb device (host:connect:host:port)
// Accepts host:port format (e.g., "192.168.1.100:5555")
// Uses a 30-second timeout by default
func (c *Client) Connect(hostPort string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return c.ConnectContext(ctx, hostPort)
}

// DisconnectContext disconnects from a remote adb device with context support (timeout, cancellation)
// Accepts host:port format (e.g., "192.168.1.100:5555")
func (c *Client) DisconnectContext(ctx context.Context, hostPort string) (string, error) {
	host, port, err := ParseHostPort(hostPort, 5555)
	if err != nil {
		return "", err
	}
	cmd := fmt.Sprintf("host:disconnect:%s:%d", host, port)
	payload, err := c.SendHostCommandContext(ctx, cmd)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(payload)), nil
}

// Disconnect disconnects a remote adb device (host:disconnect:host:port)
// Accepts host:port format (e.g., "192.168.1.100:5555")
// Uses a 30-second timeout by default
func (c *Client) Disconnect(hostPort string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return c.DisconnectContext(ctx, hostPort)
}

// ListDevices returns the list of attached devices (host:devices)
func (c *Client) ListDevices() ([]DeviceInfo, error) {
	payload, err := c.SendHostCommand("host:devices")
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(string(payload)), "\n")
	devices := make([]DeviceInfo, 0, len(lines))
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			devices = append(devices, DeviceInfo{Serial: parts[0], State: parts[1]})
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
	if status != StatusOkay {
		payload, _ := readLengthPrefixed(conn)
		conn.Close()
		return nil, fmt.Errorf("trackDevices failed: %s", string(payload))
	}

	return conn, nil
}

// TrackDevicesWithCallback tracks devices and calls the callback for each update
func (c *Client) TrackDevicesWithCallback(callback func([]DeviceInfo)) error {
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
func parseDevicesList(line string) []DeviceInfo {
	parts := strings.Fields(line)
	devices := make([]DeviceInfo, 0)
	for i := 0; i < len(parts); i += 2 {
		if i+1 < len(parts) {
			devices = append(devices, DeviceInfo{
				Serial: parts[i],
				State:  parts[i+1],
			})
		}
	}
	return devices
}

// KillServer kills the adb server
func (c *Client) KillServer() (bool, error) {
	payload, err := c.SendHostCommand("host:kill")
	if err != nil {
		return false, err
	}
	// If we get here, the command was sent successfully
	return len(payload) == 0, nil
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

// ListForwards lists all port forwards for all devices (host:list-forward)
func (c *Client) ListForwards() ([]ForwardEntry, error) {
	payload, err := c.SendHostCommand("host:list-forward")
	if err != nil {
		return nil, err
	}
	out := strings.TrimSpace(string(payload))
	if out == "" {
		return []ForwardEntry{}, nil
	}
	lines := strings.Split(out, "\n")
	forwards := make([]ForwardEntry, 0, len(lines))
	for _, line := range lines {
		parts := strings.Fields(line)
		if len(parts) >= 3 {
			forwards = append(forwards, ForwardEntry{Serial: parts[0], Local: parts[1], Remote: parts[2]})
		}
	}
	return forwards, nil
}

// Features retrieves the ADB server features (host:features)
func (c *Client) Features() ([]string, error) {
	payload, err := c.SendHostCommand("host:features")
	if err != nil {
		return nil, err
	}

	// Features are returned as a comma-separated list
	features := strings.Split(strings.TrimSpace(string(payload)), ",")
	return features, nil
}
