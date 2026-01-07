package adb

import (
	"context"
	"fmt"
	"io"
	"net"
)

// Transport switches the connection to communicate directly with the device
func (d *Device) Transport() (*Transport, error) {
	return d.TransportContext(context.Background())
}

// TransportContext switches the connection to communicate directly with the device, with context support
func (d *Device) TransportContext(ctx context.Context) (*Transport, error) {
	conn, err := d.client.ConnectionContext(ctx)
	if err != nil {
		return nil, err
	}

	cmd := fmt.Sprintf("host:%s", d.descriptor.getTransportDescriptor())
	status, err := sendADBCommand(conn, cmd)
	if err != nil {
		conn.Close()
		return nil, err
	}
	if status != StatusOkay {
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

// ReadStatus reads a 4-byte status and returns an error if it's a failure
// Returns the status string, and for failure status, also reads and returns the error message
func (t *Transport) ReadStatus() (string, error) {
	statusBuf := make([]byte, 4)
	if _, err := io.ReadFull(t, statusBuf); err != nil {
		return "", err
	}
	status := string(statusBuf)
	if status == StatusFail {
		msg, err := readLengthPrefixed(t)
		if err != nil {
			return StatusFail, fmt.Errorf("operation failed (unable to read error message)")
		}
		return StatusFail, fmt.Errorf("operation failed: %s", string(msg))
	}
	return status, nil
}

// ValidateOkayStatus reads a 4-byte status and ensures it's StatusOkay
// Returns an error if the status is not StatusOkay
func (t *Transport) ValidateOkayStatus() error {
	status, err := t.ReadStatus()
	if err != nil {
		return err
	}
	if status != StatusOkay {
		return fmt.Errorf("unexpected status: %s (expected %s)", status, StatusOkay)
	}
	return nil
}

// SendCommand sends a length-prefixed command and reads the status
// Returns the status string and error if failed
func (t *Transport) SendCommand(cmd string) (string, error) {
	if _, err := t.Write([]byte(fmt.Sprintf("%04x%s", len(cmd), cmd))); err != nil {
		return "", err
	}
	return t.ReadStatus()
}

// ExecuteSimpleCommand executes a command that expects OKAY/FAIL response
// Commonly used for commands that don't return data
func (t *Transport) ExecuteSimpleCommand(cmd string) error {
	status, err := t.SendCommand(cmd)
	if err != nil {
		return err
	}
	if status == StatusFail {
		return fmt.Errorf("command '%s' failed", cmd)
	}
	return nil
}

// ExecuteCommandWithResponse executes a command and returns the response data
// Returns data if status is OKAY, error otherwise
func (t *Transport) ExecuteCommandWithResponse(cmd string) ([]byte, error) {
	status, err := t.SendCommand(cmd)
	if err != nil {
		return nil, err
	}
	if status == StatusFail {
		return nil, fmt.Errorf("command '%s' failed", cmd)
	}
	return io.ReadAll(t)
}

// executeTransportCommand executes a simple command with automatic transport lifecycle management
func (d *Device) executeTransportCommand(cmd string) error {
	transport, err := d.Transport()
	if err != nil {
		return err
	}
	defer transport.Close()

	return transport.ExecuteSimpleCommand(cmd)
}

// executeTransportCommandWithResponse executes a command and returns response
func (d *Device) executeTransportCommandWithResponse(cmd string) ([]byte, error) {
	transport, err := d.Transport()
	if err != nil {
		return nil, err
	}
	defer transport.Close()

	return transport.ExecuteCommandWithResponse(cmd)
}

// openTransportConnection opens a transport connection for long-running operations
// The caller is responsible for closing the connection
func (d *Device) openTransportConnection(cmd string) (*Transport, error) {
	transport, err := d.Transport()
	if err != nil {
		return nil, err
	}

	if err := transport.ExecuteSimpleCommand(cmd); err != nil {
		transport.Close()
		return nil, err
	}

	return transport, nil
}
