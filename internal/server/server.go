package server

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/deckhouse/iscsi-command/internal/entity"
	"os/exec"

	"github.com/deckhouse/iscsi-command/internal/logger"
	pb "github.com/deckhouse/iscsi-command/pkg/iscsi-command"
)

// Server implements the CommandExecutorServer interface
type Server struct {
	pb.UnimplementedCommandExecutorServer
}

// Execute processes gRPC requests and executes iscsi-ls
func (s *Server) Execute(ctx context.Context, req *pb.CommandRequest) (*pb.CommandResponse, error) {
	log := logger.Log.WithFields(map[string]interface{}{
		"command":       req.Command,
		"portal":        req.Portal,
		"initiatorName": req.InitiatorName,
		"targetIQN":     req.TargetIQN,
	})

	log.Info("Received Execute request")

	if req.Command != "iscsi-ls" {
		log.Warn("Unsupported command received")
		return &pb.CommandResponse{Error: "Unsupported command"}, nil
	}

	// Prepare the target URL with authentication if provided
	var targetURL string
	if req.AuthLogin != "" && req.AuthPassword != "" {
		targetURL = fmt.Sprintf("iscsi://%s%%%s@%s", req.AuthLogin, req.AuthPassword, req.Portal)
	} else {
		targetURL = fmt.Sprintf("iscsi://%s", req.Portal)
	}

	// Execute iscsi-ls with parameters
	cmdStr := fmt.Sprintf("iscsi-ls %s -i %s -s -T %s", targetURL, req.InitiatorName, req.TargetIQN)
	log = log.WithField("cmd", cmdStr)
	log.Info("Executing command")

	cmd := exec.Command("iscsi-ls", targetURL, "-i", req.InitiatorName, "-s", "-T", req.TargetIQN)
	output, err := cmd.CombinedOutput()

	// Handle context cancellation
	if ctx.Err() == context.Canceled {
		log.Warn("Request canceled by client")
		return nil, fmt.Errorf("command canceled: %w", ctx.Err())
	}
	if ctx.Err() == context.DeadlineExceeded {
		log.Warn("Request timed out")
		return nil, fmt.Errorf("command timed out: %w", ctx.Err())
	}
	if err != nil {
		log.WithError(err).Error("Command execution failed")
		return &pb.CommandResponse{Error: fmt.Sprintf("failed to run iscsi-ls: %s: %v", cmdStr, err)}, nil
	}

	log.Info("Command executed successfully, parsing output")

	// Parse the JSON output
	var targetsRaw []struct {
		Target  string           `json:"Target"`
		Portals []string         `json:"Portals"`
		LUNs    []entity.LUNInfo `json:"LUNs"`
	}
	if err := json.Unmarshal(output, &targetsRaw); err != nil {
		log.WithError(err).WithField("output", string(output)).Error("Failed to parse iscsi-ls output")
		return nil, fmt.Errorf("failed to parse iscsi-ls output: %w", err)
	}

	// Return LUNs for the specified target IQN
	for _, targetRaw := range targetsRaw {
		if targetRaw.Target == req.TargetIQN {
			log.Info("Found matching target")

			// Convert targetRaw.LUNs to []*pb.LUNInfo
			var luns []*pb.LUNInfo
			for _, lun := range targetRaw.LUNs {
				luns = append(luns, &pb.LUNInfo{
					Lun:    lun.LUN,
					Wwid:   lun.WWID,
					Size:   lun.Size,
					Errors: lun.Errors,
				})
			}

			log.WithField("lun_count", len(luns)).Info("Returning LUNs")
			return &pb.CommandResponse{
				Output: string(output),
				Luns:   luns,
			}, nil
		}
	}

	log.Warn("No matching LUNs found for target")
	return nil, fmt.Errorf("no LUNs found for target %s", req.TargetIQN)
}

// Ping responds with the status of the service
func (s *Server) Ping(ctx context.Context, req *pb.PingRequest) (*pb.PingResponse, error) {
	logger.Log.Info("Received Ping request")
	return &pb.PingResponse{
		Status: "Service is running",
	}, nil
}
