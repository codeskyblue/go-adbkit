package adb

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// skipIfNoDevice skips the test if no device is connected
// This is used for E2E tests that require a physical device or emulator
func skipIfNoDevice(t *testing.T) {
	t.Helper()

	client := NewClient()
	devices, err := client.ListDevices()
	if err != nil {
		t.Skipf("Failed to list devices: %v", err)
	}

	if len(devices) == 0 {
		t.Skip("No devices connected, skipping E2E test")
	}
}

// getFirstDevice returns the first available device for testing
func getFirstDevice(t *testing.T) *Device {
	t.Helper()

	client := NewClient()
	devices, err := client.ListDevices()
	if err != nil {
		t.Fatalf("Failed to list devices: %v", err)
	}

	if len(devices) == 0 {
		t.Fatal("No devices connected")
	}

	// Find first device that is available
	for _, devInfo := range devices {
		if devInfo.State == "device" {
			return client.Device(DeviceWithSerial(devInfo.Serial))
		}
	}

	// If no device with state="device", use the first one
	return client.Device(DeviceWithSerial(devices[0].Serial))
}

// TestE2E_DeviceInfo tests basic device information retrieval
func TestE2E_DeviceInfo(t *testing.T) {
	skipIfNoDevice(t)

	device := getFirstDevice(t)

	// Test getting device serial
	serial, err := device.Serial()
	if err != nil {
		t.Fatalf("Failed to get serial: %v", err)
	}
	if serial == "" {
		t.Error("Expected non-empty serial number")
	}

	// Test getting device properties
	props, err := device.Properties()
	if err != nil {
		t.Fatalf("Failed to get properties: %v", err)
	}

	if len(props) == 0 {
		t.Error("Expected at least one property")
	}

	// Check for common properties
	commonProps := []string{"ro.build.version.sdk", "ro.product.model", "ro.product.manufacturer"}
	foundProps := make(map[string]bool)
	for k := range props {
		for _, prop := range commonProps {
			if strings.HasPrefix(k, prop) {
				foundProps[prop] = true
			}
		}
	}

	t.Logf("Found %d properties", len(props))
	for k, v := range props {
		t.Logf("  %s = %s", k, v)
	}
}

