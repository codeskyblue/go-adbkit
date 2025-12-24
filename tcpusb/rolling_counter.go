package tcpusb

// RollingCounter provides a rolling counter with min/max bounds
type RollingCounter struct {
	max uint32
	min uint32
	now uint32
}

// NewRollingCounter creates a new rolling counter
func NewRollingCounter(max uint32) *RollingCounter {
	return &RollingCounter{
		max: max,
		min: 1,
		now: 1,
	}
}

// Next returns the next value in the counter
func (rc *RollingCounter) Next() uint32 {
	if rc.now >= rc.max {
		rc.now = rc.min
	} else {
		rc.now++
	}
	return rc.now
}
