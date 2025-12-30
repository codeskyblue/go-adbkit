package adb

import (
	"fmt"
	"io"
	"net"
	"strings"
)

// Forward creates a port forward on the server for the given device serial.
func (c *Client) Forward(serial, local, remote string) (bool, error) {
	addr := c.addr()
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return false, err
	}
	defer conn.Close()

	cmd := fmt.Sprintf("host-serial:%s:forward:%s;%s", serial, local, remote)
	// send command length + cmd
	if _, err := conn.Write([]byte(fmt.Sprintf("%04x%s", len(cmd), cmd))); err != nil {
		return false, err
	}

	// read first reply
	reply := make([]byte, 4)
	if _, err := io.ReadFull(conn, reply); err != nil {
		return false, err
	}
	if string(reply) == "OKAY" {
		// read second reply
		if _, err := io.ReadFull(conn, reply); err != nil {
			return false, err
		}
		if string(reply) == "OKAY" {
			return true, nil
		}
		if string(reply) == "FAIL" {
			// read error
			msg, _ := readLengthPrefixed(conn)
			return false, fmt.Errorf("forward failed: %s", string(msg))
		}
		return false, fmt.Errorf("unexpected reply: %s", string(reply))
	}
	if string(reply) == "FAIL" {
		msg, _ := readLengthPrefixed(conn)
		return false, fmt.Errorf("forward failed: %s", string(msg))
	}
	return false, fmt.Errorf("unexpected reply: %s", string(reply))
}

// ListForwards lists forwards for a device (host-serial:list-forward)
func (c *Client) ListForwards(serial string) ([]ForwardEntry, error) {
	cmd := fmt.Sprintf("host-serial:%s:list-forward", serial)
	payload, err := c.SendHostCommand(cmd)
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

// Reverse requests a reverse-forward on the device transport (reverse:forward:remote;local)
func (c *Client) Reverse(serial, remote, local string) (bool, error) {
	transport, err := c.Transport(serial)
	if err != nil {
		return false, err
	}
	defer transport.Close()

	cmd := fmt.Sprintf("reverse:forward:%s;%s", remote, local)
	if _, err := transport.Write([]byte(fmt.Sprintf("%04x%s", len(cmd), cmd))); err != nil {
		return false, err
	}

	reply := make([]byte, 4)
	if _, err := transport.Read(reply); err != nil {
		return false, err
	}
	if string(reply) == "OKAY" {
		if _, err := transport.Read(reply); err != nil {
			return false, err
		}
		if string(reply) == "OKAY" {
			return true, nil
		}
		if string(reply) == "FAIL" {
			msg, _ := readLengthPrefixed(transport)
			return false, fmt.Errorf("reverse failed: %s", string(msg))
		}
		return false, fmt.Errorf("unexpected reply: %s", string(reply))
	}
	if string(reply) == "FAIL" {
		msg, _ := readLengthPrefixed(transport)
		return false, fmt.Errorf("reverse failed: %s", string(msg))
	}
	return false, fmt.Errorf("unexpected reply: %s", string(reply))
}

// ListReverses lists reverse forwards on the device (reverse:list-forward)
func (c *Client) ListReverses(serial string) ([]ReverseEntry, error) {
	transport, err := c.Transport(serial)
	if err != nil {
		return nil, err
	}
	defer transport.Close()

	cmd := "reverse:list-forward"
	if _, err := transport.Write([]byte(fmt.Sprintf("%04x%s", len(cmd), cmd))); err != nil {
		return nil, err
	}

	// read reply
	reply := make([]byte, 4)
	if _, err := transport.Read(reply); err != nil {
		return nil, err
	}
	if string(reply) == "FAIL" {
		msg, _ := readLengthPrefixed(transport)
		return nil, fmt.Errorf("listReverses failed: %s", string(msg))
	}
	if string(reply) != "OKAY" {
		return nil, fmt.Errorf("unexpected reply: %s", string(reply))
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
