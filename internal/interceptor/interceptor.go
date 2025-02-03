package interceptor

import (
	"context"

	"github.com/deckhouse/iscsi-command/internal/logger"
	"google.golang.org/grpc"
)

// LoggingInterceptor logs incoming requests
func LoggingInterceptor(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {
	log := logger.Log.WithField("method", info.FullMethod)

	log.Info("Received gRPC request")

	// Call the handler to process the request
	resp, err := handler(ctx, req)

	if err != nil {
		log.WithError(err).Error("Request processing failed")
	} else {
		log.Info("Request processed successfully")
	}

	return resp, err
}
