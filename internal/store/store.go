// Package store provides an in-memory session store for WhatsApp conversations.
// In production, replace with Redis or a persistent database.
package store

import (
	"sync"
)

// Step represents the current step in the onboarding flow.
type Step int

const (
	StepStart       Step = iota // 0 – just arrived, show language picker
	StepLangChosen              // 1 – language chosen, waiting for PIN
	StepWardChosen              // 2 – PIN valid, waiting for ward selection
	StepNagarsevak              // 3 – ward chosen, waiting for nagarsevak selection
	StepMainMenu                // 4 – fully onboarded, showing main menu
)

// Session holds the state for a single WhatsApp user.
type Session struct {
	PhoneNumber string
	Step        Step
	Lang        string // "en" | "mr" | "hi"

	// Location resolved from DB
	Pincode  string
	State    string
	District string
	Ward     string // ward code, e.g. "17C"

	// Nagarsevak DB UUID chosen by the user
	NagarsevakID   string
	NagarsevakName string

	// Pending context key to route list-reply
	Pending string
}

// Store is a thread-safe in-memory session map.
type Store struct {
	mu       sync.RWMutex
	sessions map[string]*Session
}

// New returns an initialised Store.
func New() *Store {
	return &Store{sessions: make(map[string]*Session)}
}

// Get returns the session for a phone number, creating a new one if absent.
func (s *Store) Get(phone string) *Session {
	s.mu.Lock()
	defer s.mu.Unlock()
	if sess, ok := s.sessions[phone]; ok {
		return sess
	}
	sess := &Session{PhoneNumber: phone, Step: StepStart}
	s.sessions[phone] = sess
	return sess
}

// Save persists an updated session.
func (s *Store) Save(sess *Session) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[sess.PhoneNumber] = sess
}

// Reset removes a session (e.g. user types "reset").
func (s *Store) Reset(phone string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, phone)
}
