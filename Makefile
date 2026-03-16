# Makefile for HashiCorp-IBMi-plugin

PLUGIN_NAME=vault-plugin-secrets-ibmi
PLUGIN_DIR=cmd/ibmiauth
GO=go

# Auto-detect platform
UNAME_S := $(shell uname -s)
UNAME_M := $(shell uname -m)

# Set defaults based on platform
ifeq ($(UNAME_S),Darwin)
    ifeq ($(UNAME_M),arm64)
        GOOS?=darwin
        GOARCH?=arm64
    else
        GOOS?=darwin
        GOARCH?=amd64
    endif
else ifeq ($(UNAME_S),Linux)
    GOOS?=linux
    GOARCH?=amd64
else
    GOOS?=linux
    GOARCH?=amd64
endif

.PHONY: all build clean test fmt vet

all: build

build:
	@echo "Building $(PLUGIN_NAME) for $(GOOS)/$(GOARCH)..."
	GOOS=$(GOOS) GOARCH=$(GOARCH) $(GO) build -o $(PLUGIN_NAME) $(PLUGIN_DIR)/main.go
	@echo "Build complete: $(PLUGIN_NAME)"

build-all:
	@echo "Building for multiple platforms..."
	GOOS=linux GOARCH=amd64 $(GO) build -o $(PLUGIN_NAME)-linux-amd64 $(PLUGIN_DIR)/main.go
	GOOS=darwin GOARCH=amd64 $(GO) build -o $(PLUGIN_NAME)-darwin-amd64 $(PLUGIN_DIR)/main.go
	GOOS=darwin GOARCH=arm64 $(GO) build -o $(PLUGIN_NAME)-darwin-arm64 $(PLUGIN_DIR)/main.go
	GOOS=windows GOARCH=amd64 $(GO) build -o $(PLUGIN_NAME)-windows-amd64.exe $(PLUGIN_DIR)/main.go
	@echo "Multi-platform build complete"

clean:
	@echo "Cleaning build artifacts..."
	rm -f $(PLUGIN_NAME)*
	@echo "Clean complete"

test:
	@echo "Running tests..."
	$(GO) test -v ./...

fmt:
	@echo "Formatting code..."
	$(GO) fmt ./...

vet:
	@echo "Running go vet..."
	$(GO) vet ./...

deps:
	@echo "Downloading dependencies..."
	$(GO) mod download
	$(GO) mod tidy

help:
	@echo "Available targets:"
	@echo "  build      - Build the plugin for current platform (default: linux/amd64)"
	@echo "  build-all  - Build for multiple platforms (linux, darwin, windows)"
	@echo "  clean      - Remove build artifacts"
	@echo "  test       - Run tests"
	@echo "  fmt        - Format code"
	@echo "  vet        - Run go vet"
	@echo "  deps       - Download and tidy dependencies"
	@echo ""
	@echo "Environment variables:"
	@echo "  GOOS       - Target operating system (default: linux)"
	@echo "  GOARCH     - Target architecture (default: amd64)"