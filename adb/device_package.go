package adb

import (
	"fmt"
	"io"
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

// Clear clears the data of an application
func (d *Device) Clear(pkg string) (bool, error) {
	transport, err := d.Transport()
	if err != nil {
		return false, err
	}
	defer transport.Close()

	cmd := fmt.Sprintf("shell:pm clear %s", pkg)
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
		return false, fmt.Errorf("clear failed: %s", string(msg))
	}
	return false, fmt.Errorf("unexpected reply: %s", string(reply))
}

// Install installs an APK file to the device
func (d *Device) Install(apkPath string) (bool, error) {
	transport, err := d.Transport()
	if err != nil {
		return false, err
	}
	defer transport.Close()

	cmd := fmt.Sprintf("shell:pm install -r %s", apkPath)
	if _, err := transport.Write([]byte(fmt.Sprintf("%04x%s", len(cmd), cmd))); err != nil {
		return false, err
	}

	reply := make([]byte, 4)
	if _, err := transport.Read(reply); err != nil {
		return false, err
	}
	if string(reply) == "OKAY" {
		data, _ := io.ReadAll(transport)
		return strings.Contains(string(data), "Success"), nil
	}
	if string(reply) == "FAIL" {
		msg, _ := readLengthPrefixed(transport)
		return false, fmt.Errorf("install failed: %s", string(msg))
	}
	return false, fmt.Errorf("unexpected reply: %s", string(reply))
}

// InstallRemote installs an APK that's already on the device
func (d *Device) InstallRemote(apkPath string) (bool, error) {
	transport, err := d.Transport()
	if err != nil {
		return false, err
	}
	defer transport.Close()

	cmd := fmt.Sprintf("shell:pm install -r %s", apkPath)
	if _, err := transport.Write([]byte(fmt.Sprintf("%04x%s", len(cmd), cmd))); err != nil {
		return false, err
	}

	reply := make([]byte, 4)
	if _, err := transport.Read(reply); err != nil {
		return false, err
	}
	if string(reply) == "OKAY" {
		data, _ := io.ReadAll(transport)
		output := strings.TrimSpace(string(data))
		d.Shell(fmt.Sprintf("rm -f %s", apkPath))
		return strings.Contains(output, "Success"), nil
	}
	if string(reply) == "FAIL" {
		msg, _ := readLengthPrefixed(transport)
		return false, fmt.Errorf("installRemote failed: %s", string(msg))
	}
	return false, fmt.Errorf("unexpected reply: %s", string(reply))
}

// Uninstall uninstalls a package from the device
func (d *Device) Uninstall(pkg string) (bool, error) {
	transport, err := d.Transport()
	if err != nil {
		return false, err
	}
	defer transport.Close()

	cmd := fmt.Sprintf("shell:pm uninstall %s", pkg)
	if _, err := transport.Write([]byte(fmt.Sprintf("%04x%s", len(cmd), cmd))); err != nil {
		return false, err
	}

	reply := make([]byte, 4)
	if _, err := transport.Read(reply); err != nil {
		return false, err
	}
	if string(reply) == "OKAY" {
		data, _ := io.ReadAll(transport)
		return strings.Contains(string(data), "Success"), nil
	}
	if string(reply) == "FAIL" {
		msg, _ := readLengthPrefixed(transport)
		return false, fmt.Errorf("uninstall failed: %s", string(msg))
	}
	return false, fmt.Errorf("unexpected reply: %s", string(reply))
}

// IsInstalled checks if a package is installed
func (d *Device) IsInstalled(pkg string) (bool, error) {
	transport, err := d.Transport()
	if err != nil {
		return false, err
	}
	defer transport.Close()

	cmd := fmt.Sprintf("shell:pm path %s", pkg)
	if _, err := transport.Write([]byte(fmt.Sprintf("%04x%s", len(cmd), cmd))); err != nil {
		return false, err
	}

	reply := make([]byte, 4)
	if _, err := transport.Read(reply); err != nil {
		return false, err
	}
	if string(reply) == "OKAY" {
		data, _ := io.ReadAll(transport)
		return strings.Contains(string(data), "package:"), nil
	}
	if string(reply) == "FAIL" {
		msg, _ := readLengthPrefixed(transport)
		return false, fmt.Errorf("isInstalled failed: %s", string(msg))
	}
	return false, fmt.Errorf("unexpected reply: %s", string(reply))
}
