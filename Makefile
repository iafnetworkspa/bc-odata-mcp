.PHONY: build build-windows build-linux build-darwin clean test lint

# Default target
build: build-windows

# Build for Windows
build-windows:
	@echo "Building for Windows..."
	go build -v -o bc-odata-mcp.exe ./cmd/server

# Build for Linux
build-linux:
	@echo "Building for Linux..."
	GOOS=linux GOARCH=amd64 go build -v -o bc-odata-mcp-linux-amd64 ./cmd/server

# Build for Darwin (macOS)
build-darwin:
	@echo "Building for Darwin..."
	GOOS=darwin GOARCH=amd64 go build -v -o bc-odata-mcp-darwin-amd64 ./cmd/server

# Build for all platforms
build-all: build-windows build-linux build-darwin

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -f bc-odata-mcp.exe
	rm -f bc-odata-mcp-linux-amd64
	rm -f bc-odata-mcp-darwin-amd64
	rm -f server.exe
	rm -f server
	rm -rf dist/

# Run tests
test:
	@echo "Running tests..."
	go test -v -race -coverprofile=coverage.out ./...

# Run linter
lint:
	@echo "Running linter..."
	golangci-lint run

