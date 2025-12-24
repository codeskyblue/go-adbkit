package tcpusb

import (
	"encoding/binary"
	"fmt"
	"io"
)

// PacketReader reads ADB packets from a stream
type PacketReader struct {
	stream io.Reader
	buffer []byte
	inBody bool
	packet *Packet
}

// NewPacketReader creates a new packet reader
func NewPacketReader(stream io.Reader) *PacketReader {
	return &PacketReader{
		stream: stream,
		buffer: make([]byte, 0),
		inBody: false,
	}
}

// ReadPacket reads the next packet from the stream
func (pr *PacketReader) ReadPacket() (*Packet, error) {
	for {
		if pr.inBody {
			// Read packet body
			if len(pr.buffer) < int(pr.packet.Length) {
				if err := pr.appendChunk(); err != nil {
					return nil, err
				}
				continue
			}

			pr.packet.Data = make([]byte, pr.packet.Length)
			copy(pr.packet.Data, pr.buffer[:pr.packet.Length])
			pr.buffer = pr.buffer[pr.packet.Length:]

			if !pr.packet.VerifyChecksum() {
				return nil, fmt.Errorf("checksum mismatch")
			}

			pr.inBody = false
			return pr.packet, nil
		} else {
			// Read packet header
			if len(pr.buffer) < 24 {
				if err := pr.appendChunk(); err != nil {
					return nil, err
				}
				continue
			}

			header := pr.buffer[:24]
			pr.packet = &Packet{
				Command: binary.LittleEndian.Uint32(header[0:4]),
				Arg0:    binary.LittleEndian.Uint32(header[4:8]),
				Arg1:    binary.LittleEndian.Uint32(header[8:12]),
				Length:  binary.LittleEndian.Uint32(header[12:16]),
				Check:   binary.LittleEndian.Uint32(header[16:20]),
				Magic:   binary.LittleEndian.Uint32(header[20:24]),
			}
			pr.buffer = pr.buffer[24:]

			if !pr.packet.VerifyMagic() {
				return nil, fmt.Errorf("magic value mismatch")
			}

			if pr.packet.Length == 0 {
				return pr.packet, nil
			}

			pr.inBody = true
		}
	}
}

func (pr *PacketReader) appendChunk() error {
	chunk := make([]byte, 4096)
	n, err := pr.stream.Read(chunk)
	if err != nil {
		return err
	}
	if n > 0 {
		pr.buffer = append(pr.buffer, chunk[:n]...)
	}
	return nil
}
