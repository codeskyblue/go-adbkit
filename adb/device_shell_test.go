package adb

import (
	"context"
	"testing"
	"time"
)

const shellCommandTestData = `
> 2025/12/30 23:59:17.000672487  length=27 from=0 to=26
 30 30 31 37 68 6f 73 74 3a 74 72 61 6e 73 70 6f  0017host:transpo
 72 74 3a 30 38 61 33 64 32 39 31                 rt:08a3d291
--
< 2025/12/30 23:59:17.000672690  length=4 from=0 to=3
 4f 4b 41 59                                      OKAY
--
> 2025/12/30 23:59:17.000672863  length=13 from=27 to=39
 30 30 30 39 73 68 65 6c 6c 3a 70 77 64           0009shell:pwd
--
< 2025/12/30 23:59:17.000692129  length=4 from=4 to=7
 4f 4b 41 59                                      OKAY
--
< 2025/12/30 23:59:17.000713357  length=2 from=8 to=9
 2f 0a                                            /.
--`

func TestRunCommand(t *testing.T) {
	client := NewTestClient(shellCommandTestData)
	device := client.Device(DeviceWithSerial("08a3d291"))

	output, err := device.RunCommand("pwd")
	if err != nil {
		t.Fatalf("RunCommand() error = %v", err)
	}
	if err := client.Conn.CheckRequest(); err != nil {
		t.Fatalf("CheckRequest error = %v", err)
	}

	expectedOutput := "/\n"
	if output != expectedOutput {
		t.Errorf("RunCommand() = %q, want %q", output, expectedOutput)
	}
}

func TestRunCommandContextSuccess(t *testing.T) {
	client := NewTestClient(shellCommandTestData)
	device := client.Device(DeviceWithSerial("08a3d291"))

	ctx := context.Background()
	output, err := device.RunCommandContext(ctx, "pwd")
	if err != nil {
		t.Fatalf("RunCommandContext() error = %v", err)
	}
	if err := client.Conn.CheckRequest(); err != nil {
		t.Fatalf("CheckRequest error = %v", err)
	}

	expectedOutput := "/\n"
	if output != expectedOutput {
		t.Errorf("RunCommandContext() = %q, want %q", output, expectedOutput)
	}
}

func TestRunCommandContextCancel(t *testing.T) {
	client := NewTestClient(shellCommandTestData)
	device := client.Device(DeviceWithSerial("08a3d291"))

	// Create a context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := device.RunCommandContext(ctx, "pwd")
	if err == nil {
		t.Fatal("RunCommandContext() with cancelled context should return error")
	}
	if err != context.Canceled {
		t.Errorf("RunCommandContext() error = %v, want %v", err, context.Canceled)
	}
}

func TestRunCommandContextTimeout(t *testing.T) {
	client := NewTestClient(shellCommandTestData)
	device := client.Device(DeviceWithSerial("08a3d291"))

	// Create a context with a very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// Give it a moment to expire
	time.Sleep(10 * time.Millisecond)

	_, err := device.RunCommandContext(ctx, "pwd")
	if err == nil {
		t.Fatal("RunCommandContext() with expired timeout should return error")
	}
	if err != context.DeadlineExceeded {
		t.Errorf("RunCommandContext() error = %v, want %v", err, context.DeadlineExceeded)
	}
}
