package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/nats-io/nats.go"

	"github.com/landerix/snip/internal/collector"
	"github.com/landerix/snip/internal/common"
)

func main() {
	cfg := common.LoadConfig()
	log := common.NewLogger()

	// ClickHouse
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{cfg.ClickHouse.Addr},
		Auth: clickhouse.Auth{
			Database: cfg.ClickHouse.Database,
			Username: cfg.ClickHouse.Username,
			Password: cfg.ClickHouse.Password,
		},
	})
	if err != nil {
		log.Error("failed to connect to clickhouse", "error", err)
		os.Exit(1)
	}
	defer conn.Close()

	if err := conn.Ping(context.Background()); err != nil {
		log.Error("failed to ping clickhouse", "error", err)
		os.Exit(1)
	}

	// NATS JetStream
	nc, err := nats.Connect(cfg.NATS.URL)
	if err != nil {
		log.Error("failed to connect to nats", "error", err)
		os.Exit(1)
	}
	defer nc.Close()

	js, err := nc.JetStream()
	if err != nil {
		log.Error("failed to create jetstream context", "error", err)
		os.Exit(1)
	}

	// Writer + Consumer
	writer := collector.NewWriter(conn, log)
	consumer := collector.NewConsumer(js, writer, log)

	if err := consumer.Start(); err != nil {
		log.Error("failed to start consumer", "error", err)
		os.Exit(1)
	}

	log.Info("collector started")

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	log.Info("shutting down", slog.String("signal", sig.String()))

	_ = consumer.Stop()
	writer.Stop()
}
