package main

import (
	"flag"
	"fmt"
	"image/png"
	"io"
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
// $0 push <local> <remote> [-s <serial>]
// $0 pull <remote> <local> [-s <serial>]

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
	case "push":
		handlePush(client, os.Args[2:])
	case "pull":
		handlePull(client, os.Args[2:])
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
	fmt.Println("  basic devices                      - List connected devices")
	fmt.Println("  basic kill-server                   - Kill ADB server")
	fmt.Println("  basic shell <command>              - Execute shell command")
	fmt.Println("  basic screenshot [options]         - Take screenshot")
	fmt.Println("  basic push <local> <remote> [opts]  - Push file to device")
	fmt.Println("  basic pull <remote> <local> [opts]  - Pull file from device")
	fmt.Println()
	fmt.Println("Shell Command:")
	fmt.Println("  -s <serial>    Device serial number (auto-detect if only one device)")
	fmt.Println("  <command>      Shell command to execute")
	fmt.Println()
	fmt.Println("Screenshot Options:")
	fmt.Println("  -s <serial>    Device serial number (auto-detect if only one device)")
	fmt.Println("  -o <output>    Output file path (default: screenshot.png)")
	fmt.Println()
	fmt.Println("Push/Pull Options:")
	fmt.Println("  -s <serial>    Device serial number (auto-detect if only one device)")
	fmt.Println("  -m <mode>      File permissions (octal, e.g., 0644)")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  basic devices")
	fmt.Println("  basic kill-server")
	fmt.Println("  basic shell getprop ro.product.model")
	fmt.Println("  basic shell -s emulator-5554 ls /sdcard")
	fmt.Println("  basic screenshot                           # Auto-detect device")
	fmt.Println("  basic screenshot -s emulator-5554           # Specify device")
	fmt.Println("  basic screenshot -s emulator-5554 -o screen.png")
	fmt.Println("  basic push test.txt /sdcard/test.txt")
	fmt.Println("  basic push test.txt /sdcard/test.txt -s emulator-5554")
	fmt.Println("  basic pull /sdcard/test.txt ./test.txt")
	fmt.Println("  basic pull /sdcard/test.txt ./test.txt -s emulator-5554")
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

func handlePush(client *adb.Client, args []string) {
	fs := flag.NewFlagSet("push", flag.ExitOnError)
	serial := fs.String("s", "", "Device serial number (auto-detect if only one device)")
	mode := fs.Uint64("m", 0644, "File permissions (octal, default: 0644)")

	if err := fs.Parse(args); err != nil {
		log.Fatalf("Failed to parse arguments: %v", err)
	}

	// Get local and remote paths
	paths := fs.Args()
	if len(paths) < 2 {
		fmt.Println("Error: Both local and remote paths are required")
		fmt.Println()
		fs.Usage()
		fmt.Println()
		fmt.Println("Example: basic push test.txt /sdcard/test.txt")
		fmt.Println("         basic push test.txt /sdcard/test.txt -s emulator-5554")
		os.Exit(1)
	}

	localPath := paths[0]
	remotePath := paths[1]

	// Auto-detect device if serial not specified
	autoDetectDevice(client, serial)

	fmt.Printf("Pushing %s to %s on device %s...\n", localPath, remotePath, *serial)

	// Open local file
	f, err := os.Open(localPath)
	if err != nil {
		log.Fatalf("Failed to open local file: %v", err)
	}
	defer f.Close()

	// Push file to device
	device := client.Device(adb.DeviceWithSerial(*serial))
	if err := device.Push(f, remotePath, uint32(*mode)); err != nil {
		log.Fatalf("Failed to push file: %v", err)
	}

	fmt.Println("File pushed successfully")
}

func handlePull(client *adb.Client, args []string) {
	fs := flag.NewFlagSet("pull", flag.ExitOnError)
	serial := fs.String("s", "", "Device serial number (auto-detect if only one device)")

	if err := fs.Parse(args); err != nil {
		log.Fatalf("Failed to parse arguments: %v", err)
	}

	// Get remote and local paths
	paths := fs.Args()
	if len(paths) < 2 {
		fmt.Println("Error: Both remote and local paths are required")
		fmt.Println()
		fs.Usage()
		fmt.Println()
		fmt.Println("Example: basic pull /sdcard/test.txt ./test.txt")
		fmt.Println("         basic pull /sdcard/test.txt ./test.txt -s emulator-5554")
		os.Exit(1)
	}

	remotePath := paths[0]
	localPath := paths[1]

	// Auto-detect device if serial not specified
	autoDetectDevice(client, serial)

	fmt.Printf("Pulling %s from device %s to %s...\n", remotePath, *serial, localPath)

	// Pull file from device
	device := client.Device(adb.DeviceWithSerial(*serial))
	reader, err := device.Pull(remotePath)
	if err != nil {
		log.Fatalf("Failed to pull file: %v", err)
	}
	defer reader.Close()

	// Create local file
	f, err := os.Create(localPath)
	if err != nil {
		log.Fatalf("Failed to create local file: %v", err)
	}
	defer f.Close()

	// Copy data
	if _, err := io.Copy(f, reader); err != nil {
		log.Fatalf("Failed to write file: %v", err)
	}

	fmt.Println("File pulled successfully")
}


