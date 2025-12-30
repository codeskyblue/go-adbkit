package adb

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net"
	"os"
	"strings"
	"time"
)

// GetSerialNo gets the serial number of the device
func (c *Client) GetSerialNo(serial string) (string, error) {
	cmd := fmt.Sprintf("host-serial:%s:get-serialno", serial)
	payload, err := c.SendHostCommand(cmd)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(payload)), nil
}

// GetDevicePath gets the device path
func (c *Client) GetDevicePath(serial string) (string, error) {
	cmd := fmt.Sprintf("host-serial:%s:get-devpath", serial)
	payload, err := c.SendHostCommand(cmd)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(payload)), nil
}

// GetState gets the device state
func (c *Client) GetState(serial string) (string, error) {
	cmd := fmt.Sprintf("host-serial:%s:get-state", serial)
	payload, err := c.SendHostCommand(cmd)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(payload)), nil
}

// GetFeatures retrieves device features
func (c *Client) GetFeatures(serial string) (map[string]string, error) {
	transport, err := c.Transport(serial)
	if err != nil {
		return nil, err
	}
	defer transport.Close()

	cmd := "features:"
	if _, err := transport.Write([]byte(fmt.Sprintf("%04x%s", len(cmd), cmd))); err != nil {
		return nil, err
	}

	reply := make([]byte, 4)
	if _, err := transport.Read(reply); err != nil {
		return nil, err
	}
	if string(reply) == "FAIL" {
		msg, _ := readLengthPrefixed(transport)
		return nil, fmt.Errorf("getFeatures failed: %s", string(msg))
	}
	if string(reply) != "OKAY" {
		return nil, fmt.Errorf("unexpected reply: %s", string(reply))
	}

	payload, err := readLengthPrefixed(transport)
	if err != nil {
		return nil, err
	}

	features := make(map[string]string)
	lines := strings.Split(strings.TrimSpace(string(payload)), "\n")
	for _, line := range lines {
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			features[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}
	return features, nil
}

// GetProperties returns device properties by running `getprop` over a shell transport
func (c *Client) GetProperties(serial string) (map[string]string, error) {
	conn, err := c.Shell(serial, "getprop")
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	data, err := ReadAllFromConn(conn)
	if err != nil {
		return nil, err
	}
	props := make(map[string]string)
	// getprop prints lines like: [key]: [value]
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// naive parse
		if !strings.HasPrefix(line, "[") {
			continue
		}
		// find two bracketed parts
		endKey := strings.Index(line, "]:")
		if endKey == -1 {
			continue
		}
		key := strings.Trim(line[1:endKey], " ")
		// value part
		rest := strings.TrimSpace(line[endKey+2:])
		if strings.HasPrefix(rest, "[") && strings.HasSuffix(rest, "]") {
			rest = rest[1 : len(rest)-1]
		}
		props[key] = rest
	}
	return props, nil
}

// GetDHCPIpAddress returns the DHCP IP address for the given interface (defaults to wlan0)
func (c *Client) GetDHCPIpAddress(serial string, iface string) (string, error) {
	if iface == "" {
		iface = "wlan0"
	}
	props, err := c.GetProperties(serial)
	if err != nil {
		return "", err
	}
	key := fmt.Sprintf("dhcp.%s.ipaddress", iface)
	if ip, ok := props[key]; ok && ip != "" {
		return ip, nil
	}
	return "", fmt.Errorf("unable to find ipaddress for '%s'", iface)
}

// Shell opens a shell stream on the device. The returned net.Conn is ready for read/write of raw data.
// Caller is responsible for closing the connection.
func (c *Client) Shell(serial string, command string) (net.Conn, error) {
	transport, err := c.Transport(serial)
	if err != nil {
		return nil, err
	}

	// Write length-prefixed service name (e.g. "shell:ls -l")
	svc := []byte(command)
	length := fmt.Sprintf("%04x", len(svc))
	if _, err := transport.Write([]byte(length)); err != nil {
		transport.Close()
		return nil, err
	}
	if _, err := transport.Write(svc); err != nil {
		transport.Close()
		return nil, err
	}

	// The transport is now a live stream for shell output
	return transport, nil
}

