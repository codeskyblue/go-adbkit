package adb

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net"
)

// Framebuffer requests the device framebuffer and returns an image.Image
func (d *Device) Framebuffer() (image.Image, error) {
	transport, err := d.Transport()
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

	header := make([]byte, 52)
	if _, err := io.ReadFull(transport, header); err != nil {
		return nil, err
	}

	meta := make(map[string]uint32)
	offset := 0
	// The ADB framebuffer header format:
	// version, bpp, reserved, size, height, width, red_offset, red_length, blue_offset, blue_length, green_offset, green_length, alpha_offset
	// Note: alpha_length is not included in the 52-byte header
	fields := []string{"version", "bpp", "reserved", "size", "height", "width", "red_offset", "red_length", "blue_offset", "blue_length", "green_offset", "green_length", "alpha_offset"}
	for i := 0; i < len(fields); i++ {
		meta[fields[i]] = binary.LittleEndian.Uint32(header[offset : offset+4])
		offset += 4
	}

	width := int(meta["width"])
	height := int(meta["height"])
	bpp := int(meta["bpp"])

	// Note: alpha_length is not included in the header, default to 0
	meta["alpha_length"] = 0

	pixelData := make([]byte, meta["size"])
	if _, err := io.ReadFull(transport, pixelData); err != nil {
		return nil, err
	}

	return decodeFramebuffer(pixelData, width, height, bpp, meta)
}

// Screencap runs screencap and returns an image.Image
func (d *Device) Screencap() (image.Image, error) {
	conn, err := d.Shell("screencap -p")
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	data, err := io.ReadAll(conn)
	if err != nil {
		return nil, fmt.Errorf("failed to read screencap data: %w", err)
	}

	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to decode screencap image: %w", err)
	}

	return img, nil
}

// OpenLogcat starts `logcat` via shell and returns the live connection
func (d *Device) OpenLogcat(options string) (net.Conn, error) {
	cmd := "logcat"
	if options != "" {
		cmd = cmd + " " + options
	}
	return d.Shell(cmd)
}

// decodeFramebuffer decodes raw framebuffer data into an image.Image
func decodeFramebuffer(data []byte, width, height, bpp int, meta map[string]uint32) (image.Image, error) {
	bytesPerPixel := bpp / 8

	img := image.NewRGBA(image.Rect(0, 0, width, height))

	redOffset := meta["red_offset"]
	redLength := meta["red_length"]
	greenOffset := meta["green_offset"]
	greenLength := meta["green_length"]
	blueOffset := meta["blue_offset"]
	blueLength := meta["blue_length"]
	alphaOffset := meta["alpha_offset"]
	alphaLength := meta["alpha_length"]

	redMask := uint32(0xFFFFFFFF>>(32-redLength)) << redOffset
	greenMask := uint32(0xFFFFFFFF>>(32-greenLength)) << greenOffset
	blueMask := uint32(0xFFFFFFFF>>(32-blueLength)) << blueOffset
	alphaMask := uint32(0xFFFFFFFF>>(32-alphaLength)) << alphaOffset

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			offset := (y*width + x) * bytesPerPixel
			pixel := uint32(0)

			for i := 0; i < bytesPerPixel; i++ {
				pixel |= uint32(data[offset+i]) << (i * 8)
			}

			red := uint8((pixel & redMask) >> redOffset)
			green := uint8((pixel & greenMask) >> greenOffset)
			blue := uint8((pixel & blueMask) >> blueOffset)
			alpha := uint8(0xFF)

			if alphaLength > 0 {
				alpha = uint8((pixel & alphaMask) >> alphaOffset)
			}

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
