package tcpusb

import (
	"encoding/binary"
	"fmt"
)

// ADB protocol commands
const (
	A_SYNC = 0x434e5953
	A_CNXN = 0x4e584e43
	A_OPEN = 0x4e45504f
	A_OKAY = 0x59414b4f
	A_CLSE = 0x45534c43
	A_WRTE = 0x45545257
	A_AUTH = 0x48545541
)

// Packet represents an ADB protocol packet
type Packet struct {
	Command uint32
	Arg0    uint32
	Arg1    uint32
	Length  uint32
	Check   uint32
	Magic   uint32
	Data    []byte
}

// Checksum calculates the checksum of data
func Checksum(data []byte) uint32 {
	var sum uint32
	for _, b := range data {
		sum += uint32(b)
	}
	return sum
}

// Magic calculates the magic value for a command
func Magic(command uint32) uint32 {
	return command ^ 0xffffffff
}

// Assemble creates a new packet with the given parameters
func Assemble(command, arg0, arg1 uint32, data []byte) []byte {
	var length uint32
	var check uint32
	if data != nil {
		length = uint32(len(data))
		check = Checksum(data)
	}

	chunk := make([]byte, 24+length)
	binary.LittleEndian.PutUint32(chunk[0:4], command)
	binary.LittleEndian.PutUint32(chunk[4:8], arg0)
	binary.LittleEndian.PutUint32(chunk[8:12], arg1)
	binary.LittleEndian.PutUint32(chunk[12:16], length)
	binary.LittleEndian.PutUint32(chunk[16:20], check)
	binary.LittleEndian.PutUint32(chunk[20:24], Magic(command))

	if data != nil {
		copy(chunk[24:], data)
	}

	return chunk
}

// Swap32 swaps the byte order of a uint32
func Swap32(n uint32) uint32 {
	return (n>>24)&0xff |
		(n>>8)&0xff00 |
		(n<<8)&0xff0000 |
		(n << 24)
}

// VerifyChecksum verifies the packet checksum
func (p *Packet) VerifyChecksum() bool {
	return p.Check == Checksum(p.Data)
}

// VerifyMagic verifies the packet magic value
func (p *Packet) VerifyMagic() bool {
	return p.Magic == Magic(p.Command)
}

// String returns a string representation of the packet
func (p *Packet) String() string {
	var cmdType string
	switch p.Command {
	case A_SYNC:
		cmdType = "SYNC"
	case A_CNXN:
		cmdType = "CNXN"
	case A_OPEN:
		cmdType = "OPEN"
	case A_OKAY:
		cmdType = "OKAY"
	case A_CLSE:
		cmdType = "CLSE"
	case A_WRTE:
		cmdType = "WRTE"
	case A_AUTH:
		cmdType = "AUTH"
	default:
		cmdType = fmt.Sprintf("UNKNOWN(0x%08x)", p.Command)
	}
	return fmt.Sprintf("%s arg0=%d arg1=%d length=%d", cmdType, p.Arg0, p.Arg1, p.Length)
}
