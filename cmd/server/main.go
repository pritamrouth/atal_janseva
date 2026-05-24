package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/ataljanseva/whatsapp-bot/config"
	"github.com/ataljanseva/whatsapp-bot/internal/bot"
	"github.com/ataljanseva/whatsapp-bot/internal/db"
	"github.com/ataljanseva/whatsapp-bot/internal/store"
	"github.com/ataljanseva/whatsapp-bot/internal/whatsapp"
	"github.com/ataljanseva/whatsapp-bot/internal/worker"

	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
)

func main() {
	// ── Structured logging (JSON in prod, text in dev) ────────────────────────
	if os.Getenv("ENV") == "production" {
		slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})))
	} else {
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})))
	}

	// ── Config ────────────────────────────────────────────────────────────────
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		slog.Warn("godotenv", "err", err)
	}
	cfg, err := config.Load()
	if err != nil {
		slog.Error("config", "err", err)
		os.Exit(1)
	}

	// ── Redis (local) ─────────────────────────────────────────────────────────
	rdb := redis.NewClient(&redis.Options{
		Addr:         cfg.RedisAddr,
		Password:     cfg.RedisPassword,
		DB:           cfg.RedisDB,
		PoolSize:     cfg.WorkerCount + 10, // one conn per worker + headroom
		MinIdleConns: 5,
		DialTimeout:  3 * time.Second,
		ReadTimeout:  2 * time.Second,
		WriteTimeout: 2 * time.Second,
	})

	sessionStore, err := store.New(rdb)
	if err != nil {
		slog.Error("redis", "err", err)
		os.Exit(1)
	}
	slog.Info("redis connected", "addr", cfg.RedisAddr)

	// ── PostgreSQL ────────────────────────────────────────────────────────────
	repo, err := db.New(cfg.DatabaseURL, cfg.DBMaxOpenConns, cfg.DBMaxIdleConns)
	if err != nil {
		slog.Error("database", "err", err)
		os.Exit(1)
	}
	defer repo.Close()
	slog.Info("database connected")

	// ── Bot + Worker pool ─────────────────────────────────────────────────────
	waClient   := whatsapp.New(cfg.WAPhoneNumberID, cfg.WAAccessToken)
	botHandler := bot.New(waClient, sessionStore, repo)

	pool := worker.New(cfg.WorkerCount, cfg.QueueDepth, botHandler.HandleRaw)
	pool.Start()

	// ── HTTP server ───────────────────────────────────────────────────────────
	webhook := &webhookHandler{
		verifyToken: cfg.WAVerifyToken,
		pool:        pool,
	}

	mux := http.NewServeMux()
	mux.Handle("/webhook", webhook)
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintf(w, `{"status":"ok","queue_len":%d}`, pool.QueueLen())
	})

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	slog.Info("Ataljanseva WhatsApp Bot started",
		"port", cfg.Port,
		"workers", cfg.WorkerCount,
		"queue_depth", cfg.QueueDepth,
	)
	if err := srv.ListenAndServe(); err != nil {
		slog.Error("server", "err", err)
		os.Exit(1)
	}
}
