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

func TestListForwards(t *testing.T) {
	testdata := `
> 2025/12/30 20:32:09.000172082  length=21 from=0 to=20
 30 30 31 31 68 6f 73 74 3a 6c 69 73 74 2d 66 6f  0011host:list-fo
 72 77 61 72 64                                   rward
--
< 2025/12/30 20:32:09.000172633  length=4 from=0 to=3
 4f 4b 41 59                                      OKAY
--
< 2025/12/30 20:32:09.000173547  length=46 from=4 to=49
 30 30 32 61 52 46 43 54 38 30 41 53 58 58 50 20  002aRFCT80ASXXP
 74 63 70 3a 38 38 39 39 20 6c 6f 63 61 6c 61 62  tcp:8899 localab
 73 74 72 61 63 74 3a 73 63 72 63 70 79 0a        stract:scrcpy.
--`

	client := NewTestClient(testdata)
	forwards, err := client.ListForwards()
	if err != nil {
		t.Fatalf("ListForwards() error = %v", err)
	}
	if err := client.Conn.CheckRequest(); err != nil {
		t.Fatalf("CheckRequest error = %v", err)
	}

	// Check return values match response
	if len(forwards) != 1 {
		t.Fatalf("ListForwards() returned %d entries, want 1", len(forwards))
	}

	entry := forwards[0]
	if entry.Serial != "RFCT80ASXXP" {
		t.Errorf("Serial = %v, want %v", entry.Serial, "RFCT80ASXXP")
	}
	if entry.Local != "tcp:8899" {
		t.Errorf("Local = %v, want %v", entry.Local, "tcp:8899")
	}
	if entry.Remote != "localabstract:scrcpy" {
		t.Errorf("Remote = %v, want %v", entry.Remote, "localabstract:scrcpy")
	}
}

func TestListForwardsEmpty(t *testing.T) {
	testdata := `
> 2025/12/30 20:32:09.000172082  length=21 from=0 to=20
 30 30 31 31 68 6f 73 74 3a 6c 69 73 74 2d 66 6f  0011host:list-fo
 72 77 61 72 64                                   rward
--
< 2025/12/30 20:32:09.000172633  length=4 from=0 to=3
 4f 4b 41 59                                      OKAY
--
< 2025/12/30 20:32:09.000173547  length=4 from=4 to=7
 30 30 30 30                                      0000
--`

	client := NewTestClient(testdata)
	forwards, err := client.ListForwards()
	if err != nil {
		t.Fatalf("ListForwards() error = %v", err)
	}
	if err := client.Conn.CheckRequest(); err != nil {
		t.Fatalf("CheckRequest error = %v", err)
	}

	if len(forwards) != 0 {
		t.Errorf("ListForwards() = %v, want empty list", forwards)
	}
}
