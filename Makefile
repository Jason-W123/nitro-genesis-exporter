.PHONY: all clean build build-native build-wasm serve test help

# Default target
all: build

# Build all versions
build: build-native build-wasm

# Build native version
build-native:
	@echo "Building native version..."
	go build -o nitro-genesis-exporter ./src/

# Build WASM version
build-wasm:
	@echo "Building WASM version..."
	GOOS=js GOARCH=wasm go build -o nitro-genesis-exporter.wasm ./wasm_src/

# Clean generated files
clean:
	@echo "Cleaning files..."
	rm -f nitro-genesis-exporter nitro-genesis-exporter.wasm wasm_exec.js

# Check dependencies
deps:
	@echo "Checking Go dependencies..."
	go mod tidy
	go mod download

# Show help
help:
	@echo "Available commands:"
	@echo "  build         - Build all versions (native + WASM)"
	@echo "  build-native  - Build only native version"
	@echo "  build-wasm    - Build only WASM version"
	@echo "  clean         - Clean generated files"
	@echo "  deps          - Check and download dependencies"
	@echo "  help          - Show this help information"

# Example: Use native version to calculate state root
example-native: build-native
	@echo "Example: Using native version (requires genesis.json file)"
	@if [ -f "genesis.json" ]; then \
		./nitro-genesis-exporter -g genesis.json; \
	else \
		echo "Please provide genesis.json file"; \
	fi 