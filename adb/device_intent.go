package adb

import (
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
	args := buildActivityCommandArgs(opts, "am start")
	command := strings.Join(args, " ")

	output, err := d.RunCommand(command)
	if err != nil {
		return false, err
	}

	// Handle user parameter retry
	if strings.Contains(output, "No user") {
		opts.User = 0
		return d.StartActivity(opts)
	}

	return !strings.Contains(output, "Error"), nil
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
	args := buildServiceCommandArgs(opts, "am startservice")
	command := strings.Join(args, " ")

	output, err := d.RunCommand(command)
	if err != nil {
		return false, err
	}

	// Handle user parameter retry
	if strings.Contains(output, "--user") {
		opts.User = -1
		return d.StartService(opts)
	}

	return !strings.Contains(output, "Error"), nil
}

// buildActivityCommandArgs builds the activity command arguments as a string slice
func buildActivityCommandArgs(opts StartActivityOptions, baseCmd string) []string {
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

	return args
}

// buildServiceCommandArgs builds the service command arguments as a string slice
func buildServiceCommandArgs(opts StartServiceOptions, baseCmd string) []string {
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

	if opts.Component != "" {
		args = append(args, "-n", opts.Component)
	}

	for key, val := range opts.Extras {
		args = append(args, "--es", key, val)
	}

	return args
}
