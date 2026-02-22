.PHONY: all build test cover lint clean dev fmt vet setup

# --- Config ---
BINARY    := tenantiq
BUILD_DIR := ./bin
COVER_DIR := ./coverage

all: fmt vet lint test build

# --- Setup ---
setup:
	@echo "==> Setting up development environment..."
	@echo "  Installing lefthook..."
	@which lefthook > /dev/null 2>&1 || go install github.com/evilmartians/lefthook@latest
	lefthook install
	@echo "  Downloading tool dependencies..."
	go mod download
	@echo "==> Done! Pre-commit hooks are active."

# --- Build ---
build:
	@echo "==> Building $(BINARY)..."
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY) ./cmd/tenantiq

# --- Quality ---
fmt:
	@echo "==> Formatting..."
	gofmt -w .

vet:
	@echo "==> Vetting..."
	go vet ./...

lint:
	@echo "==> Linting..."
	go tool golangci-lint run ./...

# --- Testing ---
test:
	@echo "==> Running tests with coverage..."
	@mkdir -p $(COVER_DIR)
	go test ./... -coverprofile=$(COVER_DIR)/coverage.out -covermode=atomic -v
	@echo ""
	@echo "==> Coverage summary:"
	@go tool cover -func=$(COVER_DIR)/coverage.out | tail -1

cover: test
	@echo "==> Generating HTML coverage report..."
	go tool cover -html=$(COVER_DIR)/coverage.out -o $(COVER_DIR)/coverage.html
	@echo "  Report: $(COVER_DIR)/coverage.html"
	@open $(COVER_DIR)/coverage.html 2>/dev/null || xdg-open $(COVER_DIR)/coverage.html 2>/dev/null || echo "  Open $(COVER_DIR)/coverage.html in your browser"

# --- Development ---
dev:
	@echo "==> Running in development mode..."
	go run ./cmd/tenantiq

# --- Frontend (stubs for future React integration) ---
# frontend-install:
# 	@echo "==> Installing frontend dependencies..."
# 	cd web && npm install
#
# frontend-build:
# 	@echo "==> Building frontend..."
# 	cd web && npm run build
#
# frontend-dev:
# 	@echo "==> Starting frontend dev server..."
# 	cd web && npm run dev

# --- Cleanup ---
clean:
	@echo "==> Cleaning..."
	rm -rf $(BUILD_DIR) $(COVER_DIR)
	go clean

# --- Help ---
help:
	@echo "Available targets:"
	@echo "  make setup     Install dev tools and git hooks (run once after clone)"
	@echo "  make all       Run fmt, vet, lint, test, and build"
	@echo "  make build     Build the binary"
	@echo "  make test      Run tests with coverage"
	@echo "  make cover     Run tests and open HTML coverage report"
	@echo "  make fmt       Format Go code"
	@echo "  make vet       Run go vet"
	@echo "  make lint      Run golangci-lint"
	@echo "  make dev       Run in development mode"
	@echo "  make clean     Remove build artifacts"
	@echo "  make help      Show this help"
