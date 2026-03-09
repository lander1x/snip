FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /bin/shortener ./cmd/shortener
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /bin/collector ./cmd/collector

# Shortener
FROM alpine:3.20 AS shortener
RUN apk --no-cache add ca-certificates
COPY --from=builder /bin/shortener /usr/local/bin/shortener
EXPOSE 8080
CMD ["shortener"]

# Collector
FROM alpine:3.20 AS collector
RUN apk --no-cache add ca-certificates
COPY --from=builder /bin/collector /usr/local/bin/collector
CMD ["collector"]
