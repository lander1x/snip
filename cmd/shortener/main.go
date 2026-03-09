package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"github.com/redis/go-redis/v9"

	"github.com/landerix/snip/internal/common"
	"github.com/landerix/snip/internal/shortener"
)

func main() {
	cfg := common.LoadConfig()
	log := common.NewLogger()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// PostgreSQL
	pool, err := pgxpool.New(ctx, cfg.Postgres.DSN)
	if err != nil {
		log.Error("failed to connect to postgres", "error", err)
		os.Exit(1)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		log.Error("failed to ping postgres", "error", err)
		os.Exit(1)
	}

	// Redis
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	defer rdb.Close()

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

	// Ensure stream exists (idempotent — update if already exists)
	streamCfg := &nats.StreamConfig{
		Name:     "CLICKS",
		Subjects: []string{"clicks.>"},
		Storage:  nats.FileStorage,
	}
	if _, err = js.AddStream(streamCfg); err != nil {
		if _, err = js.UpdateStream(streamCfg); err != nil {
			log.Error("failed to ensure jetstream stream", "error", err)
			os.Exit(1)
		}
	}

	// Services
	repo := shortener.NewRepository(pool)
	cache := shortener.NewCache(rdb, cfg.Redis.TTL)
	svc := shortener.NewService(repo, cache, js, cfg.BaseURL, log)
	handler := shortener.NewHandler(svc)

	// HTTP server (Fiber)
	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
	})
	app.Use(recover.New())
	app.Use(logger.New())
	handler.RegisterRoutes(app)

	go func() {
		log.Info("starting HTTP server", "addr", cfg.HTTPAddr)
		if err := app.Listen(cfg.HTTPAddr); err != nil {
			log.Error("HTTP server error", "error", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	log.Info("shutting down", slog.String("signal", sig.String()))

	_ = app.Shutdown()
}
