package adb

import (
	"strings"
	"testing"
)

func TestDevicePush(t *testing.T) {
	testdata := `
> 2025/12/30 23:31:21.000298416  length=27 from=0 to=26
 30 30 31 37 68 6f 73 74 3a 74 72 61 6e 73 70 6f  0017host:transpo
 72 74 3a 30 38 61 33 64 32 39 31                 rt:08a3d291
--
< 2025/12/30 23:31:21.000298636  length=4 from=0 to=3
 4f 4b 41 59                                      OKAY
--
> 2025/12/30 23:31:21.000298799  length=9 from=27 to=35
 30 30 30 35 73 79 6e 63 3a                       0005sync:
--
< 2025/12/30 23:31:21.000301763  length=4 from=4 to=7
 4f 4b 41 59                                      OKAY
--
> 2025/12/30 23:31:21.000301932  length=39 from=36 to=74
 53 45 4e 44 1f 00 00 00 2f 64 61 74 61 2f 6c 6f  SEND..../data/lo
 63 61 6c 2f 74 6d 70 2f 68 65 6c 6c 6f 2e 74 78  cal/tmp/hello.tx
 74 2c 33 33 31 38 38                             t,33188
--
> 2025/12/30 23:31:21.000302342  length=22 from=75 to=96
 44 41 54 41 06 00 00 00 68 65 6c 6c 6f 0a        DATA....hello.
 44 4f 4e 45 00 00 00 00                          DONE....
--
< 2025/12/30 23:31:21.000305734  length=8 from=8 to=15
 4f 4b 41 59 00 00 00 00                          OKAY....
--`

	client := NewTestClient(testdata)
	device := client.Device(DeviceWithSerial("08a3d291"))

	// Test content
	content := "hello\n"
	reader := strings.NewReader(content)
	remotePath := "/data/local/tmp/hello.txt"
	mode := uint32(33188) // File mode from testdata

	err := device.Push(reader, remotePath, mode)
	if err != nil {
		t.Fatalf("Push() error = %v", err)
	}
	if err := client.Conn.CheckRequest(); err != nil {
		t.Fatalf("CheckRequest error = %v", err)
	}
}
