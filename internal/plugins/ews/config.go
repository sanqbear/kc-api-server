package ews

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds Exchange Web Services configuration
type Config struct {
	ServerURL             string
	ImpersonationUsername string
	ImpersonationPassword string
	Domain                string
	Timeout               time.Duration
	MaxRetries            int
	SkipTLSVerify         bool
}

// LoadConfig reads EWS configuration from environment variables
func LoadConfig() (*Config, error) {
	cfg := &Config{
		ServerURL:             getEnv("EWS_SERVER_URL", ""),
		ImpersonationUsername: getEnv("EWS_IMPERSONATION_USERNAME", ""),
		ImpersonationPassword: getEnv("EWS_IMPERSONATION_PASSWORD", ""),
		Domain:                getEnv("EWS_DOMAIN", ""),
		Timeout:               getDurationEnv("EWS_TIMEOUT", 30*time.Second),
		MaxRetries:            getIntEnv("EWS_MAX_RETRIES", 3),
		SkipTLSVerify:         getBoolEnv("EWS_SKIP_TLS_VERIFY", false),
	}

	// EWS is optional - return nil config if not configured
	if cfg.ServerURL == "" {
		return nil, nil
	}

	// Validate configuration if ServerURL is provided
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate checks if the EWS configuration is valid
func (c *Config) Validate() error {
	if c.ServerURL == "" {
		return fmt.Errorf("EWS_SERVER_URL is required")
	}

	if err := ValidateEWSURL(c.ServerURL); err != nil {
		return err
	}

	if c.ImpersonationUsername == "" {
		return fmt.Errorf("EWS_IMPERSONATION_USERNAME is required when EWS_SERVER_URL is set")
	}

	if c.ImpersonationPassword == "" {
		return fmt.Errorf("EWS_IMPERSONATION_PASSWORD is required when EWS_SERVER_URL is set")
	}

	return nil
}

// IsEnabled returns true if EWS is configured
func (c *Config) IsEnabled() bool {
	return c != nil && c.ServerURL != ""
}

// FolderName constants for well-known Exchange folders
const (
	FolderInbox        = "inbox"
	FolderSentItems    = "sentitems"
	FolderDrafts       = "drafts"
	FolderDeletedItems = "deleteditems"
	FolderJunkEmail    = "junkemail"
	FolderOutbox       = "outbox"
)

// EWS API version
const (
	EWSVersionExchange2013    = "Exchange2013"
	EWSVersionExchange2013SP1 = "Exchange2013_SP1"
	EWSVersionExchange2016    = "Exchange2016"
)

// GetFolderID maps common folder names to EWS DistinguishedFolderId values
func GetFolderID(folderName string) string {
	folderMap := map[string]string{
		"inbox":        "inbox",
		"sent":         "sentitems",
		"sentitems":    "sentitems",
		"drafts":       "drafts",
		"deleted":      "deleteditems",
		"deleteditems": "deleteditems",
		"junk":         "junkemail",
		"junkemail":    "junkemail",
		"outbox":       "outbox",
		"archive":      "archiveinbox",
	}

	normalized := strings.ToLower(strings.TrimSpace(folderName))
	if id, ok := folderMap[normalized]; ok {
		return id
	}

	return "inbox"
}

// ValidateMailbox performs basic validation on email address format
func ValidateMailbox(mailbox string) error {
	if mailbox == "" {
		return fmt.Errorf("mailbox email address is required")
	}

	if !strings.Contains(mailbox, "@") {
		return fmt.Errorf("invalid mailbox email address format")
	}

	parts := strings.Split(mailbox, "@")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return fmt.Errorf("invalid mailbox email address format")
	}

	return nil
}

// ValidateEWSURL validates the Exchange server URL
func ValidateEWSURL(ewsURL string) error {
	if ewsURL == "" {
		return fmt.Errorf("EWS server URL is required")
	}

	parsedURL, err := url.Parse(ewsURL)
	if err != nil {
		return fmt.Errorf("invalid EWS server URL: %w", err)
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("EWS server URL must use http or https scheme")
	}

	return nil
}

// SanitizeLimit ensures pagination limit is within acceptable bounds
func SanitizeLimit(limit int) int {
	const (
		DefaultLimit = 50
		MaxLimit     = 100
		MinLimit     = 1
	)

	if limit <= 0 {
		return DefaultLimit
	}

	if limit > MaxLimit {
		return MaxLimit
	}

	return limit
}

// SanitizeOffset ensures pagination offset is non-negative
func SanitizeOffset(offset int) int {
	if offset < 0 {
		return 0
	}
	return offset
}

// GetEWSAPIVersion returns the EWS API version to use
func GetEWSAPIVersion() string {
	return EWSVersionExchange2013SP1
}

// Helper functions for environment variables
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

func getBoolEnv(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}
