package adb

import (
	"fmt"
	"io"
	"net"
)

// Transport establishes a transport to the given serial and returns the live connection
func (c *Client) Transport(serial string) (net.Conn, error) {
	conn, err := net.Dial("tcp", c.addr())
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
	var result int64
	for _, c := range s {
		var v int64
		switch {
		case c >= '0' && c <= '9':
			v = int64(c - '0')
		case c >= 'a' && c <= 'f':
			v = int64(c - 'a' + 10)
		case c >= 'A' && c <= 'F':
			v = int64(c - 'A' + 10)
		default:
			return 0, fmt.Errorf("invalid hex digit: %c", c)
		}
		result = result*16 + v
	}
	return result, nil
}
