// Package inactivity tracks per-user message timestamps in Redis and fires a
// timeout callback when a user has been silent for InactivityTimeout.
//
// Design:
//   - On every inbound message, Touch(phone) resets the deadline.
//   - A single background goroutine scans every ScanInterval for expired entries.
//   - When a user times out, the callback sends the inactivity message and the
//     session key is deleted from Redis so the next "Hi" starts fresh.
//
// Redis keys used:
//
//	wa:active:<phone>   STRING  –  unix-nano timestamp of last message
//	                               TTL = InactivityTimeout + 30 s (auto-cleanup)
package inactivity

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	// InactivityTimeout is how long a user may be silent before the warning fires.
	InactivityTimeout = 90 * time.Second // 1 minute 30 seconds

	// ScanInterval controls how often the background goroutine checks for timeouts.
	// Shorter = more accurate; 10 s gives ±10 s accuracy with negligible CPU.
	ScanInterval = 10 * time.Second

	activeKeyPrefix  = "wa:active:"
	sessionKeyPrefix = "wa:session:"
	activeTTL        = InactivityTimeout + 30*time.Second
)

// TimeoutFunc is called when a user times out.
// phone is the E.164 number (no "+").
// lang is the user's last-known language ("en", "mr", "hi", or "" for unknown).
type TimeoutFunc func(ctx context.Context, phone, lang string)

// Monitor manages inactivity tracking.
type Monitor struct {
	rdb     *redis.Client
	onTimer TimeoutFunc
}

// New creates a Monitor. Call Start() to launch the background goroutine.
func New(rdb *redis.Client, onTimer TimeoutFunc) *Monitor {
	return &Monitor{rdb: rdb, onTimer: onTimer}
}

// Touch records "user phone just sent a message" — resets their deadline.
func (m *Monitor) Touch(ctx context.Context, phone string) {
	key := activeKeyPrefix + phone
	now := strconv.FormatInt(time.Now().UnixNano(), 10)
	if err := m.rdb.Set(ctx, key, now, activeTTL).Err(); err != nil {
		slog.Warn("inactivity.Touch redis SET failed", "phone", phone, "err", err)
	}
}

// Cancel removes the inactivity entry for a phone (e.g. user reset manually).
func (m *Monitor) Cancel(ctx context.Context, phone string) {
	m.rdb.Del(ctx, activeKeyPrefix+phone)
}

// Start launches the background scanner goroutine. It runs until ctx is cancelled.
func (m *Monitor) Start(ctx context.Context) {
	slog.Info("inactivity monitor started",
		"timeout", InactivityTimeout,
		"scan_interval", ScanInterval,
	)
	go m.loop(ctx)
}

func (m *Monitor) loop(ctx context.Context) {
	ticker := time.NewTicker(ScanInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("inactivity monitor stopped")
			return
		case <-ticker.C:
			m.scan(ctx)
		}
	}
}

// scan iterates over all wa:active:* keys and fires the callback for any that
// have exceeded InactivityTimeout.
func (m *Monitor) scan(ctx context.Context) {
	var cursor uint64
	pattern := activeKeyPrefix + "*"

	for {
		keys, nextCursor, err := m.rdb.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			if !errors.Is(err, context.Canceled) {
				slog.Warn("inactivity scan SCAN error", "err", err)
			}
			return
		}

		for _, key := range keys {
			m.checkKey(ctx, key)
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}
}

func (m *Monitor) checkKey(ctx context.Context, key string) {
	val, err := m.rdb.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return // already expired / deleted
	}
	if err != nil {
		slog.Warn("inactivity checkKey GET", "key", key, "err", err)
		return
	}

	lastNano, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		// Corrupt value — clean up
		m.rdb.Del(ctx, key)
		return
	}

	elapsed := time.Since(time.Unix(0, lastNano))
	if elapsed < InactivityTimeout {
		return // still active
	}

	// Extract phone from key
	phone := key[len(activeKeyPrefix):]

	slog.Info("inactivity timeout firing", "phone", phone, "idle", elapsed.Round(time.Second))

	// Read the user's language from their session BEFORE we delete it,
	// so the timeout message can be sent in their preferred language.
	lang := m.sessionLang(ctx, phone)

	// Delete the active-tracking key first so we don't double-fire
	m.rdb.Del(ctx, key)

	// Delete the session so the next message starts fresh
	sessionKey := sessionKeyPrefix + phone
	if delErr := m.rdb.Del(ctx, sessionKey).Err(); delErr != nil && !errors.Is(delErr, redis.Nil) {
		slog.Warn("inactivity: failed to delete session", "phone", phone, "err", delErr)
	}

	// Fire the callback with the user's language
	cbCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	m.onTimer(cbCtx, phone, lang)
}

// sessionLang reads the "lang" field from the session JSON in Redis.
// Returns "" if the session is missing or unparseable (caller uses "en" fallback).
func (m *Monitor) sessionLang(ctx context.Context, phone string) string {
	raw, err := m.rdb.Get(ctx, sessionKeyPrefix+phone).Result()
	if err != nil {
		return ""
	}
	// Minimal parse — only unmarshal the lang field to avoid a circular import.
	var s struct {
		Lang string `json:"lang"`
	}
	if err := json.Unmarshal([]byte(raw), &s); err != nil {
		return ""
	}
	return s.Lang
}

// InactivityMessage returns the inactivity warning text for a given language.
// lang may be empty / unknown — falls back to English.
func InactivityMessage(lang string) string {
	switch lang {
	case "mr":
		return `⏰ *सत्र कालबाह्य झाले*

असे दिसते की आपण काही वेळ निष्क्रिय होता.

आपल्या सुरक्षिततेसाठी आपचे सत्र संपुष्टात आले आहे.

पुन्हा सुरू करण्यासाठी *"Hi"* टाइप करा 🙏`

	case "hi":
		return `⏰ *सत्र समाप्त हो गया*

ऐसा लगता है कि आप कुछ समय से निष्क्रिय थे.

आपकी सुरक्षा के लिए आपका सत्र समाप्त कर दिया गया है.

फिर से शुरू करने के लिए *"Hi"* टाइप करें 🙏`

	default: // "en" and unknown
		return fmt.Sprintf(`⏰ *Session Timed Out*

It seems you've been inactive for a while.

For your security, your session has been cleared.

To continue, please type *"Hi"* to access the main menu 🙏`)
	}
}