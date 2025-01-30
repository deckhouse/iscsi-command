package main

import (
	"flag"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/deckhouse/iscsi-command/config"
	"github.com/deckhouse/iscsi-command/internal/interceptor"
	"github.com/deckhouse/iscsi-command/internal/server"
	pb "github.com/deckhouse/iscsi-command/pkg/iscsi-command"
	"google.golang.org/grpc"
)

func main() {
	// Command-line parameter for config file
	configFile := flag.String("config", "config.yaml", "Path to configuration file")
	flag.Parse()

	// Load configuration from file
	cfg, err := config.LoadConfig(*configFile)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Validate socket path
	if cfg.SocketPath == "" {
		log.Fatalf("Unix socket path is required in config.yaml")
	}

	// Remove existing socket (if any)
	if err := os.RemoveAll(cfg.SocketPath); err != nil {
		log.Fatalf("Failed to remove existing socket: %v", err)
	}

	// Create Unix socket listener
	lis, err := net.Listen("unix", cfg.SocketPath)
	if err != nil {
		log.Fatalf("Failed to listen on Unix socket %s: %v", cfg.SocketPath, err)
	}

	if err := os.Chmod(cfg.SocketPath, 0777); err != nil {
		log.Fatalf("Failed to set permissions on Unix socket: %v", err)
	}

	log.Printf("Server is running on Unix socket %s", cfg.SocketPath)

	// Create gRPC server with logging interceptor
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(interceptor.LoggingInterceptor),
	)

	srv := &server.Server{}
	pb.RegisterCommandExecutorServer(grpcServer, srv)

	// Handle signals for graceful shutdown
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Println("gRPC server started on Unix socket")
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to serve: %v", err)
		}
	}()

	// Wait for termination signal
	<-stopChan
	log.Println("Shutting down gracefully...")
	grpcServer.GracefulStop()
	log.Println("Server stopped")
}
