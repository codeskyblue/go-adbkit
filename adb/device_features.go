package adb

import (
	"context"
	"fmt"
	"strings"
)

// Features returns the features advertised by the device.
//
// Unlike Client.Features, which returns the features of the ADB server, this
// queries the feature set of the selected device (host-serial:<serial>:features).
func (d *Device) Features() ([]string, error) {
	return d.FeaturesContext(context.Background())
}

// FeaturesContext returns the features advertised by the device with context
// support.
func (d *Device) FeaturesContext(ctx context.Context) ([]string, error) {
	prefix := d.descriptor.getHostPrefix()

	// host:features without a device prefix returns the server's features, not
	// the device's. Resolve an actual device first when no selector was given.
	if d.descriptor.descriptorType == DeviceAny {
		serial, err := d.firstDeviceSerial()
		if err != nil {
			return nil, err
		}
		prefix = "host-serial:" + serial
	}

	payload, err := d.client.SendHostCommandContext(ctx, prefix+":features")
	if err != nil {
		return nil, err
	}

	raw := strings.TrimSpace(string(payload))
	if raw == "" {
		return []string{}, nil
	}
	return strings.Split(raw, ","), nil
}

// HasFeature reports whether the device advertises the given feature.
func (d *Device) HasFeature(feature string) (bool, error) {
	features, err := d.Features()
	if err != nil {
		return false, err
	}
	for _, f := range features {
		if f == feature {
			return true, nil
		}
	}
	return false, nil
}

func (d *Device) firstDeviceSerial() (string, error) {
	devices, err := d.client.ListDevices()
	if err != nil {
		return "", err
	}

	// Prefer an online device over unauthorized/offline entries.
	for _, device := range devices {
		if device.State == "device" {
			return device.Serial, nil
		}
	}
	if len(devices) > 0 {
		return devices[0].Serial, nil
	}
	return "", fmt.Errorf("adb: no devices found")
}
