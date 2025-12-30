package adb

import (
	"bufio"
	"io"
	"net"
)


// ReadAllFromConn reads all data from a connection until closed
func ReadAllFromConn(conn net.Conn) ([]byte, error) {
	defer conn.Close()
	return io.ReadAll(bufio.NewReader(conn))
}

// newScanner creates a buffered scanner for reading from a connection
func newScanner(conn net.Conn) *bufio.Scanner {
	return bufio.NewScanner(conn)
}
