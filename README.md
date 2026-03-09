# Snip — Distributed URL Shortener

A distributed URL shortener built with Go, showcasing microservice architecture with gRPC, NATS, ClickHouse, and Grafana.

## Architecture

```
                  ┌──────────────┐
   HTTP ──────▶   │  shortener   │  (Go, Fiber)
   gRPC ──────▶   │  api         │──▶ PostgreSQL (links)
                  │              │──▶ Redis (cache hot links)
                  │              │──▶ NATS (publish click events)
                  └──────────────┘
                         │
                    NATS subject
                   "clicks.created"
                         │
                  ┌──────────────┐
                  │  analytics   │  (Go)
                  │  collector   │──▶ ClickHouse (click events)
                  └──────────────┘
                         │
                  ┌──────────────┐
                  │  Grafana     │──▶ ClickHouse datasource
                  │  dashboards  │
                  └──────────────┘
```

## Tech Stack

| Component   | Technology                |
|-------------|---------------------------|
| API         | Go, Fiber, gRPC           |
| Database    | PostgreSQL                |
| Cache       | Redis                     |
| Messaging   | NATS JetStream            |
| Analytics   | ClickHouse                |
| Dashboards  | Grafana                   |
| Deployment  | Docker Compose, Kubernetes|

## Quick Start

### Prerequisites

- Go 1.23+
- Docker & Docker Compose

### Run locally

```bash
# Start infrastructure
make docker-up

# Run shortener API
make run-shortener

# Run analytics collector (in another terminal)
make run-collector
```

### Access services

| Service    | URL                    |
|------------|------------------------|
| HTTP API   | http://localhost:8080   |
| gRPC       | localhost:50051         |
| Grafana    | http://localhost:3000   |
| NATS       | http://localhost:8222   |

## API

### HTTP Endpoints

```bash
# Create short link
curl -X POST http://localhost:8080/api/v1/links \
  -H "Content-Type: application/json" \
  -d '{"url": "https://example.com", "custom_alias": "ex"}'

# Redirect (follow short link)
curl -L http://localhost:8080/ex

# Get link info
curl http://localhost:8080/api/v1/links/ex

# Delete link
curl -X DELETE http://localhost:8080/api/v1/links/ex

# Health check
curl http://localhost:8080/api/v1/health
```

### gRPC

```protobuf
service Shortener {
  rpc CreateLink(CreateLinkRequest) returns (Link);
  rpc ResolveLink(ResolveLinkRequest) returns (Link);
  rpc DeleteLink(DeleteLinkRequest) returns (google.protobuf.Empty);
}
```

## Validation Rules

- **URL**: must be absolute `http://` or `https://` with a host
- **custom_alias**: optional, 1-12 alphanumeric characters (`[a-zA-Z0-9]`)
- Auto-generated codes: 6-char base62, retry on collision (up to 5 attempts)

## Data Flow

1. Client creates a short link via HTTP/gRPC → validated, stored in PostgreSQL
2. Client visits short link → URL resolved from Redis cache (or PostgreSQL fallback)
3. On redirect (302), a click event is published to NATS JetStream subject `clicks.created`
4. Collector service subscribes with a durable consumer, batches events (1000 or 5s), inserts into ClickHouse with retry
5. Grafana dashboards query ClickHouse for real-time analytics

## Development

```bash
make build          # Build both binaries
make test           # Run tests
make lint           # Run linter
make proto          # Regenerate protobuf code
make docker-up      # Start infrastructure
make docker-down    # Stop infrastructure
```

## Project Structure

```
snip/
├── cmd/
│   ├── shortener/     # HTTP/gRPC server entrypoint
│   └── collector/     # NATS consumer entrypoint
├── internal/
│   ├── shortener/     # Shortener service (handlers, service, repo, cache)
│   ├── collector/     # Collector service (consumer, writer)
│   └── common/        # Shared config and logging
├── proto/             # Protobuf definitions and generated code
├── migrations/        # Database migrations (PostgreSQL, ClickHouse)
├── grafana/           # Grafana dashboard provisioning
└── deployments/       # Docker Compose and Kubernetes manifests
```
