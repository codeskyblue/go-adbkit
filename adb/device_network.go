package adb

import (
	"context"
	"fmt"
	"net"
	"strings"
)

// TcpIp switches the device to TCP/IP mode
func (d *Device) TcpIp(port int) (bool, error) {
	if port == 0 {
		port = 5555
	}
	cmd := fmt.Sprintf("tcpip:%d", port)
	data, err := d.ExecuteTransportCommandWithResponse(cmd)
	if err != nil {
		return false, err
	}
	return strings.Contains(string(data), "starting in"), nil
}

// Usb switches the device back to USB mode
func (d *Device) Usb() (bool, error) {
	err := d.ExecuteTransportCommand("usb:")
	return err == nil, err
}

// OpenLocal opens a local file/abstract socket on the device
func (d *Device) OpenLocal(path string) (net.Conn, error) {
	cmd := fmt.Sprintf("local:%s", path)
	return d.OpenTransportConnection(cmd)
}

// OpenLog opens a log buffer on the device
func (d *Device) OpenLog(name string) (net.Conn, error) {
	cmd := fmt.Sprintf("log:%s", name)
	return d.OpenTransportConnection(cmd)
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

	if err := transport.ExecuteSimpleCommand(addr); err != nil {
		transport.Close()
		return nil, err
	}

	return transport, nil
}
