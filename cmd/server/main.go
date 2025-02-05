package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/deckhouse/iscsi-command/config"
	"github.com/deckhouse/iscsi-command/internal/interceptor"
	"github.com/deckhouse/iscsi-command/internal/logger"
	"github.com/deckhouse/iscsi-command/internal/server"
	pb "github.com/deckhouse/iscsi-command/pkg/iscsi-command"
	"google.golang.org/grpc"
	"net"
)

const defaultSocketPath = "/csi/iscsi.sock"

func main() {
	logger.Init()
	log := logger.Log

	log.Info("Starting gRPC server...")

	// Command-line parameter for config file (default: config.yaml)
	configFile := flag.String("config", "config.yaml", "Path to configuration file")
	flag.Parse()

	// Load configuration from file
	cfg, err := config.LoadConfig(*configFile)
	if err != nil {
		log.WithError(err).Warn("Failed to load configuration, using defaults")
		cfg = &config.Config{}
	}

	// Validate or set default socket path
	if cfg.SocketPath == "" {
		cfg.SocketPath = defaultSocketPath
		log.WithField("socketPath", cfg.SocketPath).Warn("Using default Unix socket path")
	}

	// Remove existing socket (if any)
	if err := os.RemoveAll(cfg.SocketPath); err != nil {
		log.WithError(err).Fatal("Failed to remove existing socket")
	}

	// Create Unix socket listener
	lis, err := net.Listen("unix", cfg.SocketPath)
	if err != nil {
		log.WithError(err).Fatalf("Failed to listen on Unix socket %s", cfg.SocketPath)
	}

	if err := os.Chmod(cfg.SocketPath, 0777); err != nil {
		log.WithError(err).Fatalf("Failed to set permissions on Unix socket %s", cfg.SocketPath)
	}

	log.WithField("socketPath", cfg.SocketPath).Info("Server is running on Unix socket")

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
		log.Info("gRPC server started on Unix socket")
		if err := grpcServer.Serve(lis); err != nil {
			log.WithError(err).Fatal("Failed to serve gRPC server")
		}
	}()

	// Wait for termination signal
	<-stopChan
	log.Info("Shutting down gracefully...")
	grpcServer.GracefulStop()
	log.Info("Server stopped")
}
