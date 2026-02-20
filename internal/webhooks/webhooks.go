package webhooks

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

// EventType represents a webhook event type
type EventType string

const (
	EventMemoryCreated EventType = "memory.created"
	EventMemoryDeleted EventType = "memory.deleted"
	EventBundleCreated EventType = "bundle.created"
	EventBundleDeleted EventType = "bundle.deleted"
)

// WebhookPayload represents a webhook payload
type WebhookPayload struct {
	Event     string                 `json:"event"`
	Timestamp string                 `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
}

// WebhookConfig represents webhook configuration
type WebhookConfig struct {
	URL    string
	Secret string
	Events []EventType
}

// DeliverWebhook sends a webhook payload to the configured URL
func DeliverWebhook(config WebhookConfig, event EventType, data map[string]interface{}) error {
	payload := WebhookPayload{
		Event:     string(event),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Data:      data,
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook payload: %w", err)
	}

	// Create request
	req, err := http.NewRequest("POST", config.URL, bytes.NewBuffer(payloadJSON))
	if err != nil {
		return fmt.Errorf("failed to create webhook request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Cortex-Webhook/1.0")

	// Sign payload if secret is provided
	if config.Secret != "" {
		signature := signPayload(config.Secret, payloadJSON)
		req.Header.Set("X-Cortex-Signature", signature)
	}

	// Send request with timeout
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to deliver webhook: %w", err)
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook delivery failed with status %d", resp.StatusCode)
	}

	slog.Debug("webhook delivered", "url", config.URL, "event", event, "status", resp.StatusCode)
	return nil
}

// signPayload creates HMAC-SHA256 signature of the payload
func signPayload(secret string, payload []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

// VerifySignature verifies webhook signature
func VerifySignature(secret string, payload []byte, signature string) bool {
	expectedSignature := signPayload(secret, payload)
	return hmac.Equal([]byte(signature), []byte(expectedSignature))
}

// DeliverWebhooksAsync delivers webhooks asynchronously
func DeliverWebhooksAsync(configs []WebhookConfig, event EventType, data map[string]interface{}) {
	for _, config := range configs {
		// Check if event is subscribed
		subscribed := false
		for _, e := range config.Events {
			if e == event {
				subscribed = true
				break
			}
		}

		if !subscribed {
			continue
		}

		// Deliver asynchronously
		go func(cfg WebhookConfig) {
			if err := DeliverWebhook(cfg, event, data); err != nil {
				slog.Warn("webhook delivery failed", "url", cfg.URL, "event", event, "error", err)
			}
		}(config)
	}
}
