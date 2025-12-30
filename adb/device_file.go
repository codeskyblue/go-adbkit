package adb

import (
	"io"
)

// Stat gets file information from the device using sync service
func (d *Device) Stat(path string) (*SyncStatEntry, error) {
	syncService, err := d.NewSyncService()
	if err != nil {
		return nil, err
	}
	defer syncService.Close()

	return syncService.Stat(path)
}

// Readdir lists files in a directory on the device using sync service
func (d *Device) Readdir(path string) ([]string, error) {
	syncService, err := d.NewSyncService()
	if err != nil {
		return nil, err
	}
	defer syncService.Close()

	entries, err := syncService.Readdir(path)
	if err != nil {
		return nil, err
	}

	// Convert DirEntry to string list (names only)
	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		files = append(files, entry.Name)
	}
	return files, nil
}

// Pull pulls a file from the device using sync service
func (d *Device) Pull(path string) (io.ReadCloser, error) {
	syncService, err := d.NewSyncService()
	if err != nil {
		return nil, err
	}

	return syncService.Pull(path)
}

// Push pushes a file to the device using sync service
func (d *Device) Push(reader io.Reader, path string, mode uint32) error {
	syncService, err := d.NewSyncService()
	if err != nil {
		return err
	}
	defer syncService.Close()

	opts := SyncPushOptions{
		Mode: mode,
	}
	return syncService.Push(reader, path, opts)
}
