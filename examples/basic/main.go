package main

import (
	"log"

	"github.com/codeskyblue/adbkit/tcpusb"
)

func main() {
	// Example 1: Simple usage with default settings
	log.Println("Example 1: Simple bridge")
	bridge := tcpusb.NewBridge("your-device-serial")

	server, err := bridge.StartWithServer()
	if err != nil {
		log.Fatal(err)
	}
	defer server.Close()

	// Example 2: Custom port and ADB settings
	log.Println("\nExample 2: Bridge with custom settings")
	bridge2 := tcpusb.NewBridge("another-device-serial")
	bridge2.Config.Port = 7000
	bridge2.Config.ADBHost = "192.168.1.100"
	bridge2.Config.ADBPort = 5037

	server2, err := bridge2.StartWithServer()
	if err != nil {
		log.Fatal(err)
	}
	defer server2.Close()

	// Example 3: Custom authentication handler
	log.Println("\nExample 3: Bridge with custom auth")
	bridge3 := tcpusb.NewBridge("secure-device-serial")
	bridge3.Config.Port = 8000
	bridge3.Config.AuthHandler = func(publicKey []byte) error {
		log.Printf("Device attempting to connect with public key: %x", publicKey[:20])
		return nil
	}

	server3, err := bridge3.StartWithServer()
	if err != nil {
		log.Fatal(err)
	}
	defer server3.Close()

	// Keep the program running
	log.Println("\nAll bridges started. Press Ctrl+C to exit.")
	select {}
}
