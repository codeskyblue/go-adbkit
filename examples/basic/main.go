package main

import (
	"flag"
	"fmt"
	"image/png"
	"log"
	"os"
	"strings"

	"github.com/codeskyblue/go-adbkit/adb"
)

// cli program
// support the following commands
// $0 devices
// $0 kill-server
// $0 shell <command>
// $0 screenshot [-s <serial>] [-o <output file>]

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	// Create ADB client
	client := adb.NewClient()

	switch command {
	case "devices":
		listDevices(client)
	case "kill-server":
		killServer(client)
	case "shell":
		handleShell(client, os.Args[2:])
	case "screenshot":
		handleScreenshot(client, os.Args[2:])
	case "-h", "--help", "help":
		printUsage()
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("ADB command-line tool")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  basic devices                - List connected devices")
	fmt.Println("  basic kill-server             - Kill ADB server")
	fmt.Println("  basic shell <command>        - Execute shell command")
	fmt.Println("  basic screenshot [options]   - Take screenshot")
	fmt.Println()
	fmt.Println("Shell Command:")
	fmt.Println("  -s <serial>    Device serial number (auto-detect if only one device)")
	fmt.Println("  <command>      Shell command to execute")
	fmt.Println()
	fmt.Println("Screenshot Options:")
	fmt.Println("  -s <serial>    Device serial number (auto-detect if only one device)")
	fmt.Println("  -o <output>    Output file path (default: screenshot.png)")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  basic devices")
	fmt.Println("  basic kill-server")
	fmt.Println("  basic shell getprop ro.product.model")
	fmt.Println("  basic shell -s emulator-5554 ls /sdcard")
	fmt.Println("  basic screenshot                           # Auto-detect device")
	fmt.Println("  basic screenshot -s emulator-5554           # Specify device")
	fmt.Println("  basic screenshot -s emulator-5554 -o screen.png")
}

func listDevices(client *adb.Client) {
	devices, err := client.ListDevices()
	if err != nil {
		log.Fatalf("Failed to list devices: %v", err)
	}

	if len(devices) == 0 {
		fmt.Println("No devices found")
		return
	}

	fmt.Printf("Found %d device(s):\n", len(devices))
	for _, dev := range devices {
		fmt.Printf("  %s\t%s\n", dev.Serial, dev.State)
	}
}

func killServer(client *adb.Client) {
	ok, err := client.Kill()
	if err != nil {
		log.Fatalf("Failed to kill server: %v", err)
	}

	if ok {
		fmt.Println("ADB server killed successfully")
	} else {
		fmt.Println("Failed to kill ADB server")
	}
}

// autoDetectDevice auto-detects device serial if not specified
// Returns the device serial to use
func autoDetectDevice(client *adb.Client, serial *string) {
	if *serial != "" {
		return
	}

	devices, err := client.ListDevices()
	if err != nil {
		log.Fatalf("Failed to list devices: %v", err)
	}

	if len(devices) == 0 {
		fmt.Println("Error: No devices found")
		os.Exit(1)
	}

	if len(devices) > 1 {
		fmt.Println("Error: Multiple devices found, please specify serial with -s")
		fmt.Println()
		fmt.Println("Available devices:")
		for _, dev := range devices {
			fmt.Printf("  %s\t%s\n", dev.Serial, dev.State)
		}
		os.Exit(1)
	}

	*serial = devices[0].Serial
}

func handleShell(client *adb.Client, args []string) {
	fs := flag.NewFlagSet("shell", flag.ExitOnError)
	serial := fs.String("s", "", "Device serial number (auto-detect if only one device)")

	if err := fs.Parse(args); err != nil {
		log.Fatalf("Failed to parse arguments: %v", err)
	}

	// Get remaining args as command
	commandArgs := fs.Args()
	if len(commandArgs) == 0 {
		fmt.Println("Error: Shell command is required")
		fmt.Println()
		fs.Usage()
		fmt.Println()
		fmt.Println("Example: basic shell getprop ro.product.model")
		fmt.Println("         basic shell -s emulator-5554 ls /sdcard")
		os.Exit(1)
	}

	command := strings.Join(commandArgs, " ")

	// Auto-detect device if serial not specified
	autoDetectDevice(client, serial)

	// New API: Use Device with Shell command
	device := client.Device(adb.DeviceWithSerial(*serial))
	output, err := device.RunCommand(command)
	if err != nil {
		log.Fatalf("Failed to execute shell command: %v", err)
	}

	fmt.Print(output)
}

func handleScreenshot(client *adb.Client, args []string) {
	fs := flag.NewFlagSet("screenshot", flag.ExitOnError)
	serial := fs.String("s", "", "Device serial number (auto-detect if only one device)")
	output := fs.String("o", "screenshot.png", "Output file path")

	if err := fs.Parse(args); err != nil {
		log.Fatalf("Failed to parse arguments: %v", err)
	}

	// Auto-detect device if serial not specified
	if *serial == "" {
		autoDetectDevice(client, serial)
		fmt.Printf("Auto-detected device: %s\n", *serial)
	}

	fmt.Printf("Taking screenshot from device %s...\n", *serial)

	// New API: Use Device with Screencap command
	device := client.Device(adb.DeviceWithSerial(*serial))
	img, err := device.Screencap()
	if err != nil {
		log.Fatalf("Failed to take screenshot: %v", err)
	}

	// Save image to file
	f, err := os.Create(*output)
	if err != nil {
		log.Fatalf("Failed to create output file: %v", err)
	}
	defer f.Close()

	if err := png.Encode(f, img); err != nil {
		log.Fatalf("Failed to encode image: %v", err)
	}

	fmt.Printf("Screenshot saved to: %s\n", *output)
}
