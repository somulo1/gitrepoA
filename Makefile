# VaultKe Backend Makefile
# Provides convenient commands for testing, building, and running the application

.PHONY: help test test-verbose test-coverage test-specific test-auth test-meeting test-chat test-notification test-placeholder bench clean build run dev lint format deps install-deps

# Default target
help:
	@echo "VaultKe Backend - Available Commands:"
	@echo "=================================="
	@echo "Testing:"
	@echo "  test              - Run all tests"
	@echo "  test-verbose      - Run all tests with verbose output"
	@echo "  test-coverage     - Run tests and generate coverage report"
	@echo "  test-specific     - Run specific test (usage: make test-specific TEST=TestName)"
	@echo "  test-auth         - Run authentication tests only"
	@echo "  test-meeting      - Run meeting tests only"
	@echo "  test-chat         - Run chat tests only"
	@echo "  test-notification - Run notification tests only"
	@echo "  test-placeholder  - Run placeholder tests only"
	@echo "  bench             - Run benchmark tests"
	@echo ""
	@echo "Development:"
	@echo "  build             - Build the application"
	@echo "  run               - Run the application"
	@echo "  dev               - Run in development mode with auto-reload"
	@echo "  clean             - Clean build artifacts and test coverage"
	@echo ""
	@echo "Code Quality:"
	@echo "  lint              - Run linter"
	@echo "  format            - Format code"
	@echo "  deps              - Download dependencies"
	@echo "  install-deps      - Install required tools"

# Test commands
test:
	@echo "🧪 Running all API tests..."
	@go test ./internal/api/...

test-verbose:
	@echo "🧪 Running all API tests (verbose)..."
	@go test -v ./internal/api/...

test-coverage:
	@echo "📊 Running tests with coverage..."
	@mkdir -p coverage
	@go test -coverprofile=coverage/coverage.out ./internal/api/...
	@go tool cover -html=coverage/coverage.out -o coverage/coverage.html
	@echo "📄 Coverage report generated: coverage/coverage.html"

test-specific:
	@if [ -z "$(TEST)" ]; then \
		echo "❌ Error: TEST variable is required. Usage: make test-specific TEST=TestName"; \
		exit 1; \
	fi
	@echo "🎯 Running specific test: $(TEST)"
	@go test -v -run $(TEST) ./internal/api/...

test-auth:
	@echo "🔐 Running authentication tests..."
	@go test -v ./internal/api -run TestAuth

test-meeting:
	@echo "📅 Running meeting tests..."
	@go test -v ./internal/api -run TestMeeting

test-chat:
	@echo "💬 Running chat tests..."
	@go test -v ./internal/api -run TestChat

test-notification:
	@echo "🔔 Running notification tests..."
	@go test -v ./internal/api -run TestNotification

test-placeholder:
	@echo "📝 Running placeholder tests..."
	@go test -v ./internal/api -run TestPlaceholder

bench:
	@echo "🏃 Running benchmark tests..."
	@go test -bench=. -benchmem ./internal/api/...

# Development commands
build:
	@echo "🔨 Building application..."
	@go build -o bin/vaultke-backend main.go
	@echo "✅ Build complete: bin/vaultke-backend"

run:
	@echo "🚀 Starting VaultKe Backend..."
	@go run main.go

dev:
	@echo "🔄 Starting development server with auto-reload..."
	@if command -v air > /dev/null; then \
		air; \
	else \
		echo "📦 Installing air for auto-reload..."; \
		go install github.com/cosmtrek/air@latest; \
		air; \
	fi

# Code quality commands
lint:
	@echo "🔍 Running linter..."
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run; \
	else \
		echo "📦 Installing golangci-lint..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
		golangci-lint run; \
	fi

format:
	@echo "✨ Formatting code..."
	@go fmt ./...
	@if command -v goimports > /dev/null; then \
		goimports -w .; \
	else \
		echo "📦 Installing goimports..."; \
		go install golang.org/x/tools/cmd/goimports@latest; \
		goimports -w .; \
	fi

# Dependency management
deps:
	@echo "📦 Downloading dependencies..."
	@go mod download
	@go mod tidy

install-deps:
	@echo "🛠️  Installing development tools..."
	@go install github.com/cosmtrek/air@latest
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install golang.org/x/tools/cmd/goimports@latest
	@go install github.com/stretchr/testify@latest
	@echo "✅ Development tools installed"

# Cleanup
clean:
	@echo "🧹 Cleaning up..."
	@rm -rf bin/
	@rm -rf coverage/
	@rm -rf tmp/
	@go clean -cache
	@echo "✅ Cleanup complete"

# Database commands
db-migrate:
	@echo "🗄️  Running database migrations..."
	@go run main.go migrate

db-seed:
	@echo "🌱 Seeding database with test data..."
	@go run main.go seed

db-reset:
	@echo "🔄 Resetting database..."
	@rm -f vaultke.db
	@go run main.go migrate
	@go run main.go seed

# Docker commands
docker-build:
	@echo "🐳 Building Docker image..."
	@docker build -t vaultke-backend .

docker-run:
	@echo "🐳 Running Docker container..."
	@docker run -p 8080:8080 vaultke-backend

# CI/CD helpers
ci-test:
	@echo "🤖 Running CI tests..."
	@go test -race -coverprofile=coverage.out ./internal/api/...
	@go tool cover -func=coverage.out

ci-build:
	@echo "🤖 Running CI build..."
	@go build -race -o bin/vaultke-backend main.go

# Quick development setup
setup: install-deps deps db-reset
	@echo "🎉 Development environment setup complete!"
	@echo "Run 'make dev' to start the development server"

# Test everything (comprehensive)
test-all: test-coverage lint
	@echo "🎯 All tests and checks completed!"

# Show test coverage in browser
coverage-view: test-coverage
	@if command -v open > /dev/null; then \
		open coverage/coverage.html; \
	elif command -v xdg-open > /dev/null; then \
		xdg-open coverage/coverage.html; \
	else \
		echo "📄 Coverage report available at: coverage/coverage.html"; \
	fi
