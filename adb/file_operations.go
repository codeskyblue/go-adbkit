package adb

import (
	"fmt"
	"io"
	"net"
	"strings"
)

// SyncStatEntry represents file stat information from sync service
type SyncStatEntry struct {
	Mode    uint32
	Size    uint32
	Time    uint32
	ModeStr string // String representation like "-rw-r--r--"
}

// Stat gets file information from the device
func (c *Client) Stat(serial, path string) (*SyncStatEntry, error) {
	// This requires sync service implementation
	// For now, use shell stat command as fallback
	conn, err := c.Shell(serial, fmt.Sprintf("stat %s", path))
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	data, err := ReadAllFromConn(conn)
	if err != nil {
		return nil, err
	}

	// Parse stat output (simplified)
	return &SyncStatEntry{
		ModeStr: string(data),
	}, nil
}

// Readdir lists files in a directory on the device
func (c *Client) Readdir(serial, path string) ([]string, error) {
	conn, err := c.Shell(serial, fmt.Sprintf("ls %s", path))
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	data, err := ReadAllFromConn(conn)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(data), "\n")
	files := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.Contains(line, "No such file") {
			files = append(files, line)
		}
	}
	return files, nil
}

// Pull pulls a file from the device
func (c *Client) Pull(serial, path string) (io.ReadCloser, error) {
	// For simplicity, use shell cat command
	conn, err := c.Shell(serial, fmt.Sprintf("cat %s", path))
	if err != nil {
		return nil, err
	}
	return conn, nil
}

// PushOptions holds options for pushing files
type PushOptions struct {
	Mode     uint32 // File permissions (e.g., 0644)
	Mtime    int64  // Modification time
	Compress bool
}

// Push pushes a file to the device
func (c *Client) Push(serial, contents, path string, mode uint32) error {
	// For simplicity, use shell to write file
	// Full implementation would use sync service
	cmd := fmt.Sprintf("echo '%s' > %s", contents, path)
	conn, err := c.Shell(serial, cmd)
	if err != nil {
		return err
	}
	defer conn.Close()
	return nil
}

// syncService would establish a sync service connection
// Full implementation would go here
func (c *Client) syncService(serial string) (net.Conn, error) {
	transport, err := c.Transport(serial)
	if err != nil {
		return nil, err
	}

	cmd := "sync:"
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
		return nil, fmt.Errorf("sync service failed: %s", string(msg))
	}
	transport.Close()
	return nil, fmt.Errorf("unexpected reply: %s", string(reply))
}
