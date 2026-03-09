.PHONY: build release test test-linux clean install

# Build targets
BINARY_NAME=tmux-yankee
BIN_DIR=bin
CMD_DIR=cmd/tmux-yankee

# Go build flags
GOFLAGS=-trimpath
LDFLAGS=-s -w

# Build local development binary
build:
	@echo "Building $(BINARY_NAME) for local development..."
	@mkdir -p $(BIN_DIR)
	go build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY_NAME) ./$(CMD_DIR)
	@echo "Build complete: $(BIN_DIR)/$(BINARY_NAME)"

# Build release binaries for multiple platforms
release:
	@echo "Building release binaries..."
	@mkdir -p $(BIN_DIR)

	# macOS arm64 (Apple Silicon)
	GOOS=darwin GOARCH=arm64 go build $(GOFLAGS) -ldflags "$(LDFLAGS)" \
		-o $(BIN_DIR)/$(BINARY_NAME)-darwin-arm64 ./$(CMD_DIR)

	# macOS amd64 (Intel)
	GOOS=darwin GOARCH=amd64 go build $(GOFLAGS) -ldflags "$(LDFLAGS)" \
		-o $(BIN_DIR)/$(BINARY_NAME)-darwin-amd64 ./$(CMD_DIR)

	# Linux amd64
	GOOS=linux GOARCH=amd64 go build $(GOFLAGS) -ldflags "$(LDFLAGS)" \
		-o $(BIN_DIR)/$(BINARY_NAME)-linux-amd64 ./$(CMD_DIR)

	# Linux arm64
	GOOS=linux GOARCH=arm64 go build $(GOFLAGS) -ldflags "$(LDFLAGS)" \
		-o $(BIN_DIR)/$(BINARY_NAME)-linux-arm64 ./$(CMD_DIR)

	@echo "Release builds complete in $(BIN_DIR)/"
	@ls -lh $(BIN_DIR)/

# Run Go tests
test:
	@echo "Running Go tests..."
	go test ./...
	@echo "Running race detector tests..."
	go test -race ./...

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf $(BIN_DIR)
	go clean

# Install to system (copies binary to bin/tmux-yankee)
install: build
	@echo "Binary installed at $(BIN_DIR)/$(BINARY_NAME)"
	@echo "Ensure your plugin path points to this directory"

# Run tests inside Docker (Linux/amd64)
test-linux:
	@echo "Building and testing on Linux (Docker)..."
	docker build --platform linux/amd64 -f Dockerfile.test-linux -t tmux-yankee-test-linux .
	docker run --rm --platform linux/amd64 tmux-yankee-test-linux
