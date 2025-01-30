.PHONY: all proto docker run run-arm64 run-amd64 clean build-client

# Directories
BUILD_DIR := build/bin
BIN_DIR := $(BUILD_DIR)/bin

# All steps (excluding Docker build)
all: proto docker

# Generate protobuf files
proto:
	protoc --proto_path=proto \
	       --go_out=pkg/iscsi-command --go_opt=paths=source_relative \
	       --go-grpc_out=pkg/iscsi-command --go-grpc_opt=paths=source_relative \
	       proto/command.proto

# Build Docker image with BuildKit (supports multi-arch)
docker:
	mkdir -p $(BUILD_DIR)
	if command -v docker buildx >/dev/null 2>&1; then \
		docker buildx build --platform linux/amd64,linux/arm64 --load -t iscsi-server -f build/Dockerfile .; \
	else \
		DOCKER_BUILDKIT=1 docker build -t iscsi-server -f build/Dockerfile .; \
	fi

# Build the client binary (supports multi-arch)
build-client:
	mkdir -p $(BIN_DIR)
	GOOS=linux GOARCH=amd64 go build -o $(BIN_DIR)/iscsi-client-linux-amd64 ./cmd/client/main.go
	GOOS=linux GOARCH=arm64 go build -o $(BIN_DIR)/iscsi-client-linux-arm64 ./cmd/client/main.go
	GOOS=darwin GOARCH=amd64 go build -o $(BIN_DIR)/iscsi-client-mac-amd64 ./cmd/client/main.go
	GOOS=darwin GOARCH=arm64 go build -o $(BIN_DIR)/iscsi-client-mac-arm64 ./cmd/client/main.go

# Detects OS and architecture to set correct CONFIG_PATH
CONFIG_PATH := $(shell pwd)/config.yaml
ifeq ($(shell uname),Darwin)
	CONFIG_PATH := $(shell greadlink -f config.yaml)
endif

# Run the Docker container (compatible with MacOS & Linux)
run:
	docker run --rm \
		-v $(CONFIG_PATH):/root/config.yaml \
		-v /tmp:/tmp \
		iscsi-server --config=/root/config.yaml

# Run the Docker container on Mac M1/M2 (arm64)
run-arm64:
	docker run --rm --platform linux/arm64 \
		-v $(CONFIG_PATH):/root/config.yaml \
		-v /tmp:/tmp \
		iscsi-server --config=/root/config.yaml

# Run the Docker container on Mac Intel (amd64)
run-amd64:
	docker run --rm --platform linux/amd64 \
		-v $(CONFIG_PATH):/root/config.yaml \
		-v /tmp:/tmp \
		iscsi-server --config=/root/config.yaml

# Clean build artifacts
clean:
	rm -rf $(BUILD_DIR)
