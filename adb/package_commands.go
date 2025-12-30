package adb

import (
	"fmt"
	"io"
	"strings"
)

// GetPackages gets the list of packages installed on the device
func (c *Client) GetPackages(serial string) ([]string, error) {
	conn, err := c.Shell(serial, "pm list packages")
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
func (c *Client) Clear(serial, pkg string) (bool, error) {
	transport, err := c.Transport(serial)
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
func (c *Client) Install(serial, apkPath string) (bool, error) {
	// This is a simplified version - the full implementation would sync the file
	// For now, we use shell-based install
	transport, err := c.Transport(serial)
	if err != nil {
		return false, err
	}
	defer transport.Close()

	// Push the file and install (simplified - actual implementation needs sync service)
	cmd := fmt.Sprintf("shell:pm install -r %s", apkPath)
	if _, err := transport.Write([]byte(fmt.Sprintf("%04x%s", len(cmd), cmd))); err != nil {
		return false, err
	}

	reply := make([]byte, 4)
	if _, err := transport.Read(reply); err != nil {
		return false, err
	}
	if string(reply) == "OKAY" {
		// Read output to check for "Success"
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
func (c *Client) InstallRemote(serial, apkPath string) (bool, error) {
	transport, err := c.Transport(serial)
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
		// Clean up the temp file
		c.Shell(serial, fmt.Sprintf("rm -f %s", apkPath))
		return strings.Contains(output, "Success"), nil
	}
	if string(reply) == "FAIL" {
		msg, _ := readLengthPrefixed(transport)
		return false, fmt.Errorf("installRemote failed: %s", string(msg))
	}
	return false, fmt.Errorf("unexpected reply: %s", string(reply))
}

// Uninstall uninstalls a package from the device
func (c *Client) Uninstall(serial, pkg string) (bool, error) {
	transport, err := c.Transport(serial)
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
func (c *Client) IsInstalled(serial, pkg string) (bool, error) {
	transport, err := c.Transport(serial)
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

// StartActivityOptions holds options for starting an activity
type StartActivityOptions struct {
	Action       string
	Data         string
	Type         string
	Category     []string
	Component    string
	Flags        []string
	User         int // 0 for primary user, -1 for all, null for current
	Extras       map[string]string
	Wait         bool
	WaitForDebug bool
}

// StartActivity starts an activity on the device
func (c *Client) StartActivity(serial string, opts StartActivityOptions) (bool, error) {
	transport, err := c.Transport(serial)
	if err != nil {
		return false, err
	}
	defer transport.Close()

	cmd := buildActivityCommand(opts, "am start")
	if _, err := transport.Write([]byte(fmt.Sprintf("%04x%s", len(cmd), cmd))); err != nil {
		return false, err
	}

	reply := make([]byte, 4)
	if _, err := transport.Read(reply); err != nil {
		return false, err
	}
	if string(reply) == "OKAY" {
		data, _ := io.ReadAll(transport)
		output := string(data)
		// Check if the command contains "No user" error, retry with user=null
		if strings.Contains(output, "No user") {
			opts.User = 0
			return c.StartActivity(serial, opts)
		}
		return strings.Contains(output, "Error") == false, nil
	}
	if string(reply) == "FAIL" {
		msg, _ := readLengthPrefixed(transport)
		return false, fmt.Errorf("startActivity failed: %s", string(msg))
	}
	return false, fmt.Errorf("unexpected reply: %s", string(reply))
}

// StartServiceOptions holds options for starting a service
type StartServiceOptions struct {
	Action    string
	Data      string
	Type      string
	Component string
	User      int
	Extras    map[string]string
}

// StartService starts a service on the device
func (c *Client) StartService(serial string, opts StartServiceOptions) (bool, error) {
	if opts.User == 0 && (opts.User == 0 || opts.User == -1) {
		// Keep user=0 if explicitly set
	} else if opts.User == 0 {
		opts.User = 0 // Default to primary user
	}

	transport, err := c.Transport(serial)
	if err != nil {
		return false, err
	}
	defer transport.Close()

	cmd := buildServiceCommand(opts, "am startservice")
	if _, err := transport.Write([]byte(fmt.Sprintf("%04x%s", len(cmd), cmd))); err != nil {
		return false, err
	}

	reply := make([]byte, 4)
	if _, err := transport.Read(reply); err != nil {
		return false, err
	}
	if string(reply) == "OKAY" {
		data, _ := io.ReadAll(transport)
		output := string(data)
		// Check if the command contains "No user" error, retry with user=null
		if strings.Contains(output, "--user") {
			// Retry with user option removed
			opts.User = -1
			return c.StartService(serial, opts)
		}
		return strings.Contains(output, "Error") == false, nil
	}
	if string(reply) == "FAIL" {
		msg, _ := readLengthPrefixed(transport)
		return false, fmt.Errorf("startService failed: %s", string(msg))
	}
	return false, fmt.Errorf("unexpected reply: %s", string(reply))
}

// buildActivityCommand builds the activity manager command
func buildActivityCommand(opts StartActivityOptions, baseCmd string) string {
	var buf strings.Builder
	buf.WriteString(baseCmd)

	if opts.Action != "" {
		buf.WriteString(" -a ")
		buf.WriteString(opts.Action)
	}

	if opts.Data != "" {
		buf.WriteString(" -d ")
		buf.WriteString(opts.Data)
	}

	if opts.Type != "" {
		buf.WriteString(" -t ")
		buf.WriteString(opts.Type)
	}

	for _, cat := range opts.Category {
		buf.WriteString(" -c ")
		buf.WriteString(cat)
	}

	if opts.Component != "" {
		buf.WriteString(" -n ")
		buf.WriteString(opts.Component)
	}

	for _, flag := range opts.Flags {
		buf.WriteString(" -f ")
		buf.WriteString(flag)
	}

	for key, val := range opts.Extras {
		buf.WriteString(" --es ")
		buf.WriteString(key)
		buf.WriteString(" ")
		buf.WriteString(val)
	}

	if opts.Wait {
		buf.WriteString(" -W")
	}

	if opts.WaitForDebug {
		buf.WriteString(" -D")
	}

	return fmt.Sprintf("shell:%s", buf.String())
}

// buildServiceCommand builds the service start command
func buildServiceCommand(opts StartServiceOptions, baseCmd string) string {
	var buf strings.Builder
	buf.WriteString(baseCmd)

	if opts.Action != "" {
		buf.WriteString(" -a ")
		buf.WriteString(opts.Action)
	}

	if opts.Data != "" {
		buf.WriteString(" -d ")
		buf.WriteString(opts.Data)
	}

	if opts.Type != "" {
		buf.WriteString(" -t ")
		buf.WriteString(opts.Type)
	}

	if opts.Component != "" {
		buf.WriteString(" -n ")
		buf.WriteString(opts.Component)
	}

	for key, val := range opts.Extras {
		buf.WriteString(" --es ")
		buf.WriteString(key)
		buf.WriteString(" ")
		buf.WriteString(val)
	}

	return fmt.Sprintf("shell:%s", buf.String())
}
