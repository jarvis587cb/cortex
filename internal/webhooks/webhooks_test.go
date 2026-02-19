package webhooks

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestDeliverWebhook(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}

		if r.Header.Get("Content-Type") != "application/json" {
			t.Error("Content-Type should be application/json")
		}

		var payload WebhookPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("failed to decode payload: %v", err)
		}

		if payload.Event != string(EventMemoryCreated) {
			t.Errorf("expected event %s, got %s", EventMemoryCreated, payload.Event)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := WebhookConfig{
		URL:    server.URL,
		Secret: "test-secret",
		Events: []EventType{EventMemoryCreated},
	}

	data := map[string]interface{}{
		"id":   1,
		"content": "test",
	}

	err := DeliverWebhook(config, EventMemoryCreated, data)
	if err != nil {
		t.Errorf("webhook delivery failed: %v", err)
	}
}

func TestSignPayload(t *testing.T) {
	secret := "test-secret"
	payload := []byte(`{"test":"data"}`)

	signature1 := signPayload(secret, payload)
	signature2 := signPayload(secret, payload)

	if signature1 != signature2 {
		t.Error("signatures should be identical for same payload")
	}

	if signature1 == "" {
		t.Error("signature should not be empty")
	}

	// Verify signature
	if !VerifySignature(secret, payload, signature1) {
		t.Error("signature verification should succeed")
	}

	// Wrong secret should fail
	if VerifySignature("wrong-secret", payload, signature1) {
		t.Error("signature verification with wrong secret should fail")
	}
}

func TestDeliverWebhooksAsync(t *testing.T) {
	received := make(chan bool, 2)

	// Create test servers
	server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received <- true
		w.WriteHeader(http.StatusOK)
	}))
	defer server1.Close()

	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received <- true
		w.WriteHeader(http.StatusOK)
	}))
	defer server2.Close()

	configs := []WebhookConfig{
		{
			URL:    server1.URL,
			Events: []EventType{EventMemoryCreated},
		},
		{
			URL:    server2.URL,
			Events: []EventType{EventMemoryCreated},
		},
	}

	data := map[string]interface{}{"test": "data"}
	DeliverWebhooksAsync(configs, EventMemoryCreated, data)

	// Wait for webhooks to be delivered
	time.Sleep(100 * time.Millisecond)

	// Check that both webhooks were received
	select {
	case <-received:
		select {
		case <-received:
			// Both received
		case <-time.After(1 * time.Second):
			t.Error("second webhook not received")
		}
	case <-time.After(1 * time.Second):
		t.Error("webhooks not received")
	}
}

func TestDeliverWebhooksAsyncFiltered(t *testing.T) {
	received := make(chan bool, 1)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received <- true
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	configs := []WebhookConfig{
		{
			URL:    server.URL,
			Events: []EventType{EventMemoryDeleted}, // Different event
		},
	}

	data := map[string]interface{}{"test": "data"}
	DeliverWebhooksAsync(configs, EventMemoryCreated, data)

	// Wait a bit
	time.Sleep(100 * time.Millisecond)

	// Should not receive webhook for different event
	select {
	case <-received:
		t.Error("webhook should not be delivered for different event")
	case <-time.After(200 * time.Millisecond):
		// Expected - no webhook delivered
	}
}