// RunCommand runs a shell command on the device and returns the output as a string
func (c *Client) RunCommand(serial string, command string) (string, error) {
	return c.RunCommandContext(context.Background(), serial, command)
}

// RunCommandContext runs a shell command on the device with context support and returns the output as a string
func (c *Client) RunCommandContext(ctx context.Context, serial string, command string) (string, error) {
	conn, err := c.Shell(serial, command)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	// Read output with context support
	data, err := readAllWithContext(ctx, conn)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

// Reboot reboots the device
func (c *Client) Reboot(serial string) (bool, error) {
	transport, err := c.Transport(serial)
	if err != nil {
		return false, err
	}
	defer transport.Close()

	cmd := "reboot:"
	if _, err := transport.Write([]byte(fmt.Sprintf("%04x%s", len(cmd), cmd))); err != nil {
		return false, err
	}

	reply := make([]byte, 4)
	if _, err := transport.Read(reply); err != nil {
		return false, err
	}
	if string(reply) == "OKAY" {
		// read remainder (if any) then return true
		io.ReadAll(transport)
		return true, nil
	}
	if string(reply) == "FAIL" {
		msg, _ := readLengthPrefixed(transport)
		return false, fmt.Errorf("reboot failed: %s", string(msg))
	}
	return false, fmt.Errorf("unexpected reply: %s", string(reply))
}

// Remount remounts the device filesystem as read-write
func (c *Client) Remount(serial string) (bool, error) {
	transport, err := c.Transport(serial)
	if err != nil {
		return false, err
	}
	defer transport.Close()

	cmd := "remount:"
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
		return false, fmt.Errorf("remount failed: %s", string(msg))
	}
	return false, fmt.Errorf("unexpected reply: %s", string(reply))
}

// Root attempts to restart adbd as root
func (c *Client) Root(serial string) (bool, error) {
	transport, err := c.Transport(serial)
	if err != nil {
		return false, err
	}
	defer transport.Close()

	cmd := "root:"
	if _, err := transport.Write([]byte(fmt.Sprintf("%04x%s", len(cmd), cmd))); err != nil {
		return false, err
	}
	reply := make([]byte, 4)
	if _, err := transport.Read(reply); err != nil {
		return false, err
	}
	if string(reply) == "OKAY" {
		// read all output and check for success message
		data, _ := io.ReadAll(transport)
		out := string(data)
		if strings.Contains(out, "restarting adbd as root") {
			return true, nil
		}
		return false, fmt.Errorf("%s", strings.TrimSpace(out))
	}
	if string(reply) == "FAIL" {
		msg, _ := readLengthPrefixed(transport)
		return false, fmt.Errorf("root failed: %s", string(msg))
	}
	return false, fmt.Errorf("unexpected reply: %s", string(reply))
}

// WaitForDevice waits for the device to be available
func (c *Client) WaitForDevice(serial string) (bool, error) {
	payload, err := c.SendHostCommand(fmt.Sprintf("host-serial:%s:wait-for-device", serial))
	if err != nil {
		return false, err
	}
	return len(payload) == 0, nil
}

// WaitForDeviceWithTimeout waits for a device with a timeout
func (c *Client) WaitForDeviceWithTimeout(serial string, timeout time.Duration) error {
	timeoutChan := time.After(timeout)
	errChan := make(chan error, 1)

	go func() {
		_, err := c.WaitForDevice(serial)
		errChan <- err
	}()

	select {
	case err := <-errChan:
		return err
	case <-timeoutChan:
		return fmt.Errorf("wait for device timed out")
	}
}

