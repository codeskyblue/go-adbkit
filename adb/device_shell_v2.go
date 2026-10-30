package adb

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"strings"
	"sync"
)

// Shell protocol v2 packet IDs.
const (
	shellV2IDStdin            byte = 0
	shellV2IDStdout           byte = 1
	shellV2IDStderr           byte = 2
	shellV2IDExit             byte = 3
	shellV2IDCloseStdin       byte = 4
	shellV2IDWindowSizeChange byte = 5
)

const (
	// shellV2HeaderSize is the size of a shell protocol v2 packet header:
	// 1 byte packet ID followed by a 4 byte little-endian payload length.
	shellV2HeaderSize = 5
	// shellV2MaxPayload bounds the size of a single packet to avoid
	// allocating huge buffers for malformed data.
	shellV2MaxPayload = 16 * 1024 * 1024
	// shellV2WriteChunkSize is the maximum stdin payload sent per packet.
	// The device side splits larger packets anyway, but keeping chunks small
	// avoids large temporary buffers.
	shellV2WriteChunkSize = 64 * 1024
)

// ShellV2Options configures a shell protocol v2 session.
type ShellV2Options struct {
	// Term sets the TERM environment variable on the remote shell.
	// An empty value leaves it unset.
	Term string
	// PTY allocates a pseudo-terminal for the session. When false (the
	// default), the command runs in raw mode. Raw mode is required to keep
	// stdout and stderr separate and to receive a real exit code.
	PTY bool
}

// ShellV2Result contains the result of a shell protocol v2 command.
type ShellV2Result struct {
	Stdout   []byte
	Stderr   []byte
	ExitCode int
}

// ShellV2Session is an interactive shell protocol v2 session.
//
// Output is buffered internally, so stdout and stderr can be consumed at any
// time without blocking the session. Write writes data to the remote stdin.
type ShellV2Session struct {
	transport *Transport
	writeMu   sync.Mutex
	closeOnce sync.Once

	stdout *shellV2Stream
	stderr *shellV2Stream

	done     chan struct{}
	gotExit  bool
	exitCode int
	err      error
}

// ShellV2 opens an interactive shell v2 session in raw mode.
func (d *Device) ShellV2(command string) (*ShellV2Session, error) {
	return d.ShellV2Context(context.Background(), command)
}

// ShellV2Context opens an interactive shell v2 session in raw mode with
// context support.
func (d *Device) ShellV2Context(ctx context.Context, command string) (*ShellV2Session, error) {
	return d.ShellV2WithOptionsContext(ctx, command, ShellV2Options{})
}

// ShellV2WithOptions opens an interactive shell v2 session with custom options.
func (d *Device) ShellV2WithOptions(command string, opts ShellV2Options) (*ShellV2Session, error) {
	return d.ShellV2WithOptionsContext(context.Background(), command, opts)
}

// ShellV2WithOptionsContext opens an interactive shell v2 session with custom
// options and context support.
func (d *Device) ShellV2WithOptionsContext(ctx context.Context, command string, opts ShellV2Options) (*ShellV2Session, error) {
	transport, err := d.TransportContext(ctx)
	if err != nil {
		return nil, err
	}

	status, err := transport.SendCommand(buildShellV2Service(command, opts))
	if err != nil {
		transport.Close()
		return nil, err
	}
	if status != StatusOkay {
		msg, _ := readLengthPrefixed(transport)
		transport.Close()
		if len(msg) > 0 {
			return nil, fmt.Errorf("shell v2 failed: %s", string(msg))
		}
		return nil, fmt.Errorf("unexpected status: %s", status)
	}

	session := &ShellV2Session{
		transport: transport,
		stdout:    newShellV2Stream(),
		stderr:    newShellV2Stream(),
		done:      make(chan struct{}),
	}
	go session.readLoop()
	return session, nil
}

// buildShellV2Service builds a shell service string such as
// "shell,v2,raw:ls /sdcard" or "shell,v2,TERM=xterm-256color,pty:sh".
func buildShellV2Service(command string, opts ShellV2Options) string {
	args := []string{"shell", "v2"}
	if opts.Term != "" {
		args = append(args, "TERM="+opts.Term)
	}
	if opts.PTY {
		args = append(args, "pty")
	} else {
		args = append(args, "raw")
	}
	return strings.Join(args, ",") + ":" + command
}

// Stdout returns the reader for the remote stdout stream.
func (s *ShellV2Session) Stdout() io.Reader {
	return s.stdout
}

// Stderr returns the reader for the remote stderr stream.
func (s *ShellV2Session) Stderr() io.Reader {
	return s.stderr
}

// Write sends data to the remote stdin.
func (s *ShellV2Session) Write(p []byte) (int, error) {
	written := 0
	for len(p) > 0 {
		n := len(p)
		if n > shellV2WriteChunkSize {
			n = shellV2WriteChunkSize
		}
		if err := s.writePacket(shellV2IDStdin, p[:n]); err != nil {
			return written, err
		}
		written += n
		p = p[n:]
	}
	return written, nil
}

// CloseStdin closes the remote stdin, allowing commands that read until EOF
// to finish.
func (s *ShellV2Session) CloseStdin() error {
	return s.writePacket(shellV2IDCloseStdin, nil)
}

// Resize notifies the remote PTY of a new window size. It is only meaningful
// when the session was created with ShellV2Options.PTY set.
func (s *ShellV2Session) Resize(rows, cols uint16) error {
	payload := append([]byte(fmt.Sprintf("%dx%d,0x0", rows, cols)), 0)
	return s.writePacket(shellV2IDWindowSizeChange, payload)
}

