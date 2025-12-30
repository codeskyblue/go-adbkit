package adb

import "fmt"

// DeviceDescriptorType defines the type of device selector
type DeviceDescriptorType int

const (
	// DeviceAny selects any device (host:transport-any)
	DeviceAny DeviceDescriptorType = iota
	// DeviceSerial selects a device by serial number (host:transport:serial)
	DeviceSerial
	// DeviceUsb selects any USB device (host:transport-usb)
	DeviceUsb
	// DeviceLocal selects any local/emulator device (host:transport-local)
	DeviceLocal
)

// DeviceDescriptor describes how to select a device
type DeviceDescriptor struct {
	descriptorType DeviceDescriptorType
	serial         string
}

// AnyDevice returns a descriptor that selects any device
func AnyDevice() DeviceDescriptor {
	return DeviceDescriptor{descriptorType: DeviceAny}
}

// AnyUsbDevice returns a descriptor that selects any USB device
func AnyUsbDevice() DeviceDescriptor {
	return DeviceDescriptor{descriptorType: DeviceUsb}
}

// AnyLocalDevice returns a descriptor that selects any local/emulator device
func AnyLocalDevice() DeviceDescriptor {
	return DeviceDescriptor{descriptorType: DeviceLocal}
}

// DeviceWithSerial returns a descriptor that selects a device with the specified serial number
func DeviceWithSerial(serial string) DeviceDescriptor {
	return DeviceDescriptor{
		descriptorType: DeviceSerial,
		serial:         serial,
	}
}

// String returns a string representation of the descriptor
func (d DeviceDescriptor) String() string {
	if d.descriptorType == DeviceSerial {
		return fmt.Sprintf("%s[%s]", d.descriptorType, d.serial)
	}
	return d.descriptorType.String()
}

// getHostPrefix returns the host prefix for commands
func (d DeviceDescriptor) getHostPrefix() string {
	switch d.descriptorType {
	case DeviceAny:
		return "host"
	case DeviceUsb:
		return "host-usb"
	case DeviceLocal:
		return "host-local"
	case DeviceSerial:
		return fmt.Sprintf("host-serial:%s", d.serial)
	default:
		panic(fmt.Sprintf("invalid DeviceDescriptorType: %v", d.descriptorType))
	}
}

// getTransportDescriptor returns the transport descriptor for device communication
func (d DeviceDescriptor) getTransportDescriptor() string {
	switch d.descriptorType {
	case DeviceAny:
		return "transport-any"
	case DeviceUsb:
		return "transport-usb"
	case DeviceLocal:
		return "transport-local"
	case DeviceSerial:
		return fmt.Sprintf("transport:%s", d.serial)
	default:
		panic(fmt.Sprintf("invalid DeviceDescriptorType: %v", d.descriptorType))
	}
}

// String returns the string representation of DeviceDescriptorType
func (t DeviceDescriptorType) String() string {
	switch t {
	case DeviceAny:
		return "DeviceAny"
	case DeviceSerial:
		return "DeviceSerial"
	case DeviceUsb:
		return "DeviceUsb"
	case DeviceLocal:
		return "DeviceLocal"
	default:
		return "Unknown"
	}
}