// WaitBootComplete waits for the device to finish booting
func (c *Client) WaitBootComplete(serial string) (bool, error) {
	transport, err := c.Transport(serial)
	if err != nil {
		return false, err
	}
	defer transport.Close()

	cmd := "shell:getprop sys.boot_completed"
	if _, err := transport.Write([]byte(fmt.Sprintf("%04x%s", len(cmd), cmd))); err != nil {
		return false, err
	}

	reply := make([]byte, 4)
	if _, err := transport.Read(reply); err != nil {
		return false, err
	}
	if string(reply) == "OKAY" {
		data, _ := io.ReadAll(transport)
		return strings.Contains(string(data), "1"), nil
	}
	if string(reply) == "FAIL" {
		msg, _ := readLengthPrefixed(transport)
		return false, fmt.Errorf("waitBootComplete failed: %s", string(msg))
	}
	return false, fmt.Errorf("unexpected reply: %s", string(reply))
}

// TrackJdwp starts tracking jdwp pids — returns the live connection stream for reading updates
func (c *Client) TrackJdwp(serial string) (net.Conn, error) {
	transport, err := c.Transport(serial)
	if err != nil {
		return nil, err
	}
	cmd := "track-jdwp"
	if _, err := transport.Write([]byte(fmt.Sprintf("%04x%s", len(cmd), cmd))); err != nil {
		transport.Close()
		return nil, err
	}
	reply := make([]byte, 4)
	if _, err := transport.Read(reply); err != nil {
		transport.Close()
		return nil, err
	}
	if string(reply) == "OKAY" {
		// transport is now a stream of newline-separated pid lists
		return transport, nil
	}
	if string(reply) == "FAIL" {
		msg, _ := readLengthPrefixed(transport)
		transport.Close()
		return nil, fmt.Errorf("track-jdwp failed: %s", string(msg))
	}
	transport.Close()
	return nil, fmt.Errorf("unexpected reply: %s", string(reply))
}

// Framebuffer requests the device framebuffer and returns an image.Image
func (c *Client) Framebuffer(serial string) (image.Image, error) {
	transport, err := c.Transport(serial)
	if err != nil {
		return nil, err
	}
	defer transport.Close()

	cmd := "framebuffer:"
	if _, err := transport.Write([]byte(fmt.Sprintf("%04x%s", len(cmd), cmd))); err != nil {
		return nil, err
	}

	reply := make([]byte, 4)
	if _, err := transport.Read(reply); err != nil {
		return nil, err
	}

	if string(reply) == "FAIL" {
		msg, _ := readLengthPrefixed(transport)
		return nil, fmt.Errorf("framebuffer failed: %s", string(msg))
	}

	if string(reply) != "OKAY" {
		return nil, fmt.Errorf("unexpected reply: %s", string(reply))
	}

	// Read header
	header := make([]byte, 52)
	if _, err := io.ReadFull(transport, header); err != nil {
		return nil, err
	}

	// Parse metadata
	meta := make(map[string]uint32)
	offset := 0
	fields := []string{"version", "bpp", "size", "width", "height", "red_offset", "red_length", "blue_offset", "blue_length", "green_offset", "green_length", "alpha_offset", "alpha_length"}
	for i := 0; i < len(fields); i++ {
		meta[fields[i]] = binary.LittleEndian.Uint32(header[offset : offset+4])
		offset += 4
	}

	width := int(meta["width"])
	height := int(meta["height"])
	bpp := int(meta["bpp"])

	// Read pixel data
	pixelData := make([]byte, meta["size"])
	if _, err := io.ReadFull(transport, pixelData); err != nil {
		return nil, err
	}

	// Create image based on format
	return decodeFramebuffer(pixelData, width, height, bpp, meta)
}

