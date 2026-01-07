package adb

import (
	"fmt"
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


// parseGetpropOutput parses the output from getprop command into a map
func parseGetpropOutput(output string) map[string]string {
	props := make(map[string]string)
	lines := strings.Split(output, "\n")
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
	return props
}

// Properties returns device properties by running `getprop`
func (d *Device) Properties() (map[string]string, error) {
	output, err := d.RunCommand("getprop")
	if err != nil {
		return nil, err
	}
	return parseGetpropOutput(output), nil
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
	err := d.ExecuteTransportCommand("reboot:")
	return err == nil, err
}

// Remount remounts the device filesystem as read-write
func (d *Device) Remount() (bool, error) {
	err := d.ExecuteTransportCommand("remount:")
	return err == nil, err
}

// Root attempts to restart adbd as root
func (d *Device) Root() (bool, error) {
	data, err := d.ExecuteTransportCommandWithResponse("root:")
	if err != nil {
		return false, err
	}
	out := string(data)
	if strings.Contains(out, "restarting adbd as root") {
		return true, nil
	}
	return false, fmt.Errorf("%s", strings.TrimSpace(out))
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
	data, err := d.ExecuteTransportCommandWithResponse("shell:getprop sys.boot_completed")
	if err != nil {
		return false, err
	}
	return strings.Contains(string(data), "1"), nil
}

// TrackJdwp starts tracking jdwp pids
func (d *Device) TrackJdwp() (net.Conn, error) {
	return d.OpenTransportConnection("track-jdwp")
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
