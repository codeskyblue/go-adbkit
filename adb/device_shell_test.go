package adb

import "testing"

func TestRunCommand(t *testing.T) {
	testdata := `
> 2025/12/30 23:59:17.000672487  length=27 from=0 to=26
 30 30 31 37 68 6f 73 74 3a 74 72 61 6e 73 70 6f  0017host:transpo
 72 74 3a 30 38 61 33 64 32 39 31                 rt:08a3d291
--
< 2025/12/30 23:59:17.000672690  length=4 from=0 to=3
 4f 4b 41 59                                      OKAY
--
> 2025/12/30 23:59:17.000672863  length=13 from=27 to=39
 30 30 30 39 73 68 65 6c 6c 3a 70 77 64           0009shell:pwd
--
< 2025/12/30 23:59:17.000692129  length=4 from=4 to=7
 4f 4b 41 59                                      OKAY
--
< 2025/12/30 23:59:17.000713357  length=2 from=8 to=9
 2f 0a                                            /.
--`

	client := NewTestClient(testdata)
	device := client.Device(DeviceWithSerial("08a3d291"))

	output, err := device.RunCommand("pwd")
	if err != nil {
		t.Fatalf("RunCommand() error = %v", err)
	}
	if err := client.Conn.CheckRequest(); err != nil {
		t.Fatalf("CheckRequest error = %v", err)
	}

	expectedOutput := "/\n"
	if output != expectedOutput {
		t.Errorf("RunCommand() = %q, want %q", output, expectedOutput)
	}
}