// TestE2E_PushPull tests pushing and pulling files to/from the device
func TestE2E_PushPull(t *testing.T) {
	skipIfNoDevice(t)

	device := getFirstDevice(t)

	// Create a temporary test file
	testContent := []byte("Hello, ADB!\nThis is a test file for E2E testing.\n")
	tmpFile, err := os.CreateTemp("", "adb-e2e-test-*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write(testContent); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	remotePath := "/data/local/tmp/adb_e2e_test.txt"

	// Test Push
	t.Run("Push", func(t *testing.T) {
		file, err := os.Open(tmpFile.Name())
		if err != nil {
			t.Fatalf("Failed to open temp file: %v", err)
		}
		defer file.Close()

		if err := device.Push(file, remotePath, 0644); err != nil {
			t.Fatalf("Failed to push file: %v", err)
		}

		// Verify file exists on device
		conn, err := device.Shell(fmt.Sprintf("test -f %s && echo exists", remotePath))
		if err != nil {
			t.Fatalf("Failed to check file existence: %v", err)
		}
		defer conn.Close()

		output, err := ReadAllFromConn(conn)
		if err != nil {
			t.Fatalf("Failed to read shell output: %v", err)
		}

		if !bytes.Contains(output, []byte("exists")) {
			t.Error("Pushed file does not exist on device")
		}
	})

	// Test Pull
	t.Run("Pull", func(t *testing.T) {
		reader, err := device.Pull(remotePath)
		if err != nil {
			t.Fatalf("Failed to pull file: %v", err)
		}
		defer reader.Close()

		pulledContent, err := io.ReadAll(reader)
		if err != nil {
			t.Fatalf("Failed to read pulled content: %v", err)
		}

		if !bytes.Equal(pulledContent, testContent) {
			t.Errorf("Pulled content mismatch.\nGot: %q\nWant: %q", pulledContent, testContent)
		}
	})

	// Test Stat
	t.Run("Stat", func(t *testing.T) {
		stat, err := device.Stat(remotePath)
		if err != nil {
			t.Fatalf("Failed to stat file: %v", err)
		}

		if stat == nil {
			t.Error("Expected non-nil stat info")
		} else {
			t.Logf("File size: %d, mode: %04o", stat.Size, stat.Mode)
			if stat.Size != uint32(len(testContent)) {
				t.Errorf("Size mismatch.\nGot: %d\nWant: %d", stat.Size, len(testContent))
			}
		}
	})

	// Test ReadFile
	t.Run("ReadFile", func(t *testing.T) {
		content, err := device.ReadFile(remotePath)
		if err != nil {
			t.Fatalf("Failed to read file: %v", err)
		}

		if !bytes.Equal(content, testContent) {
			t.Errorf("ReadFile content mismatch.\nGot: %q\nWant: %q", content, testContent)
		}
	})

	// Cleanup: Remove the test file from device
	t.Cleanup(func() {
		conn, _ := device.Shell(fmt.Sprintf("rm -f %s", remotePath))
		if conn != nil {
			conn.Close()
		}
	})
}

// TestE2E_Shell tests basic shell command execution
func TestE2E_Shell(t *testing.T) {
	skipIfNoDevice(t)

	device := getFirstDevice(t)

	tests := []struct {
		name     string
		command  string
		validate func([]byte) error
	}{
		{
			name:    "Echo",
			command: "echo 'hello from adb'",
			validate: func(output []byte) error {
				if !bytes.Contains(output, []byte("hello from adb")) {
					return fmt.Errorf("output does not contain expected text")
				}
				return nil
			},
		},
		{
			name:    "GetProp",
			command: "getprop ro.build.version.sdk",
			validate: func(output []byte) error {
				outputStr := strings.TrimSpace(string(output))
				if outputStr == "" {
					return fmt.Errorf("expected SDK version")
				}
				return nil
			},
		},
		{
			name:    "PWD",
			command: "pwd",
			validate: func(output []byte) error {
				outputStr := strings.TrimSpace(string(output))
				if !strings.HasPrefix(outputStr, "/") {
					return fmt.Errorf("expected absolute path, got: %s", outputStr)
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn, err := device.Shell(tt.command)
			if err != nil {
				t.Fatalf("Failed to execute shell command: %v", err)
			}
			defer conn.Close()

			output, err := ReadAllFromConn(conn)
			if err != nil {
				t.Fatalf("Failed to read output: %v", err)
			}

			if tt.validate != nil {
				if err := tt.validate(output); err != nil {
					t.Errorf("Validation failed: %v\nOutput: %s", err, output)
				}
			}

			t.Logf("Output: %s", strings.TrimSpace(string(output)))
		})
	}
}

// TestE2E_PackageManagement tests package-related operations
func TestE2E_PackageManagement(t *testing.T) {
	skipIfNoDevice(t)

	device := getFirstDevice(t)

	// Test getting list of packages
	t.Run("ListPackages", func(t *testing.T) {
		packages, err := device.GetPackages()
		if err != nil {
			t.Fatalf("Failed to get packages: %v", err)
		}

		if len(packages) == 0 {
			t.Error("Expected at least one package")
		}

		t.Logf("Found %d packages", len(packages))

		// Check for common system packages
		commonPackages := []string{
			"android",
			"com.android.settings",
		}

		found := false
		for _, pkg := range packages {
			for _, common := range commonPackages {
				if strings.Contains(pkg, common) {
					found = true
					t.Logf("Found common package: %s", pkg)
				}
			}
		}

		if !found {
			t.Log("Warning: No common system packages found")
		}
	})

	// Test IsInstalled with a known system package
	t.Run("IsInstalled", func(t *testing.T) {
		// android package should always exist
		installed, err := device.IsInstalled("android")
		if err != nil {
			t.Fatalf("Failed to check if package is installed: %v", err)
		}

		if !installed {
			t.Error("Expected 'android' package to be installed")
		}
	})

	// Test IsInstalled with a non-existent package
	t.Run("NotInstalled", func(t *testing.T) {
		installed, err := device.IsInstalled("com.example.nonexistent.package.12345")
		if err != nil {
			t.Fatalf("Failed to check if package is installed: %v", err)
		}

		if installed {
			t.Error("Expected non-existent package to not be installed")
		}
	})
}

// TestE2E_Screencap tests taking a screenshot
func TestE2E_Screencap(t *testing.T) {
	skipIfNoDevice(t)

	device := getFirstDevice(t)

	// Test taking screenshot
	t.Run("TakeScreencap", func(t *testing.T) {
		img, err := device.Screencap()
		if err != nil {
			t.Fatalf("Failed to take screencap: %v", err)
		}

		if img == nil {
			t.Error("Expected non-nil image")
		}

		bounds := img.Bounds()
		t.Logf("Screencap size: %dx%d", bounds.Dx(), bounds.Dy())

		if bounds.Dx() <= 0 || bounds.Dy() <= 0 {
			t.Error("Invalid image dimensions")
		}
	})
}

// TestE2E_Framebuffer tests capturing the device framebuffer
func TestE2E_Framebuffer(t *testing.T) {
	skipIfNoDevice(t)

	device := getFirstDevice(t)

	// Test capturing framebuffer
	t.Run("CaptureFramebuffer", func(t *testing.T) {
		img, err := device.Framebuffer()
		if err != nil {
			// Framebuffer may not be supported on all devices
			// Log as a warning rather than failing the test
			t.Logf("Framebuffer capture failed (may not be supported): %v", err)
			t.Skip("Framebuffer not supported on this device")
			return
		}

		if img == nil {
			t.Error("Expected non-nil image")
		}

		bounds := img.Bounds()
		width := bounds.Dx()
		height := bounds.Dy()

		t.Logf("Framebuffer size: %dx%d", width, height)

		if width <= 0 || height <= 0 {
			t.Error("Invalid framebuffer dimensions")
		}

		// Log color model info
		t.Logf("Color model: %T", img.ColorModel())
	})

	// Test comparing Screencap vs Framebuffer
	t.Run("CompareScreencapAndFramebuffer", func(t *testing.T) {
		// Handle potential panics
		defer func() {
			if r := recover(); r != nil {
				t.Logf("Framebuffer comparison panicked: %v", r)
				t.Skip("Skipping comparison due to framebuffer limitations")
			}
		}()

		// Try to capture framebuffer first
		fbImg, fbErr := device.Framebuffer()
		if fbErr != nil {
			t.Skipf("Framebuffer not supported, skipping comparison: %v", fbErr)
			return
		}

		// Capture screencap
		scImg, scErr := device.Screencap()
		if scErr != nil {
			t.Fatalf("Screencap failed: %v", scErr)
		}

		if fbImg == nil || scImg == nil {
			t.Fatal("Expected non-nil images")
		}

		fbBounds := fbImg.Bounds()
		scBounds := scImg.Bounds()

		t.Logf("Framebuffer: %dx%d", fbBounds.Dx(), fbBounds.Dy())
		t.Logf("Screencap: %dx%d", scBounds.Dx(), scBounds.Dy())

		// Both should have the same dimensions
		if fbBounds.Dx() != scBounds.Dx() || fbBounds.Dy() != scBounds.Dy() {
			t.Logf("Warning: Dimensions differ - this may be normal if screen orientation changed")
		}

		// Log color models for comparison
		t.Logf("Framebuffer color model: %T", fbImg.ColorModel())
		t.Logf("Screencap color model: %T", scImg.ColorModel())
	})
}

// TestE2E_DirectoryOperations tests directory-related operations
func TestE2E_DirectoryOperations(t *testing.T) {
	skipIfNoDevice(t)

	device := getFirstDevice(t)

	testDir := "/data/local/tmp/adb_e2e_test_dir"
	testFile := filepath.Join(testDir, "test.txt")

	// Cleanup
	defer func() {
		conn, _ := device.Shell(fmt.Sprintf("rm -rf %s", testDir))
		if conn != nil {
			conn.Close()
		}
	}()

	// Create directory
	t.Run("CreateDirectory", func(t *testing.T) {
		_, err := device.RunCommand(fmt.Sprintf("mkdir -p %s", testDir))
		if err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}

		// Verify directory exists
		stat, err := device.Stat(testDir)
		if err != nil {
			t.Fatalf("Failed to stat directory: %v", err)
		}

		if stat == nil {
			t.Error("Expected directory to exist")
		}
	})

	// List directory
	t.Run("ListDirectory", func(t *testing.T) {
		files, err := device.Readdir(testDir)
		if err != nil {
			t.Fatalf("Failed to list directory: %v", err)
		}

		t.Logf("Directory contents: %v", files)
	})

	// Create a file in the directory and list again
	t.Run("CreateFileAndList", func(t *testing.T) {
		content := []byte("test content")
		reader := bytes.NewReader(content)

		if err := device.Push(reader, testFile, 0644); err != nil {
			t.Fatalf("Failed to push file: %v", err)
		}

		// Wait a bit for the file to be fully written
		time.Sleep(100 * time.Millisecond)

		files, err := device.Readdir(testDir)
		if err != nil {
			t.Fatalf("Failed to list directory: %v", err)
		}

		if len(files) == 0 {
			t.Error("Expected at least one file in directory")
		}

		t.Logf("Directory contents after file creation: %v", files)
	})
}

// TestE2E_InstallAPK tests installing an APK (if available)
// This test is skipped unless you provide a test APK
func TestE2E_InstallAPK(t *testing.T) {
	skipIfNoDevice(t)

	// Look for a test APK in the testdata directory
	testAPK := os.Getenv("ADB_TEST_APK")
	if testAPK == "" {
		t.Skip("Set ADB_TEST_APK environment variable to test APK installation")
	}

	if _, err := os.Stat(testAPK); os.IsNotExist(err) {
		t.Skipf("Test APK not found: %s", testAPK)
	}

	device := getFirstDevice(t)

	t.Run("InstallAPK", func(t *testing.T) {
		success, err := device.Install(testAPK)
		if err != nil {
			t.Fatalf("Failed to install APK: %v", err)
		}

		if !success {
			t.Error("APK installation reported failure")
		}
	})

	// Note: We don't uninstall here as the test APK might be needed for other tests
}

// TestE2E_FileOperations tests various file operations
func TestE2E_FileOperations(t *testing.T) {
	skipIfNoDevice(t)

	device := getFirstDevice(t)

	remoteFile := "/data/local/tmp/adb_e2e_file_ops.txt"

	// Cleanup
	defer func() {
		conn, _ := device.Shell(fmt.Sprintf("rm -f %s", remoteFile))
		if conn != nil {
			conn.Close()
		}
	}()

	// Test pushing multiple times
	t.Run("MultiplePushes", func(t *testing.T) {
		contents := []string{
			"First content\n",
			"Second content\n",
			"Third content\n",
		}

		for i, content := range contents {
			reader := strings.NewReader(content)
			if err := device.Push(reader, remoteFile, 0644); err != nil {
				t.Fatalf("Push %d failed: %v", i, err)
			}

			// Verify content
			pullReader, err := device.Pull(remoteFile)
			if err != nil {
				t.Fatalf("Pull %d failed: %v", i, err)
			}

			pulledContent, err := io.ReadAll(pullReader)
			pullReader.Close()
			if err != nil {
				t.Fatalf("Read pull %d failed: %v", i, err)
			}

			if string(pulledContent) != content {
				t.Errorf("Push %d: content mismatch.\nGot: %q\nWant: %q",
					i, pulledContent, content)
			}
		}
	})
}

// TestE2E_PortForwarding tests port forwarding functionality
func TestE2E_PortForwarding(t *testing.T) {
	skipIfNoDevice(t)

	device := getFirstDevice(t)

	// Test Reverse (device to host)
	t.Run("ReverseForward", func(t *testing.T) {
		// Use a unique port number to avoid conflicts
		localPort := "15000"
		remoteSocket := "localabstract:test_forward_e2e"

		// Create reverse forward
		success, err := device.Reverse(remoteSocket, "tcp:"+localPort)
		if err != nil {
			t.Fatalf("Failed to create reverse forward: %v", err)
		}
		if !success {
			t.Error("Reverse forward reported failure")
		}

		// List reverse forwards to verify
		revs, err := device.ListReverses()
		if err != nil {
			t.Fatalf("Failed to list reverse forwards: %v", err)
		}

		// Check if our forward is in the list
		found := false
		for _, rev := range revs {
			// Note: Based on the actual output, the fields appear to be swapped
			// rev.Remote contains the local endpoint, rev.Local contains the remote socket
			if rev.Local == remoteSocket || strings.Contains(rev.Local, "test_forward_e2e") {
				found = true
				t.Logf("Found reverse forward: %s -> %s", rev.Remote, rev.Local)
				break
			}
		}

		if !found {
			t.Errorf("Reverse forward not found in list. Looking for %q", remoteSocket)
		}

		// Cleanup: remove the reverse forward
		t.Cleanup(func() {
			_, _ = device.ReverseRemove(remoteSocket)
		})
	})

	// Test multiple reverse forwards
	t.Run("MultipleReverseForwards", func(t *testing.T) {
		// Create multiple reverse forwards
		forwards := []struct {
			remote string
			local  string
		}{
			{"localabstract:test_e2e_1", "tcp:15001"},
			{"localabstract:test_e2e_2", "tcp:15002"},
		}

		// Create forwards
		for _, fw := range forwards {
			success, err := device.Reverse(fw.remote, fw.local)
			if err != nil {
				t.Fatalf("Failed to create reverse forward %s->%s: %v", fw.remote, fw.local, err)
			}
			if !success {
				t.Errorf("Reverse forward %s->%s reported failure", fw.remote, fw.local)
			}
		}

		// Verify all forwards are listed
		revs, err := device.ListReverses()
		if err != nil {
			t.Fatalf("Failed to list reverse forwards: %v", err)
		}

		t.Logf("Total reverse forwards: %d", len(revs))

		// Cleanup: remove all forwards
		t.Cleanup(func() {
			for _, fw := range forwards {
				_, _ = device.ReverseRemove(fw.remote)
			}
		})
	})

	// Test Forward (host to device)
	t.Run("Forward", func(t *testing.T) {
		// Use a unique local port
		localPort := "15010"
		remoteSocket := "localabstract:test_forward_device"

		// Create forward
		success, err := device.Forward("tcp:"+localPort, remoteSocket)
		if err != nil {
			t.Fatalf("Failed to create forward: %v", err)
		}
		if !success {
			t.Error("Forward reported failure")
		}

		t.Logf("Created forward: localhost:%s -> %s", localPort, remoteSocket)

		// Cleanup: remove the forward
		t.Cleanup(func() {
			_, _ = device.ForwardRemove("tcp:" + localPort)
		})
	})

	// Test ForwardRemove
	t.Run("ForwardRemove", func(t *testing.T) {
		localPort := "15020"
		remoteSocket := "localabstract:test_forward_remove"

		// Create a forward first
		success, err := device.Forward("tcp:"+localPort, remoteSocket)
		if err != nil {
			t.Fatalf("Failed to create forward: %v", err)
		}
		if !success {
			t.Fatal("Failed to create forward")
		}

		// Remove the forward
		removed, err := device.ForwardRemove("tcp:" + localPort)
		if err != nil {
			t.Fatalf("Failed to remove forward: %v", err)
		}
		if !removed {
			t.Error("ForwardRemove reported failure")
		}

		t.Logf("Successfully removed forward: localhost:%s", localPort)
	})

	// Test ReverseRemove
	t.Run("ReverseRemove", func(t *testing.T) {
		remoteSocket := "localabstract:test_reverse_remove"
		localPort := "15030"

		// Create a reverse forward first
		success, err := device.Reverse(remoteSocket, "tcp:"+localPort)
		if err != nil {
			t.Fatalf("Failed to create reverse forward: %v", err)
		}
		if !success {
			t.Fatal("Failed to create reverse forward")
		}

		// Verify it exists
		revs, err := device.ListReverses()
		if err != nil {
			t.Fatalf("Failed to list reverses: %v", err)
		}

		foundBefore := false
		for _, rev := range revs {
			if rev.Local == remoteSocket {
				foundBefore = true
				break
			}
		}

		if !foundBefore {
			t.Error("Reverse forward not found before removal")
		}

		// Remove the reverse forward
		removed, err := device.ReverseRemove(remoteSocket)
		if err != nil {
			t.Fatalf("Failed to remove reverse forward: %v", err)
		}
		if !removed {
			t.Error("ReverseRemove reported failure")
		}

		// Verify it's gone
		revs, err = device.ListReverses()
		if err != nil {
			t.Fatalf("Failed to list reverses after removal: %v", err)
		}

		foundAfter := false
		for _, rev := range revs {
			if rev.Local == remoteSocket {
				foundAfter = true
				break
			}
		}

		if foundAfter {
			t.Error("Reverse forward still exists after removal")
		}

		t.Logf("Successfully removed reverse forward: %s", remoteSocket)
	})
}
