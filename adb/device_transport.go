package adb

import (
	"fmt"
	"io"
	"net"
)

// Transport switches the connection to communicate directly with the device
func (d *Device) Transport() (*Transport, error) {
	conn, err := d.client.Connection()
	if err != nil {
		return nil, err
	}

	cmd := fmt.Sprintf("host:%s", d.descriptor.getTransportDescriptor())
	status, err := sendADBCommand(conn, cmd)
	if err != nil {
		conn.Close()
		return nil, err
	}
	if status != StatusSuccess {
		payload, _ := readLengthPrefixed(conn)
		conn.Close()
		return nil, fmt.Errorf("device selection failed: %s", string(payload))
	}

	return NewTransport(conn), nil
}

// Transport wraps a net.Conn and provides ADB protocol-specific methods
type Transport struct {
	net.Conn
}

// NewTransport creates a new Transport from a net.Conn
func NewTransport(conn net.Conn) *Transport {
	return &Transport{Conn: conn}
}

// Read implements io.Reader
func (t *Transport) Read(p []byte) (n int, err error) {
	return t.Conn.Read(p)
}

// CheckStatus reads a 4-byte status and returns an error if it's a failure
// Returns the status string, and for failure status, also reads and returns the error message
func (t *Transport) CheckStatus() (string, error) {
	statusBuf := make([]byte, 4)
	if _, err := io.ReadFull(t, statusBuf); err != nil {
		return "", err
	}
	status := string(statusBuf)
	if status == StatusFailure {
		msg, err := readLengthPrefixed(t)
		if err != nil {
			return StatusFailure, fmt.Errorf("operation failed (unable to read error message)")
		}
		return StatusFailure, fmt.Errorf("operation failed: %s", string(msg))
	}
	return status, nil
}

// CheckStatusSuccess reads a 4-byte status and ensures it's StatusSuccess
// Returns an error if the status is not StatusSuccess
func (t *Transport) CheckStatusSuccess() error {
	status, err := t.CheckStatus()
	if err != nil {
		return err
	}
	if status != StatusSuccess {
		return fmt.Errorf("unexpected status: %s (expected %s)", status, StatusSuccess)
	}
	return nil
}
