package tcpusb

import "errors"

var (
	// ErrServiceExists indicates a service with the given ID already exists
	ErrServiceExists = errors.New("service already exists")

	// ErrUnauthorized indicates the connection is not authorized
	ErrUnauthorized = errors.New("unauthorized")

	// ErrAuthFailed indicates authentication failed
	ErrAuthFailed = errors.New("authentication failed")

	// ErrPrematurePacket indicates a packet was received before transport was ready
	ErrPrematurePacket = errors.New("premature packet")

	// ErrLateTransport indicates transport was established too late
	ErrLateTransport = errors.New("late transport")
)
