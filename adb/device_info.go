package adb

import (
	"fmt"
	"io"
	"net"
	"strings"
	"time"
)

// Serial returns the device's serial number
func (d *Device) Serial() (string, error) {
	attr, err := d.getAttribute("get-serialno")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(attr), nil
}

// DevicePath returns the device path
func (d *Device) DevicePath() (string, error) {
	attr, err := d.getAttribute("get-devpath")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(attr), nil
}

// State returns the device state
func (d *Device) State() (string, error) {
	attr, err := d.getAttribute("get-state")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(attr), nil
}

// Features retrieves device features
func (d *Device) Features() (map[string]string, error) {
	transport, err := d.Transport()
	if err != nil {
		return nil, err
	}
	defer transport.Close()

	cmd := "features:"
	if _, err := transport.Write([]byte(fmt.Sprintf("%04x%s", len(cmd), cmd))); err != nil {
		return nil, err
	}

	reply := make([]byte, 4)
	if _, err := transport.Read(reply); err != nil {
		return nil, err
	}
	if string(reply) == "FAIL" {
		msg, _ := readLengthPrefixed(transport)
		return nil, fmt.Errorf("getFeatures failed: %s", string(msg))
	}
	if string(reply) != "OKAY" {
		return nil, fmt.Errorf("unexpected reply: %s", string(reply))
	}

	payload, err := readLengthPrefixed(transport)
	if err != nil {
		return nil, err
	}

	features := make(map[string]string)
	lines := strings.Split(strings.TrimSpace(string(payload)), "\n")
	for _, line := range lines {
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			features[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}
	return features, nil
}

// Properties returns device properties by running `getprop`
func (d *Device) Properties() (map[string]string, error) {
	conn, err := d.Shell("getprop")
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	data, err := ReadAllFromConn(conn)
	if err != nil {
		return nil, err
	}
	props := make(map[string]string)
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if !strings.HasPrefix(line, "[") {
			continue
		}
		endKey := strings.Index(line, "]:")
		if endKey == -1 {
			continue
		}
		key := strings.Trim(line[1:endKey], " ")
		rest := strings.TrimSpace(line[endKey+2:])
		if strings.HasPrefix(rest, "[") && strings.HasSuffix(rest, "]") {
			rest = rest[1 : len(rest)-1]
		}
		props[key] = rest
	}
	return props, nil
}

// DHCPIpAddress returns the DHCP IP address for the given interface (defaults to wlan0)
func (d *Device) DHCPIpAddress(iface string) (string, error) {
	if iface == "" {
		iface = "wlan0"
	}
	props, err := d.Properties()
	if err != nil {
		return "", err
	}
	key := fmt.Sprintf("dhcp.%s.ipaddress", iface)
	if ip, ok := props[key]; ok && ip != "" {
		return ip, nil
	}
	return "", fmt.Errorf("unable to find ipaddress for '%s'", iface)
}

// Reboot reboots the device
func (d *Device) Reboot() (bool, error) {
	transport, err := d.Transport()
	if err != nil {
		return false, err
	}
	defer transport.Close()

	cmd := "reboot:"
	if _, err := transport.Write([]byte(fmt.Sprintf("%04x%s", len(cmd), cmd))); err != nil {
		return false, err
	}

	reply := make([]byte, 4)
	if _, err := transport.Read(reply); err != nil {
		return false, err
	}
	if string(reply) == "OKAY" {
		io.ReadAll(transport)
		return true, nil
	}
	if string(reply) == "FAIL" {
		msg, _ := readLengthPrefixed(transport)
		return false, fmt.Errorf("reboot failed: %s", string(msg))
	}
	return false, fmt.Errorf("unexpected reply: %s", string(reply))
}

// Remount remounts the device filesystem as read-write
func (d *Device) Remount() (bool, error) {
	transport, err := d.Transport()
	if err != nil {
		return false, err
	}
	defer transport.Close()

	cmd := "remount:"
	if _, err := transport.Write([]byte(fmt.Sprintf("%04x%s", len(cmd), cmd))); err != nil {
		return false, err
	}
	reply := make([]byte, 4)
	if _, err := transport.Read(reply); err != nil {
		return false, err
	}
	if string(reply) == "OKAY" {
		return true, nil
	}
	if string(reply) == "FAIL" {
		msg, _ := readLengthPrefixed(transport)
		return false, fmt.Errorf("remount failed: %s", string(msg))
	}
	return false, fmt.Errorf("unexpected reply: %s", string(reply))
}

