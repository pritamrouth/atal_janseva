package config

import (
	"fmt"
	"os"
)

// Config holds all runtime configuration loaded from environment variables.
type Config struct {
	// WhatsApp Cloud API
	WAPhoneNumberID string
	WAAccessToken   string
	WAVerifyToken   string

	// PostgreSQL
	DatabaseURL string // postgres://user:pass@host:5432/dbname?sslmode=disable

	// Server
	Port string
}

// Load reads config from environment.
func Load() (*Config, error) {
	c := &Config{
		WAPhoneNumberID: os.Getenv("WA_PHONE_NUMBER_ID"),
		WAAccessToken:   os.Getenv("WA_ACCESS_TOKEN"),
		WAVerifyToken:   os.Getenv("WA_VERIFY_TOKEN"),
		DatabaseURL:     os.Getenv("DATABASE_URL"),
		Port:            os.Getenv("PORT"),
	}
	if c.WAPhoneNumberID == "" {
		return nil, fmt.Errorf("WA_PHONE_NUMBER_ID is required")
	}
	if c.WAAccessToken == "" {
		return nil, fmt.Errorf("WA_ACCESS_TOKEN is required")
	}
	if c.WAVerifyToken == "" {
		return nil, fmt.Errorf("WA_VERIFY_TOKEN is required")
	}
	if c.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	if c.Port == "" {
		c.Port = "8080"
	}
	return c, nil
}
