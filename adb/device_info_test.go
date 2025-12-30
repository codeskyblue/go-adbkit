package adb

import (
	"testing"
)

// TestParseGetpropOutput tests the parseGetpropOutput function
func TestParseGetpropOutput(t *testing.T) {
	// Sample getprop output - simulates what RunCommand("getprop") would return
	getpropOutput := `[ro.build.version.sdk]: [29]
[ro.product.model]: [Pixel 3]
[ro.product.manufacturer]: [Google]
[ro.build.version.release]: [10]
[persist.sys.locale]: [en-US]
[ro.build.date]: [Thu Jan  2 12:34:56 CST 2020]
`

	props := parseGetpropOutput(getpropOutput)

	// Verify parsed properties
	expectedProps := map[string]string{
		"ro.build.version.sdk":    "29",
		"ro.product.model":        "Pixel 3",
		"ro.product.manufacturer": "Google",
		"ro.build.version.release": "10",
		"persist.sys.locale":      "en-US",
		"ro.build.date":           "Thu Jan  2 12:34:56 CST 2020",
	}

	for key, expectedValue := range expectedProps {
		if gotValue, ok := props[key]; !ok {
			t.Errorf("parseGetpropOutput() missing key %q", key)
		} else if gotValue != expectedValue {
			t.Errorf("parseGetpropOutput()[%q] = %q, want %q", key, gotValue, expectedValue)
		}
	}

	if len(props) != len(expectedProps) {
		t.Errorf("parseGetpropOutput() returned %d items, want %d", len(props), len(expectedProps))
	}
}

// TestParseGetpropOutputEmpty tests parseGetpropOutput with empty output
func TestParseGetpropOutputEmpty(t *testing.T) {
	getpropOutput := ``
	props := parseGetpropOutput(getpropOutput)

	if len(props) != 0 {
		t.Errorf("parseGetpropOutput() returned %d items, want 0", len(props))
	}
}

// TestParseGetpropOutputInvalidLines tests parseGetpropOutput with invalid/malformed lines
func TestParseGetpropOutputInvalidLines(t *testing.T) {
	getpropOutput := `[ro.build.version.sdk]: [29]
invalid line without brackets
[another.key]: [value]
[malformed line
[ro.product.model]: [Pixel 3]
`

	props := parseGetpropOutput(getpropOutput)

	// Should only parse valid lines
	expectedProps := map[string]string{
		"ro.build.version.sdk": "29",
		"another.key":          "value",
		"ro.product.model":     "Pixel 3",
	}

	for key, expectedValue := range expectedProps {
		if gotValue, ok := props[key]; !ok {
			t.Errorf("parseGetpropOutput() missing key %q", key)
		} else if gotValue != expectedValue {
			t.Errorf("parseGetpropOutput()[%q] = %q, want %q", key, gotValue, expectedValue)
		}
	}

	if len(props) != len(expectedProps) {
		t.Errorf("parseGetpropOutput() returned %d items, want %d", len(props), len(expectedProps))
	}
}

