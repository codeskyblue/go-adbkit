package adb

import (
	"fmt"
	"io"
	"strings"
)

// Stat gets file information from the device
func (d *Device) Stat(path string) (*SyncStatEntry, error) {
	conn, err := d.Shell(fmt.Sprintf("stat %s", path))
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	data, err := ReadAllFromConn(conn)
	if err != nil {
		return nil, err
	}

	return &SyncStatEntry{
		ModeStr: string(data),
	}, nil
}

// Readdir lists files in a directory on the device
func (d *Device) Readdir(path string) ([]string, error) {
	conn, err := d.Shell(fmt.Sprintf("ls %s", path))
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
func (d *Device) Pull(path string) (io.ReadCloser, error) {
	conn, err := d.Shell(fmt.Sprintf("cat %s", path))
	if err != nil {
		return nil, err
	}
	return conn, nil
}

// Push pushes a file to the device
func (d *Device) Push(contents, path string, mode uint32) error {
	cmd := fmt.Sprintf("echo '%s' > %s", contents, path)
	conn, err := d.Shell(cmd)
	if err != nil {
		return err
	}
	defer conn.Close()
	return nil
}
