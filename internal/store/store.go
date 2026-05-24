// Package store provides a Redis-backed session store for WhatsApp conversations.
// Sessions are stored as JSON with a 24-hour TTL.
// Key format:  wa:session:<e164_phone>
package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	sessionTTL    = 24 * time.Hour
	keyPrefix     = "wa:session:"
)

// Step represents the current step in the onboarding flow.
type Step int

const (
	StepStart      Step = iota // 0 – show language picker
	StepLangChosen             // 1 – language chosen, waiting for PIN
	StepWardChosen             // 2 – PIN valid, waiting for ward selection
	StepNagarsevak             // 3 – ward chosen, waiting for nagarsevak selection
	StepMainMenu               // 4 – fully onboarded, showing main menu
)

// Session holds the state for a single WhatsApp user.
type Session struct {
	PhoneNumber    string `json:"phone"`
	Step           Step   `json:"step"`
	Lang           string `json:"lang"`            // "en" | "mr" | "hi"
	Pincode        string `json:"pincode"`
	State          string `json:"state"`
	District       string `json:"district"`
	Ward           string `json:"ward"`
	NagarsevakID   string `json:"nagarsevak_id"`
	NagarsevakName string `json:"nagarsevak_name"`
	Pending        string `json:"pending"`
}

// Store is a Redis-backed session store.
type Store struct {
	rdb *redis.Client
}

// New returns an initialised Store and pings Redis.
func New(rdb *redis.Client) (*Store, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping: %w", err)
	}
	return &Store{rdb: rdb}, nil
}

// Get returns the session for a phone number.
// If none exists, a fresh session is returned (not yet written to Redis).
func (s *Store) Get(ctx context.Context, phone string) (*Session, error) {
	val, err := s.rdb.Get(ctx, key(phone)).Result()
	if errors.Is(err, redis.Nil) {
		return &Session{PhoneNumber: phone, Step: StepStart}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("redis GET %s: %w", phone, err)
	}
	var sess Session
	if err := json.Unmarshal([]byte(val), &sess); err != nil {
		// Corrupted session – start fresh and log
		slog.Warn("corrupted session, resetting", "phone", phone, "err", err)
		return &Session{PhoneNumber: phone, Step: StepStart}, nil
	}
	return &sess, nil
}

// Save persists a session with a refreshed TTL.
func (s *Store) Save(ctx context.Context, sess *Session) error {
	b, err := json.Marshal(sess)
	if err != nil {
		return fmt.Errorf("marshal session: %w", err)
	}
	if err := s.rdb.Set(ctx, key(sess.PhoneNumber), b, sessionTTL).Err(); err != nil {
		return fmt.Errorf("redis SET %s: %w", sess.PhoneNumber, err)
	}
	return nil
}

// Reset deletes a session (user typed "reset").
func (s *Store) Reset(ctx context.Context, phone string) error {
	if err := s.rdb.Del(ctx, key(phone)).Err(); err != nil && !errors.Is(err, redis.Nil) {
		return fmt.Errorf("redis DEL %s: %w", phone, err)
	}
	return nil
}

func key(phone string) string {
	return keyPrefix + phone
}
