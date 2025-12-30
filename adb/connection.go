package adb

import (
	"context"
	"fmt"
	"io"
	"net"
	"strconv"
)

// Connection creates a new connection to the adb server
func (c *Client) Connection() (net.Conn, error) {
	return c.ConnectionContext(context.Background())
}

// ConnectionContext creates a new connection to the adb server with context support
func (c *Client) ConnectionContext(ctx context.Context) (net.Conn, error) {
	return c.connector.ConnectionContext(ctx)
}

// sendADBCommand writes a length-prefixed command and reads the 4-byte status (OKAY/FAIL)
func sendADBCommand(conn net.Conn, command string) (string, error) {
	length := fmt.Sprintf("%04x", len(command))
	if _, err := conn.Write([]byte(length + command)); err != nil {
		return "", err
	}

	status := make([]byte, 4)
	if _, err := conn.Read(status); err != nil {
		return "", err
	}
	return string(status), nil
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
