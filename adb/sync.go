package adb

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"path"
)

const (
	// Sync service protocol commands
	SYNC_STAT = "STAT"
	SYNC_LIST = "LIST"
	SYNC_SEND = "SEND"
	SYNC_RECV = "RECV"
	SYNC_DENT = "DENT"
	SYNC_DONE = "DONE"
	SYNC_DATA = "DATA"
	SYNC_OKAY = "OKAY"
	SYNC_FAIL = "FAIL"
)

const (
	DEFAULT_CHMOD   = 0o644
	DATA_MAX_LENGTH = 65536
	TEMP_PATH       = "/data/local/tmp"
)

// SyncService handles sync service operations for file transfers
type SyncService struct {
	conn   net.Conn
	device *Device
}

// NewSyncService creates a new sync service connection
func (d *Device) NewSyncService() (*SyncService, error) {
	conn, err := d.openSyncService()
	if err != nil {
		return nil, err
	}
	return &SyncService{
		conn:   conn,
		device: d,
	}, nil
}

// Close closes the sync service connection
func (s *SyncService) Close() error {
	if s.conn != nil {
		return s.conn.Close()
	}
	return nil
}

// Stat gets file information from the device
func (s *SyncService) Stat(filePath string) (*SyncStatEntry, error) {
	if err := s.sendCommandWithArg(SYNC_STAT, filePath); err != nil {
		return nil, err
	}

	reply, err := readAscii(s.conn, 4)
	if err != nil {
		return nil, err
	}

	switch reply {
	case SYNC_STAT:
		stat, err := readBytes(s.conn, 12)
		if err != nil {
			return nil, err
		}
		mode := binary.LittleEndian.Uint32(stat[0:4])
		size := binary.LittleEndian.Uint32(stat[4:8])
		mtime := binary.LittleEndian.Uint32(stat[8:12])
		if mode == 0 {
			return nil, fmt.Errorf("ENOENT, no such file or directory '%s'", filePath)
		}
		return &SyncStatEntry{
			Mode: mode,
			Size: size,
			Time: mtime,
		}, nil
	case SYNC_FAIL:
		return nil, s.readError()
	default:
		return nil, fmt.Errorf("unexpected reply: %s, expected 'STAT' or 'FAIL'", reply)
	}
}

// DirEntry represents a directory entry
type DirEntry struct {
	Name  string
	Mode  uint32
	Size  uint32
	Mtime uint32
}

// Readdir lists files in a directory on the device
func (s *SyncService) Readdir(dirPath string) ([]DirEntry, error) {
	var files []DirEntry

	if err := s.sendCommandWithArg(SYNC_LIST, dirPath); err != nil {
		return nil, err
	}

	for {
		reply, err := readAscii(s.conn, 4)
		if err != nil {
			return nil, err
		}

		switch reply {
		case SYNC_DENT:
			stat, err := readBytes(s.conn, 16)
			if err != nil {
				return nil, err
			}
			mode := binary.LittleEndian.Uint32(stat[0:4])
			size := binary.LittleEndian.Uint32(stat[4:8])
			mtime := binary.LittleEndian.Uint32(stat[8:12])
			namelen := binary.LittleEndian.Uint32(stat[12:16])

			nameData, err := readBytes(s.conn, int(namelen))
			if err != nil {
				return nil, err
			}
			name := string(nameData)

			// Skip '.' and '..' to match Node's fs.readdir()
			if name != "." && name != ".." {
				files = append(files, DirEntry{
					Name:  name,
					Mode:  mode,
					Size:  size,
					Mtime: mtime,
				})
			}
		case SYNC_DONE:
			_, err := readBytes(s.conn, 16)
			if err != nil {
				return nil, err
			}
			return files, nil
		case SYNC_FAIL:
			return nil, s.readError()
		default:
			return nil, fmt.Errorf("unexpected reply: %s, expected 'DENT', 'DONE' or 'FAIL'", reply)
		}
	}
}

// SyncPushOptions holds options for pushing files via sync service
type SyncPushOptions struct {
	Mode  uint32
	Mtime int64
}

// PushContent pushes data to a file on the device
func (s *SyncService) PushContent(data []byte, filePath string, opts SyncPushOptions) error {
	if opts.Mode == 0 {
		opts.Mode = DEFAULT_CHMOD
	}

	mode := opts.Mode | 0o100000 // Set S_IFREG (regular file)
	arg := fmt.Sprintf("%s,%d", filePath, mode)
	if err := s.sendCommandWithArg(SYNC_SEND, arg); err != nil {
		return err
	}

	return s.writeData(data, uint32(opts.Mtime))
}

