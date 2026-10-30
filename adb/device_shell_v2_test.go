package adb

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"strings"
	"testing"
)

type scriptPacket struct {
	request bool
	data    []byte
}

func scriptRequest(command string) scriptPacket {
	return scriptPacket{
		request: true,
		data:    []byte(fmt.Sprintf("%04x%s", len(command), command)),
	}
}

func scriptResponse(data []byte) scriptPacket {
	return scriptPacket{data: data}
}

func buildScript(packets ...scriptPacket) string {
	var b strings.Builder
	for _, p := range packets {
		if p.request {
			b.WriteString("> request\n")
		} else {
			b.WriteString("< response\n")
		}
		b.WriteString(fmt.Sprintf("% x", p.data))
		b.WriteString("\n")
	}
	return b.String()
}

func shellV2Frame(id byte, payload []byte) []byte {
	frame := make([]byte, shellV2HeaderSize+len(payload))
	frame[0] = id
	binary.LittleEndian.PutUint32(frame[1:], uint32(len(payload)))
	copy(frame[shellV2HeaderSize:], payload)
	return frame
}

func joinShellV2Frames(frames ...[]byte) []byte {
	return bytes.Join(frames, nil)
}

func TestRunCommandV2(t *testing.T) {
	const serial = "08a3d291"
	frames := joinShellV2Frames(
		shellV2Frame(shellV2IDStdout, []byte("/\n")),
		shellV2Frame(shellV2IDStderr, []byte("warning: test\n")),
		shellV2Frame(shellV2IDExit, []byte{0}),
	)

	client := NewTestClient(buildScript(
		scriptRequest("host:transport:"+serial),
		scriptResponse([]byte(StatusOkay)),
		scriptRequest("shell,v2,raw:pwd"),
		scriptResponse([]byte(StatusOkay)),
		scriptResponse(frames),
	))
	device := client.Device(DeviceWithSerial(serial))

	result, err := device.RunCommandV2("pwd")
	if err != nil {
		t.Fatalf("RunCommandV2() error = %v", err)
	}
	if err := client.Conn.CheckRequest(); err != nil {
		t.Fatalf("CheckRequest error = %v", err)
	}

	if got := string(result.Stdout); got != "/\n" {
		t.Errorf("Stdout = %q, want %q", got, "/\n")
	}
	if got := string(result.Stderr); got != "warning: test\n" {
		t.Errorf("Stderr = %q, want %q", got, "warning: test\n")
	}
	if result.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", result.ExitCode)
	}
}

func TestRunCommandV2ExitCode(t *testing.T) {
	const serial = "08a3d291"
	frames := joinShellV2Frames(
		shellV2Frame(shellV2IDStderr, []byte("not found\n")),
		shellV2Frame(shellV2IDExit, []byte{127}),
	)

	client := NewTestClient(buildScript(
		scriptRequest("host:transport:"+serial),
		scriptResponse([]byte(StatusOkay)),
		scriptRequest("shell,v2,raw:missing"),
		scriptResponse([]byte(StatusOkay)),
		scriptResponse(frames),
	))
	device := client.Device(DeviceWithSerial(serial))

	result, err := device.RunCommandV2("missing")
	if err != nil {
		t.Fatalf("RunCommandV2() error = %v", err)
	}
	if result.ExitCode != 127 {
		t.Errorf("ExitCode = %d, want 127", result.ExitCode)
	}
}

func TestRunCommandV2ContextCancelled(t *testing.T) {
	const serial = "08a3d291"
	frames := joinShellV2Frames(
		shellV2Frame(shellV2IDExit, []byte{0}),
	)

	client := NewTestClient(buildScript(
		scriptRequest("host:transport:"+serial),
		scriptResponse([]byte(StatusOkay)),
		scriptRequest("shell,v2,raw:pwd"),
		scriptResponse([]byte(StatusOkay)),
		scriptResponse(frames),
	))
	device := client.Device(DeviceWithSerial(serial))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := device.RunCommandV2Context(ctx, "pwd")
	if err != context.Canceled {
		t.Errorf("RunCommandV2Context() error = %v, want %v", err, context.Canceled)
	}
}

