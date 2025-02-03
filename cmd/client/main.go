package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/deckhouse/iscsi-command/internal/logger"
	"github.com/sirupsen/logrus"
	"time"

	"github.com/deckhouse/iscsi-command/config"
	pb "github.com/deckhouse/iscsi-command/pkg/iscsi-command"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const defaultConfigPath = "config.yaml"

func main() {
	logger.Init()
	log := logger.Log

	log.Info("Starting client...")

	// Command-line flags
	configFile := flag.String("config", defaultConfigPath, "Path to configuration file")
	flag.Parse()

	// Load configuration
	cfg, err := config.LoadConfig(*configFile)
	if err != nil {
		log.WithError(err).Fatal("Failed to load configuration")
	}

	// Validate socket path
	if cfg.SocketPath == "" {
		log.Fatal("Unix socket path is required in config.yaml")
	}

	// Connect via Unix socket
	socketAddress := fmt.Sprintf("unix://%s", cfg.SocketPath)
	log.WithField("socketPath", cfg.SocketPath).Info("Connecting via Unix socket")

	conn, err := grpc.Dial(
		socketAddress,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(), // Wait until connection is established
	)
	if err != nil {
		log.WithError(err).Fatal("Failed to connect to gRPC server")
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
	log.Info("Sending request to gRPC server...")
	resp, err := client.Execute(ctx, req)
	if err != nil {
		log.WithError(err).Fatal("Error while calling Execute")
	}

	// Handle response
	log.Info("Received response from server.")
	log.WithField("rawOutput", resp.Output).Info("Command output received")
	if resp.Error != "" {
		log.WithField("serverError", resp.Error).Warn("Server returned an error")
	} else {
		log.Info("Discovered LUNs:")
		for _, lun := range resp.Luns {
			log.WithFields(logrus.Fields{
				"LUN ID":  lun.LunID,
				"Size":    lun.Size,
				"Vendor":  lun.Vendor,
				"Product": lun.Product,
				"Serial":  lun.Serial,
			}).Info("LUN details")
		}
	}
}
