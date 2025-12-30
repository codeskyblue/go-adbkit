package adb

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
)

// Shell opens a shell stream on the device
func (d *Device) Shell(command string) (net.Conn, error) {
	return d.ShellContext(context.Background(), command)
}

// ShellContext opens a shell stream with context support
func (d *Device) ShellContext(ctx context.Context, command string) (net.Conn, error) {
	transport, err := d.TransportContext(ctx)
	if err != nil {
		return nil, err
	}
	svc := []byte("shell:" + command)
	length := fmt.Sprintf("%04x", len(svc))
	if _, err := transport.Write([]byte(length)); err != nil {
		transport.Close()
		return nil, err
	}
	if _, err := transport.Write(svc); err != nil {
		transport.Close()
		return nil, err
	}
	if _, err := transport.CheckStatus(); err != nil {
		transport.Close()
		return nil, err
	}
	return transport, nil
}

// RunCommand runs a shell command on the device
func (d *Device) RunCommand(command string) (string, error) {
	return d.RunCommandContext(context.Background(), command)
}

// RunCommandContext runs a shell command with context support
func (d *Device) RunCommandContext(ctx context.Context, command string) (string, error) {
	conn, err := d.ShellContext(ctx, command)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	data, err := readAllWithContext(ctx, conn)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

// readAllWithContext reads all data from a connection with context support
func readAllWithContext(ctx context.Context, conn net.Conn) ([]byte, error) {
	type result struct {
		data []byte
		err  error
	}

	resultChan := make(chan result, 1)

	go func() {
		data, err := io.ReadAll(bufio.NewReader(conn))
		resultChan <- result{data, err}
	}()

	select {
	case <-ctx.Done():
		conn.Close()
		return nil, ctx.Err()
	case res := <-resultChan:
		return res.data, res.err
	}
}
