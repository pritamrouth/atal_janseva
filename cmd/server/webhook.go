package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/ataljanseva/whatsapp-bot/internal/bot"
	"github.com/ataljanseva/whatsapp-bot/internal/whatsapp"
)

// webhookHandler handles both GET (hub verification) and POST (inbound events).
type webhookHandler struct {
	verifyToken string
	bot         *bot.Handler
}

// ServeHTTP dispatches GET / POST accordingly.
func (wh *webhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		wh.verify(w, r)
	case http.MethodPost:
		wh.receive(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// ─────────────────────────────────────────────
// GET /webhook  – hub.challenge verification
// ─────────────────────────────────────────────

func (wh *webhookHandler) verify(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	mode      := q.Get("hub.mode")
	token     := q.Get("hub.verify_token")
	challenge := q.Get("hub.challenge")

	if mode == "subscribe" && token == wh.verifyToken {
		log.Printf("[webhook] hub verification OK")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(challenge))
		return
	}
	log.Printf("[webhook] hub verification FAILED (mode=%q token=%q)", mode, token)
	http.Error(w, "Forbidden", http.StatusForbidden)
}

// ─────────────────────────────────────────────
// POST /webhook  – inbound messages
// ─────────────────────────────────────────────

func (wh *webhookHandler) receive(w http.ResponseWriter, r *http.Request) {
	// Always acknowledge immediately – Meta retries if it doesn't get 200 within 20s.
	w.WriteHeader(http.StatusOK)

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("[webhook] read body: %v", err)
		return
	}

	// Quick guard: only process "whatsapp_business_account" events
	var probe struct {
		Object string `json:"object"`
	}
	if err := json.Unmarshal(body, &probe); err != nil || probe.Object != "whatsapp_business_account" {
		return
	}

	payload, err := whatsapp.Parse(body)
	if err != nil {
		log.Printf("[webhook] parse: %v", err)
		return
	}

	for _, msg := range payload.Messages() {
		log.Printf("[webhook] msg from=%s type=%s", msg.From, msg.Type)
		wh.bot.HandleRaw(msg)
	}
}
