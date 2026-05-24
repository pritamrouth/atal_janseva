package main

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	"github.com/ataljanseva/whatsapp-bot/internal/whatsapp"
	"github.com/ataljanseva/whatsapp-bot/internal/worker"
)

// webhookHandler handles GET (hub verification) and POST (inbound events).
type webhookHandler struct {
	verifyToken string
	pool        *worker.Pool
}

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

// GET /webhook – hub.challenge verification
func (wh *webhookHandler) verify(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	mode      := q.Get("hub.mode")
	token     := q.Get("hub.verify_token")
	challenge := q.Get("hub.challenge")

	if mode == "subscribe" && token == wh.verifyToken {
		slog.Info("webhook hub verification OK")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(challenge))
		return
	}
	slog.Warn("webhook hub verification FAILED", "mode", mode)
	http.Error(w, "Forbidden", http.StatusForbidden)
}

// POST /webhook – inbound messages.
// Responds 200 immediately, enqueues work to the pool.
func (wh *webhookHandler) receive(w http.ResponseWriter, r *http.Request) {
	// ACK Meta immediately – must be within 20s or Meta will retry
	w.WriteHeader(http.StatusOK)

	body, err := io.ReadAll(r.Body)
	if err != nil {
		slog.Error("webhook read body", "err", err)
		return
	}

	var probe struct {
		Object string `json:"object"`
	}
	if err := json.Unmarshal(body, &probe); err != nil || probe.Object != "whatsapp_business_account" {
		return
	}

	payload, err := whatsapp.Parse(body)
	if err != nil {
		slog.Error("webhook parse", "err", err)
		return
	}

	for _, msg := range payload.Messages() {
		slog.Info("message received", "from", msg.From, "type", msg.Type)
		wh.pool.Enqueue(msg)
	}
}
