package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/ataljanseva/whatsapp-bot/config"
	"github.com/ataljanseva/whatsapp-bot/internal/bot"
	"github.com/ataljanseva/whatsapp-bot/internal/store"
	"github.com/ataljanseva/whatsapp-bot/internal/whatsapp"

	// Optional: loads .env automatically when present
	"github.com/joho/godotenv"
)

func main() {
	// Load .env if present (ignored in production where real env vars are set)
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		log.Printf("[main] godotenv: %v (continuing)", err)
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("[main] config: %v", err)
	}

	// Wire dependencies
	waClient  := whatsapp.New(cfg.WAPhoneNumberID, cfg.WAAccessToken)
	sessions  := store.New()
	botHandler := bot.New(waClient, sessions)

	webhook := &webhookHandler{
		verifyToken: cfg.WAVerifyToken,
		bot:         botHandler,
	}

	mux := http.NewServeMux()
	mux.Handle("/webhook", webhook)
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintln(w, `{"status":"ok","service":"ataljanseva-wa-bot"}`)
	})

	addr := ":" + cfg.Port
	log.Printf("[main] Ataljanseva WhatsApp Bot listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("[main] server: %v", err)
	}
}
