package adb

import "testing"

func TestVersion(t *testing.T) {
	testdata := `
> 2025/12/30 13:43:07.000525656  length=16 from=0 to=15
 30 30 30 63 68 6f 73 74 3a 76 65 72 73 69 6f 6e  000chost:version
--
< 2025/12/30 13:43:07.000527987  length=12 from=0 to=11
 4f 4b 41 59 30 30 30 34 30 30 32 39              OKAY00040029
--`

	client := NewTestClient(testdata)
	got, err := client.Version()
	if err != nil {
		t.Fatalf("Version() error = %v", err)
	}
	if err := client.Conn.CheckRequest(); err != nil {
		t.Fatalf("CheckRequest error = %v", err)
	}
	// Check return value matches response
	// "0029" as hex = 0x0029 = 41
	expectedVersion := 41
	if got != expectedVersion {
		t.Errorf("Version() = %v (0x%x), want %v (0x%x)", got, got, expectedVersion, expectedVersion)
	}
}

func TestListDevices(t *testing.T) {
	testdata := `
> 2025/12/30 14:58:36.000833574  length=16 from=0 to=15
 30 30 30 63 68 6f 73 74 3a 64 65 76 69 63 65 73  000chost:devices
--
< 2025/12/30 14:58:36.000834907  length=8 from=0 to=7
 4f 4b 41 59 30 30 30 30                          OKAY0000
--`
	client := NewTestClient(testdata)
	devices, err := client.ListDevices()
	if err != nil {
		t.Fatalf("ListDevices() error = %v", err)
	}
	if err := client.Conn.CheckRequest(); err != nil {
		t.Fatalf("CheckRequest error = %v", err)
	}
	if len(devices) != 0 {
		t.Errorf("ListDevices() = %v, want empty list", devices)
	}
}