// Screencap runs screencap and returns an image.Image
func (c *Client) Screencap(serial string) (image.Image, error) {
	// Use shell-based screencap command
	conn, err := c.Shell(serial, "screencap -p")
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	// Read PNG data
	data, err := io.ReadAll(conn)
	if err != nil {
		return nil, fmt.Errorf("failed to read screencap data: %w", err)
	}
	dumper := hex.Dumper(os.Stdout)
	dumper.Write(data[:64])
	dumper.Close()

	// Decode the image
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to decode screencap image: %w", err)
	}

	return img, nil
}

// OpenLogcat starts `logcat` via shell and returns the live connection
func (c *Client) OpenLogcat(serial string, options string) (net.Conn, error) {
	// options may be additional args passed to logcat
	cmd := "logcat"
	if options != "" {
		cmd = cmd + " " + options
	}
	return c.Shell(serial, cmd)
}

// ReadAllFromConn reads all data from a connection until closed
func ReadAllFromConn(conn net.Conn) ([]byte, error) {
	defer conn.Close()
	return io.ReadAll(bufio.NewReader(conn))
}

// readAllWithContext reads all data from a connection with context support
func readAllWithContext(ctx context.Context, conn net.Conn) ([]byte, error) {
	type result struct {
		data []byte
		err  error
	}

	resultChan := make(chan result, 1)

	go func() {
		data, err := io.ReadAll(bufio.NewReader(conn))
		resultChan <- result{data, err}
	}()

	select {
	case <-ctx.Done():
		conn.Close()
		return nil, ctx.Err()
	case res := <-resultChan:
		return res.data, res.err
	}
}

// newScanner creates a buffered scanner for reading from a connection
func newScanner(conn net.Conn) *bufio.Scanner {
	return bufio.NewScanner(conn)
}

// isNoUserError checks if an error is related to the --user option
func isNoUserError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "--user")
}

// decodeFramebuffer decodes raw framebuffer data into an image.Image
func decodeFramebuffer(data []byte, width, height, bpp int, meta map[string]uint32) (image.Image, error) {
	// Calculate bytes per pixel
	bytesPerPixel := bpp / 8

	// Create RGBA image
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Get color offsets and lengths
	redOffset := meta["red_offset"]
	redLength := meta["red_length"]
	greenOffset := meta["green_offset"]
	greenLength := meta["green_length"]
	blueOffset := meta["blue_offset"]
	blueLength := meta["blue_length"]
	alphaOffset := meta["alpha_offset"]
	alphaLength := meta["alpha_length"]

	// Create masks for color channels
	redMask := uint32(0xFFFFFFFF>>(32-redLength)) << redOffset
	greenMask := uint32(0xFFFFFFFF>>(32-greenLength)) << greenOffset
	blueMask := uint32(0xFFFFFFFF>>(32-blueLength)) << blueOffset
	alphaMask := uint32(0xFFFFFFFF>>(32-alphaLength)) << alphaOffset

	// Process each pixel
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			offset := (y*width + x) * bytesPerPixel
			pixel := uint32(0)

			// Read pixel value based on bytes per pixel
			for i := 0; i < bytesPerPixel; i++ {
				pixel |= uint32(data[offset+i]) << (i * 8)
			}

			// Extract color components
			red := uint8((pixel & redMask) >> redOffset)
			green := uint8((pixel & greenMask) >> greenOffset)
			blue := uint8((pixel & blueMask) >> blueOffset)
			alpha := uint8(0xFF)

			if alphaLength > 0 {
				alpha = uint8((pixel & alphaMask) >> alphaOffset)
			}

			// Normalize color values to 8-bit
			if redLength < 8 {
				red = red << (8 - redLength)
			}
			if greenLength < 8 {
				green = green << (8 - greenLength)
			}
			if blueLength < 8 {
				blue = blue << (8 - blueLength)
			}
			if alphaLength < 8 && alphaLength > 0 {
				alpha = alpha << (8 - alphaLength)
			}

			img.SetRGBA(x, y, color.RGBA{R: red, G: green, B: blue, A: alpha})
		}
	}

	return img, nil
}