// Wait blocks until the remote command exits and returns its exit code.
func (s *ShellV2Session) Wait() (int, error) {
	<-s.done
	if s.err != nil {
		return s.exitCode, s.err
	}
	if !s.gotExit {
		return s.exitCode, io.ErrUnexpectedEOF
	}
	return s.exitCode, nil
}

// Close closes the session connection.
func (s *ShellV2Session) Close() error {
	return s.closeTransport()
}

func (s *ShellV2Session) writePacket(id byte, payload []byte) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	return writeShellV2Packet(s.transport, id, payload)
}

func (s *ShellV2Session) closeTransport() error {
	var err error
	s.closeOnce.Do(func() {
		err = s.transport.Close()
	})
	return err
}

func (s *ShellV2Session) readLoop() {
	defer close(s.done)
	defer s.closeTransport()
	defer s.stdout.close()
	defer s.stderr.close()

	for {
		id, payload, err := readShellV2Packet(s.transport)
		if err != nil {
			if err != io.EOF {
				s.err = err
			}
			return
		}

		switch id {
		case shellV2IDStdout:
			s.stdout.write(payload)
		case shellV2IDStderr:
			s.stderr.write(payload)
		case shellV2IDExit:
			s.gotExit = true
			if len(payload) > 0 {
				s.exitCode = int(payload[0])
			}
			return
		}
	}
}

// shellV2Stream is an unbounded, goroutine-safe buffer that implements
// io.Reader. Writers never block waiting for readers.
type shellV2Stream struct {
	mu     sync.Mutex
	cond   *sync.Cond
	buf    bytes.Buffer
	closed bool
}

func newShellV2Stream() *shellV2Stream {
	s := &shellV2Stream{}
	s.cond = sync.NewCond(&s.mu)
	return s
}

func (s *shellV2Stream) Read(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for s.buf.Len() == 0 && !s.closed {
		s.cond.Wait()
	}
	if s.buf.Len() == 0 {
		return 0, io.EOF
	}
	return s.buf.Read(p)
}

func (s *shellV2Stream) write(p []byte) {
	if len(p) == 0 {
		return
	}
	s.mu.Lock()
	s.buf.Write(p)
	s.cond.Broadcast()
	s.mu.Unlock()
}

func (s *shellV2Stream) close() {
	s.mu.Lock()
	s.closed = true
	s.cond.Broadcast()
	s.mu.Unlock()
}

// RunCommandV2 runs a shell command using the shell protocol v2 and returns
// stdout, stderr and the remote exit code.
func (d *Device) RunCommandV2(command string) (*ShellV2Result, error) {
	return d.RunCommandV2Context(context.Background(), command)
}

// RunCommandV2Context runs a shell command using the shell protocol v2 with
// context support.
func (d *Device) RunCommandV2Context(ctx context.Context, command string) (*ShellV2Result, error) {
	session, err := d.ShellV2WithOptionsContext(ctx, command, ShellV2Options{})
	if err != nil {
		return nil, err
	}
	defer session.Close()

	type readResult struct {
		data []byte
		err  error
	}

	stdoutCh := make(chan readResult, 1)
	stderrCh := make(chan readResult, 1)

	go func() {
		data, err := io.ReadAll(session.Stdout())
		stdoutCh <- readResult{data: data, err: err}
	}()
	go func() {
		data, err := io.ReadAll(session.Stderr())
		stderrCh <- readResult{data: data, err: err}
	}()

	// Abort the session promptly if the context is cancelled or times out.
	go func() {
		select {
		case <-ctx.Done():
			session.Close()
		case <-session.done:
		}
	}()

	stdout := <-stdoutCh
	stderr := <-stderrCh
	exitCode, waitErr := session.Wait()

	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if stdout.err != nil {
		return nil, stdout.err
	}
	if stderr.err != nil {
		return nil, stderr.err
	}
	if waitErr != nil {
		return nil, waitErr
	}

	return &ShellV2Result{
		Stdout:   stdout.data,
		Stderr:   stderr.data,
		ExitCode: exitCode,
	}, nil
}

// readShellV2Packet reads a single shell protocol v2 packet.
func readShellV2Packet(r io.Reader) (byte, []byte, error) {
	header := make([]byte, shellV2HeaderSize)
	if _, err := io.ReadFull(r, header); err != nil {
		return 0, nil, err
	}

	length := binary.LittleEndian.Uint32(header[1:])
	if length > shellV2MaxPayload {
		return 0, nil, fmt.Errorf("shell v2 packet too large: %d bytes", length)
	}
	if length == 0 {
		return header[0], nil, nil
	}

	payload := make([]byte, length)
	if _, err := io.ReadFull(r, payload); err != nil {
		return 0, nil, err
	}
	return header[0], payload, nil
}

// writeShellV2Packet writes a single shell protocol v2 packet.
func writeShellV2Packet(w io.Writer, id byte, payload []byte) error {
	header := make([]byte, shellV2HeaderSize)
	header[0] = id
	binary.LittleEndian.PutUint32(header[1:], uint32(len(payload)))

	if err := writeAll(w, header); err != nil {
		return err
	}
	return writeAll(w, payload)
}

func writeAll(w io.Writer, p []byte) error {
	for len(p) > 0 {
		n, err := w.Write(p)
		if err != nil {
			return err
		}
		if n == 0 {
			return io.ErrShortWrite
		}
		p = p[n:]
	}
	return nil
}