// Root attempts to restart adbd as root
func (d *Device) Root() (bool, error) {
	transport, err := d.Transport()
	if err != nil {
		return false, err
	}
	defer transport.Close()

	cmd := "root:"
	if _, err := transport.Write([]byte(fmt.Sprintf("%04x%s", len(cmd), cmd))); err != nil {
		return false, err
	}
	reply := make([]byte, 4)
	if _, err := transport.Read(reply); err != nil {
		return false, err
	}
	if string(reply) == "OKAY" {
		data, _ := io.ReadAll(transport)
		out := string(data)
		if strings.Contains(out, "restarting adbd as root") {
			return true, nil
		}
		return false, fmt.Errorf("%s", strings.TrimSpace(out))
	}
	if string(reply) == "FAIL" {
		msg, _ := readLengthPrefixed(transport)
		return false, fmt.Errorf("root failed: %s", string(msg))
	}
	return false, fmt.Errorf("unexpected reply: %s", string(reply))
}

// WaitForDevice waits for the device to be available
func (d *Device) WaitForDevice() (bool, error) {
	cmd := fmt.Sprintf("%s:wait-for-device", d.descriptor.getHostPrefix())
	payload, err := d.client.SendHostCommand(cmd)
	if err != nil {
		return false, err
	}
	return len(payload) == 0, nil
}

// WaitForDeviceWithTimeout waits for a device with a timeout
func (d *Device) WaitForDeviceWithTimeout(timeout time.Duration) error {
	timeoutChan := time.After(timeout)
	errChan := make(chan error, 1)

	go func() {
		_, err := d.WaitForDevice()
		errChan <- err
	}()

	select {
	case err := <-errChan:
		return err
	case <-timeoutChan:
		return fmt.Errorf("wait for device timed out")
	}
}

// WaitBootComplete waits for the device to finish booting
func (d *Device) WaitBootComplete() (bool, error) {
	transport, err := d.Transport()
	if err != nil {
		return false, err
	}
	defer transport.Close()

	cmd := "shell:getprop sys.boot_completed"
	if _, err := transport.Write([]byte(fmt.Sprintf("%04x%s", len(cmd), cmd))); err != nil {
		return false, err
	}

	reply := make([]byte, 4)
	if _, err := transport.Read(reply); err != nil {
		return false, err
	}
	if string(reply) == "OKAY" {
		data, _ := io.ReadAll(transport)
		return strings.Contains(string(data), "1"), nil
	}
	if string(reply) == "FAIL" {
		msg, _ := readLengthPrefixed(transport)
		return false, fmt.Errorf("waitBootComplete failed: %s", string(msg))
	}
	return false, fmt.Errorf("unexpected reply: %s", string(reply))
}

// TrackJdwp starts tracking jdwp pids
func (d *Device) TrackJdwp() (net.Conn, error) {
	transport, err := d.Transport()
	if err != nil {
		return nil, err
	}
	cmd := "track-jdwp"
	if _, err := transport.Write([]byte(fmt.Sprintf("%04x%s", len(cmd), cmd))); err != nil {
		transport.Close()
		return nil, err
	}
	reply := make([]byte, 4)
	if _, err := transport.Read(reply); err != nil {
		transport.Close()
		return nil, err
	}
	if string(reply) == "OKAY" {
		return transport, nil
	}
	if string(reply) == "FAIL" {
		msg, _ := readLengthPrefixed(transport)
		transport.Close()
		return nil, fmt.Errorf("track-jdwp failed: %s", string(msg))
	}
	transport.Close()
	return nil, fmt.Errorf("unexpected reply: %s", string(reply))
}

// getAttribute returns the first message returned by the server
func (d *Device) getAttribute(attr string) (string, error) {
	cmd := fmt.Sprintf("%s:%s", d.descriptor.getHostPrefix(), attr)
	payload, err := d.client.SendHostCommand(cmd)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(payload)), nil
}
