# ---- Stage 1: Build frontend ----
FROM node:24-slim AS frontend

WORKDIR /src/web

# Enable Corepack so pnpm is available at the pinned version.
RUN corepack enable

# Install dependencies first (layer cache).
COPY web/package.json web/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile

# Build production bundle → dist/
COPY web/ ./
RUN pnpm build

# ---- Stage 2: Build Go binary ----
FROM golang:1.25 AS backend

WORKDIR /src

# Download Go modules first (layer cache).
COPY go.mod go.sum ./
RUN go mod download

# Copy source and the frontend bundle from stage 1.
COPY . .
COPY --from=frontend /src/web/dist ./web/dist

# Build a fully static binary (pure Go, no CGO needed — uses modernc.org/sqlite).
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /tenantiq ./cmd/tenantiq

# ---- Stage 3: Minimal runtime ----
FROM gcr.io/distroless/static-debian12

COPY --from=backend /tenantiq /tenantiq

EXPOSE 8019

ENTRYPOINT ["/tenantiq"]
