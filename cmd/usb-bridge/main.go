package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/codeskyblue/go-adbkit/tcpusb"
)

func main() {
	serial := flag.String("serial", "", "Device serial number (required)")
	port := flag.Int("port", 6174, "Port to listen on")
	adbHost := flag.String("adb-host", "127.0.0.1", "ADB server host")
	adbPort := flag.Int("adb-port", 5037, "ADB server port")
	verbose := flag.Bool("verbose", false, "Enable verbose (debug) logging")
	flag.Parse()

	// Configure slog based on verbose flag
	logLevel := slog.LevelInfo
	if *verbose {
		logLevel = slog.LevelDebug
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: logLevel,
	}))
	slog.SetDefault(logger)

	if *serial == "" {
		slog.Error("Device serial number is required. Use -serial flag")
		os.Exit(1)
	}

	// Create bridge with configuration
	bridge := tcpusb.NewBridge(*serial)
	bridge.Config.Port = *port
	bridge.Config.ADBHost = *adbHost
	bridge.Config.ADBPort = *adbPort

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		slog.Info("Shutting down...")
		os.Exit(0)
	}()

	// Start the bridge
	slog.Info("Starting USB-to-TCP bridge", "device", *serial)
	slog.Info("Connect with", "command", fmt.Sprintf("adb connect localhost:%d", *port))

	if err := bridge.Start(); err != nil {
		slog.Error("Failed to start bridge", "error", err)
		os.Exit(1)
	}
}
