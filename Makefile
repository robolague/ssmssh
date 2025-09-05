# Makefile for ssmssh project

.PHONY: test build clean install help

# Default target
all: test build

# Run tests
test:
	go test -v

# Run tests with coverage
test-coverage:
	go test -cover -v

# Build the application
build:
	go build -o ssmssh .

# Build for multiple platforms
build-all:
	GOOS=linux GOARCH=amd64 go build -o ssmssh-linux-amd64 .
	GOOS=darwin GOARCH=amd64 go build -o ssmssh-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 go build -o ssmssh-darwin-arm64 .
	GOOS=windows GOARCH=amd64 go build -o ssmssh-windows-amd64.exe .

# Install the application
install:
	go install .

# Clean build artifacts
clean:
	go clean
	rm -f ssmssh ssmssh-*

# Format code
fmt:
	go fmt ./...

# Run go vet
vet:
	go vet ./...

# Run all quality checks
quality: fmt vet test

# Help target
help:
	@echo "Available targets:"
	@echo "  test         - Run tests with verbose output"
	@echo "  test-coverage- Run tests with coverage"
	@echo "  build        - Build the application"
	@echo "  build-all    - Build for multiple platforms"
	@echo "  install      - Install the application"
	@echo "  clean        - Clean build artifacts"
	@echo "  fmt          - Format code"
	@echo "  vet          - Run go vet"
	@echo "  quality      - Run fmt, vet, and test"
	@echo "  help         - Show this help message"