func TestShellV2WithOptions(t *testing.T) {
	const serial = "08a3d291"
	service := "shell,v2,TERM=xterm-256color,pty:sh"
	frames := joinShellV2Frames(
		shellV2Frame(shellV2IDStdout, []byte("$ ")),
		shellV2Frame(shellV2IDExit, []byte{0}),
	)

	client := NewTestClient(buildScript(
		scriptRequest("host:transport:"+serial),
		scriptResponse([]byte(StatusOkay)),
		scriptRequest(service),
		scriptResponse([]byte(StatusOkay)),
		scriptResponse(frames),
	))
	device := client.Device(DeviceWithSerial(serial))

	session, err := device.ShellV2WithOptions("sh", ShellV2Options{
		Term: "xterm-256color",
		PTY:  true,
	})
	if err != nil {
		t.Fatalf("ShellV2WithOptions() error = %v", err)
	}
	defer session.Close()

	output, err := io.ReadAll(session.Stdout())
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if got := string(output); got != "$ " {
		t.Errorf("stdout = %q, want %q", got, "$ ")
	}

	exitCode, err := session.Wait()
	if err != nil {
		t.Fatalf("Wait() error = %v", err)
	}
	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0", exitCode)
	}

	if err := client.Conn.CheckRequest(); err != nil {
		t.Fatalf("CheckRequest error = %v", err)
	}
}

func TestShellV2Unsupported(t *testing.T) {
	const serial = "08a3d291"

	client := NewTestClient(buildScript(
		scriptRequest("host:transport:"+serial),
		scriptResponse([]byte(StatusOkay)),
		scriptRequest("shell,v2,raw:pwd"),
		scriptResponse([]byte(StatusFail+"000bunsupported")),
	))
	device := client.Device(DeviceWithSerial(serial))

	_, err := device.ShellV2("pwd")
	if err == nil {
		t.Fatal("ShellV2() should return an error when the device does not support shell v2")
	}
	if !strings.Contains(err.Error(), "unsupported") {
		t.Errorf("error = %v, want it to contain %q", err, "unsupported")
	}
}

func TestBuildShellV2Service(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		opts     ShellV2Options
		expected string
	}{
		{
			name:     "raw default",
			command:  "ls /sdcard",
			expected: "shell,v2,raw:ls /sdcard",
		},
		{
			name:     "term",
			command:  "sh",
			opts:     ShellV2Options{Term: "xterm-256color"},
			expected: "shell,v2,TERM=xterm-256color,raw:sh",
		},
		{
			name:     "pty",
			command:  "sh",
			opts:     ShellV2Options{Term: "xterm", PTY: true},
			expected: "shell,v2,TERM=xterm,pty:sh",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := buildShellV2Service(tt.command, tt.opts); got != tt.expected {
				t.Errorf("buildShellV2Service() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestWriteShellV2Packet(t *testing.T) {
	var buf bytes.Buffer
	if err := writeShellV2Packet(&buf, shellV2IDStdin, []byte("hello")); err != nil {
		t.Fatalf("writeShellV2Packet() error = %v", err)
	}

	expected := []byte{shellV2IDStdin, 5, 0, 0, 0, 'h', 'e', 'l', 'l', 'o'}
	if !bytes.Equal(buf.Bytes(), expected) {
		t.Errorf("packet = %v, want %v", buf.Bytes(), expected)
	}
}

func TestReadShellV2Packet(t *testing.T) {
	data := shellV2Frame(shellV2IDStdout, []byte("hello"))

	id, payload, err := readShellV2Packet(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("readShellV2Packet() error = %v", err)
	}
	if id != shellV2IDStdout {
		t.Errorf("id = %d, want %d", id, shellV2IDStdout)
	}
	if got := string(payload); got != "hello" {
		t.Errorf("payload = %q, want %q", got, "hello")
	}
}

func TestShellV2SessionStdin(t *testing.T) {
	clientConn, deviceConn := net.Pipe()
	defer clientConn.Close()

	session := &ShellV2Session{
		transport: NewTransport(deviceConn),
		stdout:    newShellV2Stream(),
		stderr:    newShellV2Stream(),
		done:      make(chan struct{}),
	}
	go session.readLoop()

	// Capture the packet written to the remote side.
	packetCh := make(chan []byte, 1)
	go func() {
		id, payload, err := readShellV2Packet(clientConn)
		if err != nil {
			packetCh <- nil
			return
		}
		packetCh <- shellV2Frame(id, payload)
	}()

	if _, err := session.Write([]byte("hello")); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	expected := shellV2Frame(shellV2IDStdin, []byte("hello"))
	if got := <-packetCh; !bytes.Equal(got, expected) {
		t.Errorf("stdin packet = %v, want %v", got, expected)
	}

	// Send the exit packet so the read loop terminates.
	if _, err := clientConn.Write(shellV2Frame(shellV2IDExit, []byte{0})); err != nil {
		t.Fatalf("write exit packet error = %v", err)
	}
	if _, err := session.Wait(); err != nil {
		t.Fatalf("Wait() error = %v", err)
	}
}
