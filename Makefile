# Raptor Makefile
# Provides standardized build commands for development

.PHONY: all build test lint fmt coverage clean check vuln help

# Default target
all: check build

# Build the application
build:
	go build -v ./...

# Build binary to bin/ directory
build-bin:
	go build -v -o bin/raptor ./cmd/raptor

# Run all tests with race detector
test:
	go test -v -race ./...

# Run tests with coverage
coverage:
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

# Generate HTML coverage report
coverage-html: coverage
	go tool cover -html=coverage.out -o coverage.html

# Run linter
lint:
	golangci-lint run

# Fix linting issues automatically where possible
lint-fix:
	golangci-lint run --fix

# Format code
fmt:
	go fmt ./...
	goimports -w .

# Verify code is formatted
fmt-check:
	@test -z "$$(gofmt -l .)" || (echo "Code is not formatted. Run 'make fmt'" && exit 1)

# Run all checks (lint + test)
check: lint test

# Run vulnerability check
vuln:
	govulncheck ./...

# Clean build artifacts
clean:
	rm -rf bin/
	rm -f coverage.out coverage.html
	go clean

# Install development tools
tools:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install golang.org/x/tools/cmd/goimports@latest
	go install golang.org/x/vuln/cmd/govulncheck@latest

# Run the application
run:
	go run ./cmd/raptor $(ARGS)

# Show help
help:
	@echo "Raptor Makefile"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@echo "  all          Run check and build (default)"
	@echo "  build        Build the application"
	@echo "  build-bin    Build binary to bin/ directory"
	@echo "  test         Run tests with race detector"
	@echo "  coverage     Run tests with coverage report"
	@echo "  coverage-html Generate HTML coverage report"
	@echo "  lint         Run golangci-lint"
	@echo "  lint-fix     Run linter and fix issues"
	@echo "  fmt          Format code with gofmt and goimports"
	@echo "  fmt-check    Verify code is formatted"
	@echo "  check        Run lint and test"
	@echo "  vuln         Run govulncheck for vulnerabilities"
	@echo "  clean        Clean build artifacts"
	@echo "  tools        Install development tools"
	@echo "  run          Run the application (use ARGS= for arguments)"
	@echo "  help         Show this help"
