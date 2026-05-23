package config

import (
	"fmt"
	"os"
)

// Config holds all runtime configuration loaded from environment variables.
type Config struct {
	// WhatsApp Cloud API
	WAPhoneNumberID string // WABA phone-number ID (from Meta dashboard)
	WAAccessToken   string // Permanent / temporary system-user token
	WAVerifyToken   string // Arbitrary secret you set on the webhook page

	// Server
	Port string // HTTP listen port (default: 8080)
}

// Load reads config from environment (or a .env file loaded by main).
func Load() (*Config, error) {
	c := &Config{
		WAPhoneNumberID: os.Getenv("WA_PHONE_NUMBER_ID"),
		WAAccessToken:   os.Getenv("WA_ACCESS_TOKEN"),
		WAVerifyToken:   os.Getenv("WA_VERIFY_TOKEN"),
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
	if c.Port == "" {
		c.Port = "8080"
	}
	return c, nil
}
