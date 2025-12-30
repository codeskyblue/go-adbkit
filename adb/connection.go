package adb

import (
	"fmt"
	"net"
)

// Connection creates a new connection to the adb server
func (c *Client) Connection() (net.Conn, error) {
	return c.connector.Connection()
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
