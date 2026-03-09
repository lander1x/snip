.PHONY: build run test lint clean docker-up docker-down migrate

# Build
build:
	go build -o bin/shortener ./cmd/shortener
	go build -o bin/collector ./cmd/collector

run-shortener:
	go run ./cmd/shortener

run-collector:
	go run ./cmd/collector

# Test
test:
	go test ./... -v -race -count=1

test-cover:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html

# Lint
lint:
	golangci-lint run ./...

# Docker
docker-up:
	docker-compose -f deployments/docker-compose.yml up -d

docker-down:
	docker-compose -f deployments/docker-compose.yml down

docker-build:
	docker build -t snip-shortener --target shortener .
	docker build -t snip-collector --target collector .

# Migrations
migrate-postgres:
	@for f in migrations/postgres/*.sql; do \
		echo "Applying $$f..."; \
		PGPASSWORD=snip psql -h localhost -U snip -d snip -f $$f; \
	done

migrate-clickhouse:
	@for f in migrations/clickhouse/*.sql; do \
		echo "Applying $$f..."; \
		clickhouse-client --host localhost --query "$$(cat $$f)"; \
	done

# Clean
clean:
	rm -rf bin/ coverage.out coverage.html
