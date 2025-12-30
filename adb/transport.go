package adb

import (
	"fmt"
	"io"
	"net"
	"strconv"
)

// Transport establishes a transport to the given serial and returns the live connection
func (c *Client) Transport(serial string) (net.Conn, error) {
	conn, err := c.Connection()
	if err != nil {
		return nil, err
	}

	cmd := fmt.Sprintf("host:transport:%s", serial)
	status, err := sendADBCommand(conn, cmd)
	if err != nil {
		conn.Close()
		return nil, err
	}
	if status != "OKAY" {
		// Try to read error info
		payload, _ := readLengthPrefixed(conn)
		conn.Close()
		return nil, fmt.Errorf("transport failed: %s", string(payload))
	}

	// At this point the connection is a live device transport
	return conn, nil
}

// readLengthPrefixed reads an adb length-prefixed payload (4 hex bytes + data)
func readLengthPrefixed(r io.Reader) ([]byte, error) {
	lenBuf := make([]byte, 4)
	if _, err := io.ReadFull(r, lenBuf); err != nil {
		return nil, err
	}
	// lenBuf are 4 ASCII hex digits
	l, err := parseHexInt(string(lenBuf))
	if err != nil {
		return nil, err
	}
	if l == 0 {
		return []byte{}, nil
	}
	data := make([]byte, l)
	if _, err := io.ReadFull(r, data); err != nil {
		return nil, err
	}
	return data, nil
}

// parseHexInt parses a 4-digit hex string to integer
func parseHexInt(s string) (int64, error) {
	result, err := strconv.ParseInt(s, 16, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid hex string %q: %w", s, err)
	}
	return result, nil
}
