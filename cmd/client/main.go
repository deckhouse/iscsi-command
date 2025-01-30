package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/deckhouse/iscsi-command/config"
	pb "github.com/deckhouse/iscsi-command/pkg/iscsi-command"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const defaultConfigPath = "config.yaml"

func main() {
	// Command-line flags
	configFile := flag.String("config", defaultConfigPath, "Path to configuration file")
	flag.Parse()

	// Load configuration
	cfg, err := config.LoadConfig(*configFile)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Validate socket path
	if cfg.SocketPath == "" {
		log.Fatalf("Unix socket path is required in config.yaml")
	}

	// Connect via Unix socket
	socketAddress := fmt.Sprintf("unix://%s", cfg.SocketPath)
	log.Printf("Connecting via Unix socket: %s", cfg.SocketPath)

	conn, err := grpc.Dial(
		socketAddress,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(), // Wait until connection is established
	)
	if err != nil {
		log.Fatalf("Failed to connect to gRPC server: %v", err)
	}
	defer conn.Close()

	client := pb.NewCommandExecutorClient(conn)

	// Create request
	req := &pb.CommandRequest{
		Command:       "iscsi-ls",
		Portal:        "192.168.1.100",           // Example iSCSI portal
		InitiatorName: "iqn.1993-08.org.debian",  // Example initiator name
		TargetIQN:     "iqn.2023-01.com.example", // Example target IQN
		AuthLogin:     "user",                    // Example authentication login
		AuthPassword:  "password",                // Example authentication password
	}

	// Set request timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	// Execute gRPC request
	log.Println("Sending request to gRPC server...")
	resp, err := client.Execute(ctx, req)
	if err != nil {
		log.Fatalf("Error while calling Execute: %v", err)
	}

	// Handle response
	log.Println("Received response from server.")
	log.Printf("Raw Output: %s", resp.Output)
	if resp.Error != "" {
		log.Printf("Server Error: %s", resp.Error)
	} else {
		log.Println("Discovered LUNs:")
		for _, lun := range resp.Luns {
			log.Printf("LUN ID: %d, Size: %s, Vendor: %s, Product: %s, Serial: %s",
				lun.LunID, lun.Size, lun.Vendor, lun.Product, lun.Serial)
		}
	}
}
