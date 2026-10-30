package adb

import (
	"strings"
	"testing"
)

func TestDeviceFeaturesQuery(t *testing.T) {
	const serial = "08a3d291"

	client := NewTestClient(buildScript(
		scriptRequest("host-serial:"+serial+":features"),
		scriptResponse([]byte(StatusOkay+"000cshell_v2,cmd")),
	))
	device := client.Device(DeviceWithSerial(serial))

	features, err := device.Features()
	if err != nil {
		t.Fatalf("Features() error = %v", err)
	}
	if err := client.Conn.CheckRequest(); err != nil {
		t.Fatalf("CheckRequest error = %v", err)
	}

	if len(features) != 2 || features[0] != "shell_v2" || features[1] != "cmd" {
		t.Errorf("Features() = %v, want [shell_v2 cmd]", features)
	}
}

func TestHasFeature(t *testing.T) {
	const serial = "08a3d291"

	client := NewTestClient(buildScript(
		scriptRequest("host-serial:"+serial+":features"),
		scriptResponse([]byte(StatusOkay+"000cshell_v2,cmd")),
	))
	device := client.Device(DeviceWithSerial(serial))

	supported, err := device.HasFeature("shell_v2")
	if err != nil {
		t.Fatalf("HasFeature() error = %v", err)
	}
	if !supported {
		t.Error(`HasFeature("shell_v2") = false, want true`)
	}
}

func TestHasFeatureUnsupported(t *testing.T) {
	const serial = "08a3d291"

	client := NewTestClient(buildScript(
		scriptRequest("host-serial:"+serial+":features"),
		scriptResponse([]byte(StatusOkay+"0003cmd")),
	))
	device := client.Device(DeviceWithSerial(serial))

	supported, err := device.HasFeature("shell_v2")
	if err != nil {
		t.Fatalf("HasFeature() error = %v", err)
	}
	if supported {
		t.Error(`HasFeature("shell_v2") = true, want false`)
	}
}

func TestDeviceFeaturesAnyDevice(t *testing.T) {
	const serial = "08a3d291"
	devices := serial + "\tdevice\n"

	client := NewTestClient(buildScript(
		scriptRequest("host:devices"),
		scriptResponse([]byte(StatusOkay+"0010"+devices)),
		scriptRequest("host-serial:"+serial+":features"),
		scriptResponse([]byte(StatusOkay+"000cshell_v2,cmd")),
	))
	device := client.Device(AnyDevice())

	features, err := device.Features()
	if err != nil {
		t.Fatalf("Features() error = %v", err)
	}
	if got := strings.Join(features, ","); got != "shell_v2,cmd" {
		t.Errorf("Features() = %q, want %q", got, "shell_v2,cmd")
	}
}
