.PHONY: build run clean install-deps test help

# Binary name
BINARY_NAME=tdclient
CONFIG_FILE=config.yaml

# Build variables
BUILD_DIR=build
GO=go
GOFLAGS=-v

# Detect platform
UNAME_S := $(shell uname -s)
UNAME_M := $(shell uname -m)

# Set library paths based on platform
ifeq ($(UNAME_S),Darwin)
    ifeq ($(UNAME_M),arm64)
        # Apple Silicon
        BREW_PREFIX=/opt/homebrew
    else
        # Intel Mac
        BREW_PREFIX=/usr/local
    endif

    # Find OpenSSL
    OPENSSL_PATH=$(shell if [ -d "$(BREW_PREFIX)/opt/openssl@3" ]; then echo "$(BREW_PREFIX)/opt/openssl@3"; elif [ -d "$(BREW_PREFIX)/opt/openssl@1.1" ]; then echo "$(BREW_PREFIX)/opt/openssl@1.1"; else echo "$(BREW_PREFIX)/opt/openssl"; fi)

    export CGO_CFLAGS=-I$(BREW_PREFIX)/include -I$(OPENSSL_PATH)/include
    export CGO_LDFLAGS=-L$(BREW_PREFIX)/lib -L$(OPENSSL_PATH)/lib
    export PKG_CONFIG_PATH=$(OPENSSL_PATH)/lib/pkgconfig
else
    # Linux
    export CGO_CFLAGS=-I/usr/local/include
    export CGO_LDFLAGS=-L/usr/local/lib
endif

help: ## Show this help message
	@echo "Telegram Channel Monitor - Makefile"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-15s %s\n", $$1, $$2}'

build: ## Build the application
	@echo "Building $(BINARY_NAME)..."
	CGO_ENABLED=1 $(GO) build $(GOFLAGS) -tags libtdjson -o $(BINARY_NAME) ./cmd/tdclient

build-all: ## Build for multiple platforms
	@echo "Building for multiple platforms..."
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=1 $(GO) build -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 ./cmd/tdclient
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=1 $(GO) build -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/tdclient
	GOOS=linux GOARCH=amd64 CGO_ENABLED=1 $(GO) build -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/tdclient
	@echo "Binaries built in $(BUILD_DIR)/"

run: build ## Build and run the application
	@echo "Running $(BINARY_NAME)..."
	./$(BINARY_NAME) -config $(CONFIG_FILE)

dev: ## Run in development mode with debug logging
	@echo "Running in development mode..."
	CGO_ENABLED=1 $(GO) run -tags libtdjson ./cmd/tdclient -config $(CONFIG_FILE) -log-level debug

install-deps: ## Install Go dependencies
	@echo "Installing dependencies..."
	$(GO) mod download
	$(GO) mod tidy

test: ## Run tests
	@echo "Running tests..."
	$(GO) test -v ./...

clean: ## Clean build artifacts
	@echo "Cleaning..."
	rm -f $(BINARY_NAME)
	rm -rf $(BUILD_DIR)
	rm -rf data/
	@echo "Clean complete"

config: ## Create config.yaml from example
	@if [ ! -f $(CONFIG_FILE) ]; then \
		cp config.yaml.example $(CONFIG_FILE); \
		echo "Created $(CONFIG_FILE) from example"; \
		echo "Please edit $(CONFIG_FILE) with your API credentials"; \
	else \
		echo "$(CONFIG_FILE) already exists"; \
	fi

setup: install-deps config ## Initial setup (install deps + create config)
	@echo ""
	@echo "Setup complete! Next steps:"
	@echo "1. Edit config.yaml with your Telegram API credentials"
	@echo "2. Ensure TDLib is installed on your system"
	@echo "3. Run 'make build' to build the application"
	@echo "4. Run 'make run' to start the application"

fmt: ## Format Go code
	@echo "Formatting code..."
	$(GO) fmt ./...

vet: ## Run go vet
	@echo "Running go vet..."
	$(GO) vet ./...

check: fmt vet ## Run code quality checks
	@echo "Code quality checks complete"

.DEFAULT_GOAL := help
