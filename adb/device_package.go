package adb

import (
	"fmt"
	"io"
	"os"
	"strings"
)

// GetPackages gets the list of packages installed on the device
func (d *Device) GetPackages() ([]string, error) {
	conn, err := d.Shell("pm list packages")
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	data, err := ReadAllFromConn(conn)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(data), "\n")
	packages := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "package:") {
			pkg := strings.TrimPrefix(line, "package:")
			packages = append(packages, pkg)
		}
	}
	return packages, nil
}

// ClearPackageData clears the data of an application package
func (d *Device) ClearPackageData(pkg string) (bool, error) {
	cmd := fmt.Sprintf("shell:pm clear %s", pkg)
	err := d.ExecuteTransportCommand(cmd)
	if err != nil {
		return false, err
	}
	return true, nil
}

// Install installs an APK file to the device
func (d *Device) Install(apkPath string) (bool, error) {
	// Check if abb_exec feature is available
	hasAbbExec, err := d.hasAbbExecFeature()
	if err != nil {
		// If we can't check features, try traditional method
		return d.installUsingPush(apkPath)
	}

	if hasAbbExec {
		// Use modern abb_exec method (faster, no intermediate file)
		return d.installUsingAbbExec(apkPath)
	}

	// Fall back to traditional push + install method
	return d.installUsingPush(apkPath)
}

// hasAbbExecFeature checks if the device supports abb_exec feature
func (d *Device) hasAbbExecFeature() (bool, error) {
	features, err := d.client.Features()
	if err != nil {
		return false, err
	}
	for _, f := range features {
		if f == "abb_exec" {
			return true, nil
		}
	}
	return false, nil
}

// installUsingAbbExec installs APK using the abb_exec feature (streaming install)
func (d *Device) installUsingAbbExec(apkPath string) (bool, error) {
	// Open the local APK file
	file, err := os.Open(apkPath)
	if err != nil {
		return false, err
	}
	defer file.Close()

	// Get file size
	fileInfo, err := file.Stat()
	if err != nil {
		return false, err
	}

	// Build abb_exec command
	// Format: abb_exec:package install -S <size>
	cmd := fmt.Sprintf("abb_exec:package install -S %d", fileInfo.Size())

	// Create transport and send command
	transport, err := d.Transport()
	if err != nil {
		return false, err
	}
	defer transport.Close()

	if _, err := transport.Write([]byte(fmt.Sprintf("%04x%s", len(cmd), cmd))); err != nil {
		return false, err
	}

	// Check if command was accepted
	reply := make([]byte, 4)
	if _, err := transport.Read(reply); err != nil {
		return false, err
	}

	if string(reply) != "OKAY" {
		msg, _ := readLengthPrefixed(transport)
		return false, fmt.Errorf("abb_exec command failed: %s", string(msg))
	}

	// Stream the APK content
	_, err = io.Copy(transport, file)
	if err != nil {
		return false, err
	}

	// Read the response
	data, err := io.ReadAll(transport)
	if err != nil {
		return false, err
	}

	output := string(data)
	return strings.Contains(output, "Success"), nil
}

// installUsingPush installs APK using traditional push + pm install method
func (d *Device) installUsingPush(apkPath string) (bool, error) {
	// Open the local APK file
	file, err := os.Open(apkPath)
	if err != nil {
		return false, err
	}
	defer file.Close()

	// Push to temporary location on device
	tempPath := "/data/local/tmp/install.apk"
	err = d.Push(file, tempPath, 0644)
	if err != nil {
		return false, err
	}

	// Install using pm install
	output, err := d.RunCommand(fmt.Sprintf("pm install -r %s", tempPath))
	if err != nil {
		return false, err
	}

	// Clean up
	d.RunCommand(fmt.Sprintf("rm -f %s", tempPath))

	return strings.Contains(output, "Success"), nil
}

// InstallRemote installs an APK that's already on the device
func (d *Device) InstallRemote(apkPath string) (bool, error) {
	cmd := fmt.Sprintf("shell:pm install -r %s", apkPath)
	data, err := d.ExecuteTransportCommandWithResponse(cmd)
	if err != nil {
		return false, err
	}
	output := strings.TrimSpace(string(data))
	d.Shell(fmt.Sprintf("rm -f %s", apkPath))
	return strings.Contains(output, "Success"), nil
}

// Uninstall uninstalls a package from the device
func (d *Device) Uninstall(pkg string) (bool, error) {
	cmd := fmt.Sprintf("shell:pm uninstall %s", pkg)
	data, err := d.ExecuteTransportCommandWithResponse(cmd)
	if err != nil {
		return false, err
	}
	return strings.Contains(string(data), "Success"), nil
}

// IsInstalled checks if a package is installed
func (d *Device) IsInstalled(pkg string) (bool, error) {
	cmd := fmt.Sprintf("shell:pm path %s", pkg)
	data, err := d.ExecuteTransportCommandWithResponse(cmd)
	if err != nil {
		return false, err
	}
	return strings.Contains(string(data), "package:"), nil
}
