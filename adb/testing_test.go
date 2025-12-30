package adb

import (
	"bufio"
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"net"
	"strings"
	"time"
)

type TestClient struct {
	*Client
	Conn *mockConn
}

func NewTestClient(testdata string) *TestClient {
	packets := parseTestData(testdata)
	conn := newMockConn(packets)
	return &TestClient{
		Client: NewClientWithConnector(&mockConnector{conn: conn}),
		Conn:   conn,
	}
}

// Packet represents a single network packet with direction
type Packet struct {
	IsRequest bool
	Data      []byte
}

// mockConn is a mock net.Conn for testing
type mockConn struct {
	packets        []Packet     // All packets for validation
	packetIndex    int          // Current packet index
	lastCheckIndex int          // Packet index at last CheckRequest
	readBuffer     bytes.Buffer // Buffer for reading data
	writeBuffer    bytes.Buffer // Buffer for written data
	closed         bool
}

// newMockConn creates a mock connection from packets
func newMockConn(packets []Packet) *mockConn {
	return &mockConn{
		packets:        packets,
		packetIndex:    0,
		lastCheckIndex: 0,
		closed:         false,
	}
}

// fillReadBuffer fills the read buffer with the next response packet's data
func (m *mockConn) fillReadBuffer() {
	// Skip any request packets
	for m.packetIndex < len(m.packets) && m.packets[m.packetIndex].IsRequest {
		m.packetIndex++
	}

	if m.packetIndex >= len(m.packets) {
		return // No more packets
	}

	packet := m.packets[m.packetIndex]
	if packet.IsRequest {
		return // Next packet is a request, stop here
	}

	// Write the entire response packet data to buffer
	m.readBuffer.Write(packet.Data)
	m.packetIndex++
}

// hexdump returns a hexdump format string of the data
func hexdump(data []byte) string {
	var buf bytes.Buffer
	dumper := hex.Dumper(&buf)
	dumper.Write(data)
	dumper.Close()
	return buf.String()
}

// CheckRequest verifies that written requests match expected requests
// It checks requests from lastCheckIndex to current packetIndex
func (m *mockConn) CheckRequest() error {
	written := m.writeBuffer.Bytes()

	// Collect expected requests in the range [lastCheckIndex, packetIndex)
	var expectedBytes []byte
	for i := m.lastCheckIndex; i < m.packetIndex; i++ {
		if i < len(m.packets) && m.packets[i].IsRequest {
			expectedBytes = append(expectedBytes, m.packets[i].Data...)
		}
	}
	// Check if we have written all expected requests
	if !bytes.Equal(written, expectedBytes) {
		return fmt.Errorf("request mismatch\n\ngot (hexdump):\n%s\n\nwant (hexdump):\n%s",
			hexdump(written), hexdump(expectedBytes))
	}

	// Update lastCheckIndex and clear buffer
	m.lastCheckIndex = m.packetIndex
	m.writeBuffer.Reset()
	return nil
}

func (m *mockConn) Read(b []byte) (n int, err error) {
	if m.readBuffer.Len() == 0 {
		m.fillReadBuffer()
	}
	return m.readBuffer.Read(b)
}

func (m *mockConn) Write(b []byte) (n int, err error) {
	return m.writeBuffer.Write(b)
}

func (m *mockConn) Close() error {
	m.closed = true
	return nil
}

func (m *mockConn) LocalAddr() net.Addr {
	return &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 5037}
}

func (m *mockConn) RemoteAddr() net.Addr {
	return &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 5037}
}

func (m *mockConn) SetDeadline(t time.Time) error {
	return nil
}

func (m *mockConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (m *mockConn) SetWriteDeadline(t time.Time) error {
	return nil
}

// mockConnector implements Connector for testing
type mockConnector struct {
	conn *mockConn
}

func (m *mockConnector) ConnectionContext(ctx context.Context) (net.Conn, error) {
	return m.conn, nil
}

// parseTestData parses testdata format and returns packets
func parseTestData(testdata string) []Packet {
	scanner := bufio.NewScanner(strings.NewReader(testdata))
	var packets []Packet
	var currentPacket []byte
	var currentIsRequest bool // Track isRequest for current packet

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines
		if line == "" {
			continue
		}

		// Check for direction marker - save current packet if exists
		if strings.HasPrefix(line, "<") || strings.HasPrefix(line, ">") {
			if len(currentPacket) > 0 {
				packets = append(packets, Packet{IsRequest: currentIsRequest, Data: currentPacket})
				currentPacket = nil
			}
			currentIsRequest = strings.HasPrefix(line, ">")
			continue
		}

		// Skip header lines
		if strings.Contains(line, "length=") || strings.Contains(line, "from=") {
			continue
		}

		// Parse hex data
		parts := strings.Fields(line)
		for _, part := range parts {
			if len(part) > 24 { // Skip ASCII representation
				continue
			}
			data, err := hex.DecodeString(part)
			if err != nil {
				continue
			}
			currentPacket = append(currentPacket, data...)
		}

		// Check for packet separator
		if strings.HasPrefix(line, "--") {
			packets = append(packets, Packet{IsRequest: currentIsRequest, Data: currentPacket})
			currentPacket = nil
		}
	}

	// Add the last packet if exists
	if len(currentPacket) > 0 {
		packets = append(packets, Packet{IsRequest: currentIsRequest, Data: currentPacket})
	}

	return packets
}
