package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ataljanseva/whatsapp-bot/config"
	"github.com/ataljanseva/whatsapp-bot/internal/bot"
	"github.com/ataljanseva/whatsapp-bot/internal/db"
	"github.com/ataljanseva/whatsapp-bot/internal/inactivity"
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

	// ── Context for clean shutdown ────────────────────────────────────────────
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// ── Redis ─────────────────────────────────────────────────────────────────
	rdb := redis.NewClient(&redis.Options{
		Addr:         cfg.RedisAddr,
		Password:     cfg.RedisPassword,
		DB:           cfg.RedisDB,
		PoolSize:     cfg.WorkerCount + 10,
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

	// ── WhatsApp client ───────────────────────────────────────────────────────
	waClient := whatsapp.New(cfg.WAPhoneNumberID, cfg.WAAccessToken)

	// ── TASK 3 – Inactivity monitor ───────────────────────────────────────────
	// When a user is idle for 90 seconds, we:
	//   1. Send them a localised "session timed out" message.
	//   2. Delete their Redis session so the next "Hi" starts fresh.
	//      (Session deletion is handled inside the monitor itself before calling
	//       this callback, so the callback only needs to send the message.)
	inactiveMonitor := inactivity.New(rdb, func(cbCtx context.Context, phone, lang string) {
		// Session has already been cleared by the monitor before this fires.
		// lang is the user's last-known language — use it for a localised message.
		msg := inactivity.InactivityMessage(lang)
		if err := waClient.SendText(cbCtx, phone, msg); err != nil {
			slog.Warn("inactivity: failed to send timeout message", "phone", phone, "err", err)
		}
	})
	inactiveMonitor.Start(ctx)

	// ── Bot + Worker pool ─────────────────────────────────────────────────────
	botHandler := bot.New(waClient, sessionStore, repo, inactiveMonitor)

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
		"inactivity_timeout", inactivity.InactivityTimeout,
	)

	// Run server in a goroutine so we can listen for shutdown signals
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server", "err", err)
			os.Exit(1)
		}
	}()

	// Block until signal
	<-ctx.Done()
	slog.Info("shutdown signal received, draining…")

	shutCtx, shutCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutCancel()
	_ = srv.Shutdown(shutCtx)
	slog.Info("server stopped cleanly")
}