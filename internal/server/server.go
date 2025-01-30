package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"

	pb "github.com/deckhouse/iscsi-command/pkg/iscsi-command"
)

// Server implements the CommandExecutorServer interface
type Server struct {
	pb.UnimplementedCommandExecutorServer
}

// Execute processes gRPC requests and executes iscsi-ls
func (s *Server) Execute(ctx context.Context, req *pb.CommandRequest) (*pb.CommandResponse, error) {
	log.Printf("Received Execute request: Command=%s, Portal=%s, Initiator=%s, TargetIQN=%s",
		req.Command, req.Portal, req.InitiatorName, req.TargetIQN)

	if req.Command != "iscsi-ls" {
		log.Printf("Unsupported command: %s", req.Command)
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
	log.Printf("Executing command: %s", cmdStr)

	cmd := exec.Command("iscsi-ls", targetURL, "-i", req.InitiatorName, "-s", "-T", req.TargetIQN)
	output, err := cmd.CombinedOutput()

	// Handle context cancellation
	if ctx.Err() == context.Canceled {
		log.Println("Request canceled by client")
		return nil, fmt.Errorf("command canceled: %w", ctx.Err())
	}
	if ctx.Err() == context.DeadlineExceeded {
		log.Println("Request timed out")
		return nil, fmt.Errorf("command timed out: %w", ctx.Err())
	}
	if err != nil {
		log.Printf("Command failed: %s: %v", cmdStr, err)
		return &pb.CommandResponse{Error: fmt.Sprintf("failed to run iscsi-ls: %s: %v", cmdStr, err)}, nil
	}

	log.Println("Command executed successfully, parsing output")

	// Parse the JSON output
	var targetsRaw []struct {
		Target  string     `json:"Target"`
		Portals []string   `json:"Portals"`
		LUNs    []struct { // Temporary struct for parsing
			LunID   int32  `json:"LunID"`
			Size    string `json:"Size"`
			Vendor  string `json:"Vendor"`
			Product string `json:"Product"`
			Serial  string `json:"Serial"`
		} `json:"LUNs"`
	}
	if err := json.Unmarshal(output, &targetsRaw); err != nil {
		log.Printf("Failed to parse iscsi-ls output: %v", err)
		return nil, fmt.Errorf("failed to parse iscsi-ls output: %w", err)
	}

	// Return LUNs for the specified target IQN
	for _, targetRaw := range targetsRaw {
		if targetRaw.Target == req.TargetIQN {
			log.Printf("Found matching target: %s", targetRaw.Target)

			// Convert targetRaw.LUNs to []*pb.LUNInfo
			var luns []*pb.LUNInfo
			for _, lun := range targetRaw.LUNs {
				luns = append(luns, &pb.LUNInfo{
					LunID:   lun.LunID,
					Size:    lun.Size,
					Vendor:  lun.Vendor,
					Product: lun.Product,
					Serial:  lun.Serial,
				})
			}

			log.Printf("Returning %d LUNs for target %s", len(luns), req.TargetIQN)
			return &pb.CommandResponse{
				Output: string(output),
				Luns:   luns,
			}, nil
		}
	}

	log.Printf("No LUNs found for target %s", req.TargetIQN)
	return nil, fmt.Errorf("no LUNs found for target %s", req.TargetIQN)
}
