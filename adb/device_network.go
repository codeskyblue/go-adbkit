package adb

import (
	"context"
	"fmt"
	"io"
	"net"
	"strings"
)

// TcpIp switches the device to TCP/IP mode
func (d *Device) TcpIp(port int) (bool, error) {
	if port == 0 {
		port = 5555
	}
	transport, err := d.Transport()
	if err != nil {
		return false, err
	}
	defer transport.Close()

	cmd := fmt.Sprintf("tcpip:%d", port)
	if _, err := transport.Write([]byte(fmt.Sprintf("%04x%s", len(cmd), cmd))); err != nil {
		return false, err
	}

	reply := make([]byte, 4)
	if _, err := transport.Read(reply); err != nil {
		return false, err
	}
	if string(reply) == "OKAY" {
		data, _ := io.ReadAll(transport)
		return strings.Contains(string(data), "starting in"), nil
	}
	if string(reply) == "FAIL" {
		msg, _ := readLengthPrefixed(transport)
		return false, fmt.Errorf("tcpip failed: %s", string(msg))
	}
	return false, fmt.Errorf("unexpected reply: %s", string(reply))
}

// Usb switches the device back to USB mode
func (d *Device) Usb() (bool, error) {
	transport, err := d.Transport()
	if err != nil {
		return false, err
	}
	defer transport.Close()

	cmd := "usb:"
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
		return false, fmt.Errorf("usb failed: %s", string(msg))
	}
	return false, fmt.Errorf("unexpected reply: %s", string(reply))
}

// OpenLocal opens a local file/abstract socket on the device
func (d *Device) OpenLocal(path string) (net.Conn, error) {
	transport, err := d.Transport()
	if err != nil {
		return nil, err
	}

	cmd := fmt.Sprintf("local:%s", path)
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
		return nil, fmt.Errorf("openLocal failed: %s", string(msg))
	}
	transport.Close()
	return nil, fmt.Errorf("unexpected reply: %s", string(reply))
}

// OpenLog opens a log buffer on the device
func (d *Device) OpenLog(name string) (net.Conn, error) {
	transport, err := d.Transport()
	if err != nil {
		return nil, err
	}

	cmd := fmt.Sprintf("log:%s", name)
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
		return nil, fmt.Errorf("openLog failed: %s", string(msg))
	}
	transport.Close()
	return nil, fmt.Errorf("unexpected reply: %s", string(reply))
}

// OpenSocket opens a socket connection to a specific address on the device
// The address format can be:
//   - "tcp:port" (e.g., "tcp:5555")
//   - "tcp:host:port" (e.g., "tcp:192.168.1.100:5555")
//   - "local:name" (e.g., "local:/dev/socket/adb")
//   - "localabstract:name" (e.g., "localabstract:scrcpy")
//   - "localreserved:name"
//   - "localfilesystem:name"
func (d *Device) OpenSocket(addr string) (net.Conn, error) {
	return d.OpenSocketContext(context.Background(), addr)
}

// OpenSocketContext opens a socket connection to a specific address on the device with context support
func (d *Device) OpenSocketContext(ctx context.Context, addr string) (net.Conn, error) {
	transport, err := d.TransportContext(ctx)
	if err != nil {
		return nil, err
	}

	cmd := addr
	if _, err := transport.Write([]byte(fmt.Sprintf("%04x%s", len(cmd), cmd))); err != nil {
		transport.Close()
		return nil, err
	}

	reply := make([]byte, 4)
	if _, err := transport.Read(reply); err != nil {
		transport.Close()
		return nil, err
	}
	if string(reply) ==  StatusOkay {
		return transport, nil
	}
	if string(reply) == StatusFail {
		msg, _ := readLengthPrefixed(transport)
		transport.Close()
		return nil, fmt.Errorf("openSocket failed: %s", string(msg))
	}
	transport.Close()
	return nil, fmt.Errorf("unexpected reply: %s", string(reply))
}