// Push pushes a stream to a file on the device
func (s *SyncService) Push(reader io.Reader, filePath string, opts SyncPushOptions) error {
	if opts.Mode == 0 {
		opts.Mode = DEFAULT_CHMOD
	}

	mode := opts.Mode | 0o100000 // Set S_IFREG (regular file)
	arg := fmt.Sprintf("%s,%d", filePath, mode)
	if err := s.sendCommandWithArg(SYNC_SEND, arg); err != nil {
		return err
	}

	// Read all data from reader
	data, err := io.ReadAll(reader)
	if err != nil {
		return err
	}

	return s.writeData(data, uint32(opts.Mtime))
}

// Pull pulls a file from the device
func (s *SyncService) Pull(filePath string) (io.ReadCloser, error) {
	if err := s.sendCommandWithArg(SYNC_RECV, filePath); err != nil {
		return nil, err
	}

	pr, pw := io.Pipe()
	go func() {
		defer pw.Close()
		if err := s.readData(pw); err != nil {
			pw.CloseWithError(err)
		}
	}()

	return pr, nil
}

// writeData writes data to the device
func (s *SyncService) writeData(data []byte, timestamp uint32) error {
	offset := 0
	totalLen := len(data)

	for offset < totalLen {
		chunkSize := DATA_MAX_LENGTH
		if offset+chunkSize > totalLen {
			chunkSize = totalLen - offset
		}

		chunk := data[offset : offset+chunkSize]
		if err := s.sendCommandWithLength(SYNC_DATA, uint32(len(chunk))); err != nil {
			return err
		}

		if _, err := s.conn.Write(chunk); err != nil {
			return err
		}

		offset += chunkSize
	}

	// Send DONE command with timestamp
	if err := s.sendCommandWithLength(SYNC_DONE, timestamp); err != nil {
		return err
	}

	// Wait for OKAY or FAIL response
	return s.readFinalReply()
}

// readData reads data from the device
func (s *SyncService) readData(writer io.Writer) error {
	for {
		reply, err := readAscii(s.conn, 4)
		if err != nil {
			return err
		}

		switch reply {
		case SYNC_DATA:
			lengthData, err := readBytes(s.conn, 4)
			if err != nil {
				return err
			}
			length := binary.LittleEndian.Uint32(lengthData)

			data, err := readBytes(s.conn, int(length))
			if err != nil {
				return err
			}

			if _, err := writer.Write(data); err != nil {
				return err
			}
		case SYNC_DONE:
			_, err := readBytes(s.conn, 4)
			return err
		case SYNC_FAIL:
			return s.readError()
		default:
			return fmt.Errorf("unexpected reply: %s, expected 'DATA', 'DONE' or 'FAIL'", reply)
		}
	}
}

// readFinalReply reads the final reply after a push operation
func (s *SyncService) readFinalReply() error {
	reply, err := readAscii(s.conn, 4)
	if err != nil {
		return err
	}

	switch reply {
	case SYNC_OKAY:
		_, err := readBytes(s.conn, 4)
		return err
	case SYNC_FAIL:
		return s.readError()
	default:
		return fmt.Errorf("unexpected reply: %s, expected 'OKAY' or 'FAIL'", reply)
	}
}

// readError reads an error message from the device
func (s *SyncService) readError() error {
	lengthData, err := readBytes(s.conn, 4)
	if err != nil {
		return err
	}
	length := binary.LittleEndian.Uint32(lengthData)

	msgData, err := readBytes(s.conn, int(length))
	if err != nil {
		return err
	}
	return errors.New(string(msgData))
}

// sendCommandWithLength sends a command with a length parameter
func (s *SyncService) sendCommandWithLength(cmd string, length uint32) error {
	payload := make([]byte, len(cmd)+4)
	copy(payload, cmd)
	binary.LittleEndian.PutUint32(payload[len(cmd):], length)

	_, err := s.conn.Write(payload)
	return err
}

// sendCommandWithArg sends a command with a string argument
func (s *SyncService) sendCommandWithArg(cmd string, arg string) error {
	argLen := len(arg)
	payload := make([]byte, len(cmd)+4+argLen)

	pos := 0
	copy(payload[pos:], cmd)
	pos += len(cmd)

	binary.LittleEndian.PutUint32(payload[pos:], uint32(argLen))
	pos += 4

	copy(payload[pos:], arg)

	_, err := s.conn.Write(payload)
	return err
}

// TempPath returns the temporary path for a file
func (s *SyncService) TempPath(filePath string) string {
	return fmt.Sprintf("%s/%s", TEMP_PATH, path.Base(filePath))
}

// readAscii reads ASCII bytes from connection
func readAscii(conn net.Conn, n int) (string, error) {
	data := make([]byte, n)
	if _, err := io.ReadFull(conn, data); err != nil {
		return "", err
	}
	return string(data), nil
}

// readBytes reads bytes from connection
func readBytes(conn net.Conn, n int) ([]byte, error) {
	data := make([]byte, n)
	if _, err := io.ReadFull(conn, data); err != nil {
		return nil, err
	}
	return data, nil
}

// openSyncService opens a sync service connection for the device
func (d *Device) openSyncService() (net.Conn, error) {
	return d.OpenTransportConnection("sync:")
}
