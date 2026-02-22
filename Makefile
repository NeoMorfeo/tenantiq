.PHONY: all build test cover lint clean dev fmt vet setup otel otel-stop help
.DEFAULT_GOAL := help

# --- Config ---
BINARY    := tenantiq
BUILD_DIR := ./bin
COVER_DIR := ./coverage
CONTAINER := $(shell command -v podman 2>/dev/null || echo docker)

all: fmt vet lint test build ## Run fmt, vet, lint, test, and build

# --- Setup ---
setup: ## Install dev tools and git hooks (run once after clone)
	@echo "==> Setting up development environment..."
	@echo "  Installing lefthook..."
	@which lefthook > /dev/null 2>&1 || go install github.com/evilmartians/lefthook@latest
	lefthook install
	@echo "  Downloading tool dependencies..."
	go mod download
	@echo "==> Done! Pre-commit hooks are active."

# --- Build ---
build: ## Build the binary
	@echo "==> Building $(BINARY)..."
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY) ./cmd/tenantiq

# --- Quality ---
fmt: ## Format Go code
	@echo "==> Formatting..."
ifeq ($(CI),true)
	@test -z "$$(gofmt -l .)" || (echo "Unformatted files:" && gofmt -l . && exit 1)
else
	gofmt -w .
endif

vet: ## Run go vet
	@echo "==> Vetting..."
	go vet ./...

lint: ## Run golangci-lint
	@echo "==> Linting..."
	go tool golangci-lint run ./...

# --- Testing ---
test: ## Run tests with coverage
	@echo "==> Running tests with coverage..."
	@mkdir -p $(COVER_DIR)
	go tool gotestsum --junitfile $(COVER_DIR)/junit.xml -- ./... -coverprofile=$(COVER_DIR)/coverage.out -covermode=atomic
	@echo ""
	@echo "==> Coverage summary:"
	@go tool cover -func=$(COVER_DIR)/coverage.out | tail -1

cover: test ## Run tests and open HTML coverage report
	@echo "==> Generating HTML coverage report..."
	go tool cover -html=$(COVER_DIR)/coverage.out -o $(COVER_DIR)/coverage.html
	@echo "  Report: $(COVER_DIR)/coverage.html"
	@open $(COVER_DIR)/coverage.html 2>/dev/null || xdg-open $(COVER_DIR)/coverage.html 2>/dev/null || echo "  Open $(COVER_DIR)/coverage.html in your browser"

# --- Development ---
dev: ## Run in development mode
	@echo "==> Running in development mode..."
	go run ./cmd/tenantiq

# --- Observability ---
otel: ## Start Grafana LGTM stack (Docker)
	@echo "==> Starting Grafana LGTM stack (Loki, Grafana, Tempo, Mimir)..."
	$(CONTAINER) run -d --name tenantiq-otel -p 3000:3000 -p 4317:4317 -p 4318:4318 grafana/otel-lgtm
	@echo "  Grafana:   http://localhost:3000"
	@echo "  OTLP gRPC: localhost:4317"
	@echo "  OTLP HTTP: localhost:4318"
	@echo ""
	@echo "  Run with: OTEL_EXPORTER=otlp make dev"

otel-stop: ## Stop Grafana LGTM stack
	@echo "==> Stopping Grafana LGTM stack..."
	$(CONTAINER) stop tenantiq-otel && $(CONTAINER) rm tenantiq-otel

# --- Cleanup ---
clean: ## Remove build artifacts
	@echo "==> Cleaning..."
	rm -rf $(BUILD_DIR) $(COVER_DIR)
	go clean

# --- Help ---
help: ## Show this help
	@echo "Usage: make [target]"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*##' $(MAKEFILE_LIST) | awk -F ':.*## ' '{printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}'
