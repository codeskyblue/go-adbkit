package adb

// Device represents a specific Android device
type Device struct {
	client    *Client
	descriptor DeviceDescriptor
}

// String returns the string representation of the device
func (d *Device) String() string {
	return d.descriptor.String()
}
