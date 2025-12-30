package adb

import (
	"fmt"
	"io"
	"strings"
)

// StartActivityOptions holds options for starting an activity
type StartActivityOptions struct {
	Action       string
	Data         string
	Type         string
	Category     []string
	Component    string
	Flags        []string
	User         int
	Extras       map[string]string
	Wait         bool
	WaitForDebug bool
}

// StartActivity starts an activity on the device
func (d *Device) StartActivity(opts StartActivityOptions) (bool, error) {
	transport, err := d.Transport()
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
		if strings.Contains(output, "No user") {
			opts.User = 0
			return d.StartActivity(opts)
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
func (d *Device) StartService(opts StartServiceOptions) (bool, error) {
	if opts.User == 0 && (opts.User == 0 || opts.User == -1) {
	} else if opts.User == 0 {
		opts.User = 0
	}

	transport, err := d.Transport()
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
		if strings.Contains(output, "--user") {
			opts.User = -1
			return d.StartService(opts)
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
	args := []string{baseCmd}

	if opts.Action != "" {
		args = append(args, "-a", opts.Action)
	}

	if opts.Data != "" {
		args = append(args, "-d", opts.Data)
	}

	if opts.Type != "" {
		args = append(args, "-t", opts.Type)
	}

	for _, cat := range opts.Category {
		args = append(args, "-c", cat)
	}

	if opts.Component != "" {
		args = append(args, "-n", opts.Component)
	}

	for _, flag := range opts.Flags {
		args = append(args, "-f", flag)
	}

	for key, val := range opts.Extras {
		args = append(args, "--es", key, val)
	}

	if opts.Wait {
		args = append(args, "-W")
	}

	if opts.WaitForDebug {
		args = append(args, "-D")
	}

	return fmt.Sprintf("shell:%s", strings.Join(args, " "))
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
