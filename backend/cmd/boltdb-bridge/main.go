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
	name := flag.String("name", "", "Bridge name (required)")
	filePath := flag.String("file", "", "Path to BoltDB file (required)")
	unidbURL := flag.String("unidb", "http://localhost:9093", "UniDB server URL")
	reconnect := flag.Bool("reconnect", true, "Auto-reconnect on connection loss")
	reconnectDelay := flag.Duration("reconnect-delay", 5*time.Second, "Delay between reconnection attempts")
	secret := flag.String("secret", "", "Bridge secret for authentication (auto-generated if not provided)")

	flag.Parse()

	if *name == "" {
		log.Fatal("Error: -name is required")
	}

	if *filePath == "" {
		log.Fatal("Error: -file is required")
	}

	if _, err := os.Stat(*filePath); os.IsNotExist(err) {
		log.Fatalf("Error: BoltDB file not found: %s", *filePath)
	}

	bridgeSecret := *secret
	if bridgeSecret == "" {
		bridgeSecret = uuid.New().String()
		fmt.Printf("Bridge Secret: %s\n", bridgeSecret)
		fmt.Printf("Save this secret — use -secret on next run to reconnect\n\n")
	}

	config := bridge.BridgeConfig{
		Name:           *name,
		Secret:         bridgeSecret,
		FilePath:       *filePath,
		UniDBURL:       *unidbURL,
		Reconnect:      *reconnect,
		ReconnectDelay: *reconnectDelay,
	}

	client, err := bridge.NewBoltDBClient(config)
	if err != nil {
		log.Fatalf("Failed to create bridge client: %v", err)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	errChan := make(chan error, 1)
	go func() {
		fmt.Printf("BoltDB Bridge starting...\n")
		fmt.Printf("   Name: %s\n", config.Name)
		fmt.Printf("   File: %s\n", config.FilePath)
		fmt.Printf("   UniDB: %s\n", config.UniDBURL)
		fmt.Printf("   Reconnect: %v\n", config.Reconnect)
		fmt.Println()

		if err := client.Start(); err != nil {
			errChan <- err
		}
	}()

	select {
	case sig := <-sigChan:
		fmt.Printf("\nReceived signal %v, shutting down...\n", sig)
	case err := <-errChan:
		log.Printf("Bridge error: %v", err)
	}

	client.Stop()
	fmt.Println("Bridge stopped")
}
