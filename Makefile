# Variables
BINARY_DIR=bin
SERVER_BINARY=server

# Go commands
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get

# Build flags
# -ldflags="-s -w" to reduce binary size
LDFLAGS=-ldflags="-s -w"
# -gcflags="all=-N -l" to disable optimizations and inlining for better debugging
DEBUG_GCFLAGS=-gcflags="all=-N -l"
# -race to enable race detector
RACE_FLAGS=-race

# Runtime GC and Debug settings
# GOGC: threshold of GC (default 100)
# GODEBUG: gctrace=1 enables GC logging to stderr
# GODEBUG: schedtrace=1000 enables scheduler tracing every 1000ms
# GODEBUG: gcpacertrace=1 enables GC pacer tracing
export GOGC=100
export GODEBUG=gctrace=1

.PHONY: all build clean server debug run-server help tidy fmt vet test race run-debug

all: build

build: server

server:
	@echo "Building server..."
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/$(SERVER_BINARY) cmd/server/main.go

# Build with debug info (no optimizations, no inlining)
debug:
	@echo "Building with debug flags..."
	$(GOBUILD) $(DEBUG_GCFLAGS) -o $(BINARY_DIR)/$(SERVER_BINARY)_debug cmd/server/main.go

# Build with race detector
race:
	@echo "Building with race detector..."
	$(GOBUILD) $(RACE_FLAGS) -o $(BINARY_DIR)/$(SERVER_BINARY)_race cmd/server/main.go

# Run server with GC trace enabled
run-server: server
	@echo "Running server with GOGC=$(GOGC) and GODEBUG=$(GODEBUG)..."
	./$(BINARY_DIR)/$(SERVER_BINARY)

# Run server with debug info and GC trace
run-debug: debug
	@echo "Running debug server with GOGC=$(GOGC) and GODEBUG=$(GODEBUG)..."
	./$(BINARY_DIR)/$(SERVER_BINARY)_debug

# Code quality and maintenance
tidy:
	@echo "Tidying up modules..."
	$(GOCMD) mod tidy

fmt:
	@echo "Formatting code..."
	$(GOCMD) fmt ./...

vet:
	@echo "Vetting code..."
	$(GOCMD) vet ./...

test:
	@echo "Running tests..."
	$(GOTEST) -v -race ./...

clean:
	@echo "Cleaning binaries..."
	$(GOCLEAN)
	rm -rf $(BINARY_DIR)

help:
	@echo "Available targets:"
	@echo "  make build       - Build server binary"
	@echo "  make server      - Build the server binary"
	@echo "  make debug       - Build server with debugging information"
	@echo "  make race        - Build server with race detector"
	@echo "  make run-server  - Run server with GC tracing"
	@echo "  make run-debug   - Run debug server with GC tracing"
	@echo "  make tidy        - Run go mod tidy"
	@echo "  make fmt         - Run go fmt"
	@echo "  make vet         - Run go vet"
	@echo "  make test        - Run tests with race detector"
	@echo "  make clean       - Remove built binaries"
	@echo "  make help        - Show this help message"

