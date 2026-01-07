package adb

import (
	"fmt"
	"strings"
)

// Forward creates a port forward on the server for this device.
func (d *Device) Forward(local, remote string) (bool, error) {
	conn, err := d.client.Connection()
	if err != nil {
		return false, err
	}
	defer conn.Close()

	cmd := fmt.Sprintf("%s:forward:%s;%s", d.descriptor.getHostPrefix(), local, remote)
	// send command length + cmd
	if _, err := conn.Write([]byte(fmt.Sprintf("%04x%s", len(cmd), cmd))); err != nil {
		return false, err
	}

	transport := NewTransport(conn)
	// read first reply
	status, err := transport.CheckStatus()
	if err != nil {
		return false, err
	}
	if status == StatusOkay {
		// read second reply
		status, err := transport.CheckStatus()
		if err != nil {
			return false, err
		}
		if status == StatusOkay {
			return true, nil
		}
		return false, fmt.Errorf("unexpected status: %s", status)
	}
	return false, fmt.Errorf("unexpected status: %s", status)
}

// ForwardRemove removes a port forward on the server for this device.
func (d *Device) ForwardRemove(local string) (bool, error) {
	conn, err := d.client.Connection()
	if err != nil {
		return false, err
	}
	defer conn.Close()

	cmd := fmt.Sprintf("%s:killforward:%s", d.descriptor.getHostPrefix(), local)
	if _, err := conn.Write([]byte(fmt.Sprintf("%04x%s", len(cmd), cmd))); err != nil {
		return false, err
	}

	transport := NewTransport(conn)
	status, err := transport.CheckStatus()
	if err != nil {
		return false, err
	}
	if status == StatusOkay {
		status, err := transport.CheckStatus()
		if err != nil {
			return false, err
		}
		if status == StatusOkay {
			return true, nil
		}
		return false, fmt.Errorf("unexpected status: %s", status)
	}
	return false, fmt.Errorf("unexpected status: %s", status)
}

// Reverse requests a reverse-forward on the device transport (reverse:forward:remote;local)
func (d *Device) Reverse(remote, local string) (bool, error) {
	transport, err := d.Transport()
	if err != nil {
		return false, err
	}
	defer transport.Close()

	cmd := fmt.Sprintf("reverse:forward:%s;%s", remote, local)
	status, err := transport.SendCommand(cmd)
	if err != nil {
		return false, err
	}
	if status != StatusOkay {
		return false, fmt.Errorf("unexpected status: %s", status)
	}

	// Read second status
	status, err = transport.CheckStatus()
	if err != nil {
		return false, err
	}
	if status != StatusOkay {
		return false, fmt.Errorf("unexpected status: %s", status)
	}
	return true, nil
}

// ReverseRemove removes a reverse-forward on the device transport (reverse:killforward:remote)
func (d *Device) ReverseRemove(remote string) (bool, error) {
	transport, err := d.Transport()
	if err != nil {
		return false, err
	}
	defer transport.Close()

	cmd := fmt.Sprintf("reverse:killforward:%s", remote)
	if _, err := transport.Write([]byte(fmt.Sprintf("%04x%s", len(cmd), cmd))); err != nil {
		return false, err
	}

	status, err := transport.CheckStatus()
	if err != nil {
		return false, err
	}
	if status == StatusOkay {
		status, err := transport.CheckStatus()
		if err != nil {
			return false, err
		}
		if status == StatusOkay {
			return true, nil
		}
		return false, fmt.Errorf("unexpected status: %s", status)
	}
	return false, fmt.Errorf("unexpected status: %s", status)
}

// ListReverses lists reverse forwards on this device (reverse:list-forward)
func (d *Device) ListReverses() ([]ReverseEntry, error) {
	transport, err := d.Transport()
	if err != nil {
		return nil, err
	}
	defer transport.Close()

	status, err := transport.SendCommand("reverse:list-forward")
	if err != nil {
		return nil, fmt.Errorf("listReverses failed: %w", err)
	}
	if status != StatusOkay {
		return nil, fmt.Errorf("unexpected status: %s", status)
	}

	payload, err := readLengthPrefixed(transport)
	if err != nil {
		return nil, err
	}
	out := strings.TrimSpace(string(payload))
	if out == "" {
		return []ReverseEntry{}, nil
	}
	lines := strings.Split(out, "\n")
	revs := make([]ReverseEntry, 0, len(lines))
	for _, line := range lines {
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			revs = append(revs, ReverseEntry{Remote: parts[0], Local: parts[1]})
		}
	}
	return revs, nil
}
