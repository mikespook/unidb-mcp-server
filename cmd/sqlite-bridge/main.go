package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/mikespook/unidb-mcp/internal/bridge"
)

func main() {
	// Command line flags
	name := flag.String("name", "", "Bridge name (required)")
	filePath := flag.String("file", "", "Path to SQLite file (required)")
	unidbURL := flag.String("unidb", "http://localhost:9093", "UniDB server URL")
	reconnect := flag.Bool("reconnect", true, "Auto-reconnect on connection loss")
	reconnectDelay := flag.Duration("reconnect-delay", 5*time.Second, "Delay between reconnection attempts")
	secret := flag.String("secret", "", "Bridge secret for authentication (auto-generated if not provided)")

	flag.Parse()

	// Validate required flags
	if *name == "" {
		log.Fatal("Error: -name is required")
	}

	if *filePath == "" {
		log.Fatal("Error: -file is required")
	}

	// Check if SQLite file exists
	if _, err := os.Stat(*filePath); os.IsNotExist(err) {
		log.Fatalf("Error: SQLite file not found: %s", *filePath)
	}

	// Generate secret if not provided
	bridgeSecret := *secret
	if bridgeSecret == "" {
		bridgeSecret = uuid.New().String()
		fmt.Printf("Bridge Secret: %s\n", bridgeSecret)
		fmt.Printf("Save this secret — use -secret on next run to reconnect\n\n")
	}

	// Create bridge configuration
	config := bridge.BridgeConfig{
		Name:           *name,
		Secret:         bridgeSecret,
		FilePath:       *filePath,
		UniDBURL:       *unidbURL,
		Reconnect:      *reconnect,
		ReconnectDelay: *reconnectDelay,
	}

	// Create bridge client
	client, err := bridge.NewClient(config)
	if err != nil {
		log.Fatalf("Failed to create bridge client: %v", err)
	}

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start bridge in a goroutine
	errChan := make(chan error, 1)
	go func() {
		fmt.Printf("🌉 SQLite Bridge starting...\n")
		fmt.Printf("   Name: %s\n", config.Name)
		fmt.Printf("   File: %s\n", config.FilePath)
		fmt.Printf("   UniDB: %s\n", config.UniDBURL)
		fmt.Printf("   Reconnect: %v\n", config.Reconnect)
		fmt.Println()

		if err := client.Start(); err != nil {
			errChan <- err
		}
	}()

	// Wait for shutdown signal or error
	select {
	case sig := <-sigChan:
		fmt.Printf("\nReceived signal %v, shutting down...\n", sig)
	case err := <-errChan:
		log.Printf("Bridge error: %v", err)
	}

	// Stop the bridge
	client.Stop()
	fmt.Println("Bridge stopped")
}
