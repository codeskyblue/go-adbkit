package adb

// SyncStatEntry represents file stat information from sync service
type SyncStatEntry struct {
	Mode    uint32
	Size    uint32
	Time    uint32
	ModeStr string // String representation like "-rw-r--r--"
}

// PushOptions holds options for pushing files
type PushOptions struct {
	Mode     uint32 // File permissions (e.g., 0644)
	Mtime    int64  // Modification time
	Compress bool
}
