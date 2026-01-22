package status

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestWebhookNotifierSuccess(t *testing.T) {
	var receivedPayload WebhookPayload
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read request body: %v", err)
		}
		if err := json.Unmarshal(body, &receivedPayload); err != nil {
			t.Fatalf("failed to unmarshal payload: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	notifier := NewWebhookNotifier(ts.URL)
	alert := Alert{
		ServiceURL: "http://example.com",
		AlertType:  AlertTypeDown,
		Message:    "Service down: http://example.com",
		Timestamp:  time.Now(),
	}

	err := notifier.Notify(alert)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if receivedPayload.ServiceURL != "http://example.com" {
		t.Errorf("expected service URL http://example.com, got %s", receivedPayload.ServiceURL)
	}
	if receivedPayload.AlertType != "down" {
		t.Errorf("expected alert type down, got %s", receivedPayload.AlertType)
	}
}

func TestWebhookNotifierFailure(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	notifier := NewWebhookNotifier(ts.URL)
	alert := Alert{
		ServiceURL: "http://example.com",
		AlertType:  AlertTypeDown,
		Message:    "Service down",
		Timestamp:  time.Now(),
	}

	err := notifier.Notify(alert)
	if err != ErrNotifyFailed {
		t.Errorf("expected ErrNotifyFailed, got %v", err)
	}
}

func TestWebhookNotifierInvalidURL(t *testing.T) {
	notifier := NewWebhookNotifier("invalid-url")
	alert := Alert{
		ServiceURL: "http://example.com",
		AlertType:  AlertTypeDown,
		Message:    "Service down",
		Timestamp:  time.Now(),
	}

	err := notifier.Notify(alert)
	if err == nil {
		t.Error("expected error for invalid URL, got nil")
	}
}

func TestWebhookNotifierType(t *testing.T) {
	notifier := NewWebhookNotifier("http://example.com")
	if notifier.Type() != "webhook" {
		t.Errorf("expected type webhook, got %s", notifier.Type())
	}
}

func TestSlackNotifierSuccess(t *testing.T) {
	var receivedPayload SlackPayload
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read request body: %v", err)
		}
		if err := json.Unmarshal(body, &receivedPayload); err != nil {
			t.Fatalf("failed to unmarshal payload: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	notifier := NewSlackNotifier(ts.URL, "#alerts", "StatusBot")
	alert := Alert{
		ServiceURL: "http://example.com",
		AlertType:  AlertTypeDown,
		Message:    "Service down: http://example.com",
		Timestamp:  time.Now(),
	}

	err := notifier.Notify(alert)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if receivedPayload.Channel != "#alerts" {
		t.Errorf("expected channel #alerts, got %s", receivedPayload.Channel)
	}
	if receivedPayload.Username != "StatusBot" {
		t.Errorf("expected username StatusBot, got %s", receivedPayload.Username)
	}
	if len(receivedPayload.Attachments) != 1 {
		t.Fatalf("expected 1 attachment, got %d", len(receivedPayload.Attachments))
	}
	if receivedPayload.Attachments[0].Color != "#FF0000" {
		t.Errorf("expected red color, got %s", receivedPayload.Attachments[0].Color)
	}
}

func TestSlackNotifierRecovery(t *testing.T) {
	var receivedPayload SlackPayload
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read request body: %v", err)
		}
		if err := json.Unmarshal(body, &receivedPayload); err != nil {
			t.Fatalf("failed to unmarshal payload: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	notifier := NewSlackNotifier(ts.URL, "#alerts", "StatusBot")
	alert := Alert{
		ServiceURL: "http://example.com",
		AlertType:  AlertTypeRecovery,
		Message:    "Service recovered: http://example.com",
		Timestamp:  time.Now(),
	}

	err := notifier.Notify(alert)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if len(receivedPayload.Attachments) != 1 {
		t.Fatalf("expected 1 attachment, got %d", len(receivedPayload.Attachments))
	}
	if receivedPayload.Attachments[0].Color != "#00FF00" {
		t.Errorf("expected green color for recovery, got %s", receivedPayload.Attachments[0].Color)
	}
}

func TestSlackNotifierType(t *testing.T) {
	notifier := NewSlackNotifier("http://example.com", "#alerts", "StatusBot")
	if notifier.Type() != "slack" {
		t.Errorf("expected type slack, got %s", notifier.Type())
	}
}

func TestDiscordNotifierSuccess(t *testing.T) {
	var receivedPayload DiscordPayload
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read request body: %v", err)
		}
		if err := json.Unmarshal(body, &receivedPayload); err != nil {
			t.Fatalf("failed to unmarshal payload: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	notifier := NewDiscordNotifier(ts.URL, "StatusBot")
	alert := Alert{
		ServiceURL: "http://example.com",
		AlertType:  AlertTypeDown,
		Message:    "Service down: http://example.com",
		Timestamp:  time.Now(),
	}

	err := notifier.Notify(alert)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if receivedPayload.Username != "StatusBot" {
		t.Errorf("expected username StatusBot, got %s", receivedPayload.Username)
	}
	if len(receivedPayload.Embeds) != 1 {
		t.Fatalf("expected 1 embed, got %d", len(receivedPayload.Embeds))
	}
	if receivedPayload.Embeds[0].Color != 16711680 {
		t.Errorf("expected red color (16711680), got %d", receivedPayload.Embeds[0].Color)
	}
}

func TestDiscordNotifierRecovery(t *testing.T) {
	var receivedPayload DiscordPayload
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read request body: %v", err)
		}
		if err := json.Unmarshal(body, &receivedPayload); err != nil {
			t.Fatalf("failed to unmarshal payload: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	notifier := NewDiscordNotifier(ts.URL, "StatusBot")
	alert := Alert{
		ServiceURL: "http://example.com",
		AlertType:  AlertTypeRecovery,
		Message:    "Service recovered: http://example.com",
		Timestamp:  time.Now(),
	}

	err := notifier.Notify(alert)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if len(receivedPayload.Embeds) != 1 {
		t.Fatalf("expected 1 embed, got %d", len(receivedPayload.Embeds))
	}
	if receivedPayload.Embeds[0].Color != 65280 {
		t.Errorf("expected green color (65280) for recovery, got %d", receivedPayload.Embeds[0].Color)
	}
}

func TestDiscordNotifierType(t *testing.T) {
	notifier := NewDiscordNotifier("http://example.com", "StatusBot")
	if notifier.Type() != "discord" {
		t.Errorf("expected type discord, got %s", notifier.Type())
	}
}

func TestLogNotifierSuccess(t *testing.T) {
	notifier := NewLogNotifier()
	alert := Alert{
		ServiceURL: "http://example.com",
		AlertType:  AlertTypeDown,
		Message:    "Service down: http://example.com",
		Timestamp:  time.Now(),
	}

	err := notifier.Notify(alert)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestLogNotifierType(t *testing.T) {
	notifier := NewLogNotifier()
	if notifier.Type() != "log" {
		t.Errorf("expected type log, got %s", notifier.Type())
	}
}

func TestNotificationManagerCheckAndNotify(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	nm := NewNotificationManager(0) // no cooldown for testing
	nm.AddNotifier(NewWebhookNotifier(ts.URL))

	// First check - service goes down (no previous state)
	sent := nm.CheckAndNotify("http://example.com", false)
	if !sent {
		t.Error("expected notification to be sent for new down service")
	}

	// Same state - should not send
	sent = nm.CheckAndNotify("http://example.com", false)
	if sent {
		t.Error("expected no notification for same state")
	}

	// Service recovers
	sent = nm.CheckAndNotify("http://example.com", true)
	if !sent {
		t.Error("expected notification to be sent for recovery")
	}
}

func TestNotificationManagerCooldown(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	nm := NewNotificationManager(1 * time.Hour) // long cooldown
	nm.AddNotifier(NewWebhookNotifier(ts.URL))

	// First notification
	sent := nm.CheckAndNotify("http://example.com", false)
	if !sent {
		t.Error("expected notification to be sent")
	}

	// Service recovers but cooldown active
	sent = nm.CheckAndNotify("http://example.com", true)
	if sent {
		t.Error("expected no notification during cooldown")
	}
}

func TestNotificationManagerGetServiceState(t *testing.T) {
	nm := NewNotificationManager(0)

	// Set some states
	nm.CheckAndNotify("http://service1.com", true)
	nm.CheckAndNotify("http://service2.com", false)

	state := nm.GetServiceState()

	if !state["http://service1.com"] {
		t.Error("expected service1 to be up")
	}
	if state["http://service2.com"] {
		t.Error("expected service2 to be down")
	}
}

func TestNotificationManagerConcurrency(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	nm := NewNotificationManager(0)
	nm.AddNotifier(NewWebhookNotifier(ts.URL))

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			nm.CheckAndNotify("http://example.com", i%2 == 0)
		}(i)
	}
	wg.Wait()

	// Just verify no race conditions - state should be consistent
	state := nm.GetServiceState()
	if _, exists := state["http://example.com"]; !exists {
		t.Error("expected service state to exist")
	}
}

func TestWebhookFactoryCreate(t *testing.T) {
	f := WebhookFactory{}

	config := NotifierConfig{Type: "webhook", WebhookURL: "http://example.com"}
	notifier, err := f.Create(config)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if notifier.Type() != "webhook" {
		t.Errorf("expected type webhook, got %s", notifier.Type())
	}
}

func TestWebhookFactoryCreateInvalid(t *testing.T) {
	f := WebhookFactory{}

	config := NotifierConfig{Type: "slack", WebhookURL: "http://example.com"}
	_, err := f.Create(config)
	if err != ErrInvalidNotifier {
		t.Errorf("expected ErrInvalidNotifier, got %v", err)
	}
}

func TestSlackFactoryCreate(t *testing.T) {
	f := SlackFactory{}

	config := NotifierConfig{Type: "slack", WebhookURL: "http://example.com", Channel: "#alerts", Username: "Bot"}
	notifier, err := f.Create(config)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if notifier.Type() != "slack" {
		t.Errorf("expected type slack, got %s", notifier.Type())
	}
}

func TestSlackFactoryCreateInvalid(t *testing.T) {
	f := SlackFactory{}

	config := NotifierConfig{Type: "webhook", WebhookURL: "http://example.com"}
	_, err := f.Create(config)
	if err != ErrInvalidNotifier {
		t.Errorf("expected ErrInvalidNotifier, got %v", err)
	}
}

func TestDiscordFactoryCreate(t *testing.T) {
	f := DiscordFactory{}

	config := NotifierConfig{Type: "discord", WebhookURL: "http://example.com", Username: "Bot"}
	notifier, err := f.Create(config)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if notifier.Type() != "discord" {
		t.Errorf("expected type discord, got %s", notifier.Type())
	}
}

func TestDiscordFactoryCreateInvalid(t *testing.T) {
	f := DiscordFactory{}

	config := NotifierConfig{Type: "webhook", WebhookURL: "http://example.com"}
	_, err := f.Create(config)
	if err != ErrInvalidNotifier {
		t.Errorf("expected ErrInvalidNotifier, got %v", err)
	}
}

func TestLogFactoryCreate(t *testing.T) {
	f := LogFactory{}

	config := NotifierConfig{Type: "log"}
	notifier, err := f.Create(config)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if notifier.Type() != "log" {
		t.Errorf("expected type log, got %s", notifier.Type())
	}
}

func TestLogFactoryCreateInvalid(t *testing.T) {
	f := LogFactory{}

	config := NotifierConfig{Type: "webhook"}
	_, err := f.Create(config)
	if err != ErrInvalidNotifier {
		t.Errorf("expected ErrInvalidNotifier, got %v", err)
	}
}

func TestCreateNotifier(t *testing.T) {
	tt := []struct {
		name        string
		config      NotifierConfig
		expected    string
		expectError bool
	}{
		{
			name:        "webhook",
			config:      NotifierConfig{Type: "webhook", WebhookURL: "http://example.com"},
			expected:    "webhook",
			expectError: false,
		},
		{
			name:        "slack",
			config:      NotifierConfig{Type: "slack", WebhookURL: "http://example.com", Channel: "#alerts"},
			expected:    "slack",
			expectError: false,
		},
		{
			name:        "discord",
			config:      NotifierConfig{Type: "discord", WebhookURL: "http://example.com"},
			expected:    "discord",
			expectError: false,
		},
		{
			name:        "log",
			config:      NotifierConfig{Type: "log"},
			expected:    "log",
			expectError: false,
		},
		{
			name:        "invalid",
			config:      NotifierConfig{Type: "invalid"},
			expected:    "",
			expectError: true,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			notifier, err := CreateNotifier(tc.config)
			if tc.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if notifier.Type() != tc.expected {
				t.Errorf("expected type %s, got %s", tc.expected, notifier.Type())
			}
		})
	}
}

func TestAlertTypes(t *testing.T) {
	if AlertTypeDown != "down" {
		t.Errorf("expected AlertTypeDown to be 'down', got %s", AlertTypeDown)
	}
	if AlertTypeRecovery != "recovery" {
		t.Errorf("expected AlertTypeRecovery to be 'recovery', got %s", AlertTypeRecovery)
	}
}

func TestNotificationManagerMultipleNotifiers(t *testing.T) {
	callCount := 0
	var mu sync.Mutex

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		callCount++
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	nm := NewNotificationManager(0)
	nm.AddNotifier(NewWebhookNotifier(ts.URL))
	nm.AddNotifier(NewWebhookNotifier(ts.URL))
	nm.AddNotifier(NewWebhookNotifier(ts.URL))

	nm.CheckAndNotify("http://example.com", false)

	mu.Lock()
	if callCount != 3 {
		t.Errorf("expected 3 notifications, got %d", callCount)
	}
	mu.Unlock()
}

func TestNotificationManagerNoNotifiers(t *testing.T) {
	nm := NewNotificationManager(0)

	// Should not panic with no notifiers
	sent := nm.CheckAndNotify("http://example.com", false)
	if sent {
		t.Error("expected no notification with no notifiers")
	}
}

func TestNotificationManagerSetStorage(t *testing.T) {
	// Create a temporary storage
	tmpDir := t.TempDir()
	storage, err := NewStorage(tmpDir + "/test.db")
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}
	defer storage.Close()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	nm := NewNotificationManager(0)
	nm.AddNotifier(NewWebhookNotifier(ts.URL))
	nm.SetStorage(storage)

	// Trigger a notification
	sent := nm.CheckAndNotify("http://example.com", false)
	if !sent {
		t.Error("expected notification to be sent")
	}

	// Verify alert was recorded in storage
	alerts, err := storage.GetRecentAlerts(10)
	if err != nil {
		t.Fatalf("failed to get recent alerts: %v", err)
	}
	if len(alerts) != 1 {
		t.Fatalf("expected 1 alert, got %d", len(alerts))
	}
	if alerts[0].ServiceURL != "http://example.com" {
		t.Errorf("expected service URL http://example.com, got %s", alerts[0].ServiceURL)
	}
	if alerts[0].AlertType != "down" {
		t.Errorf("expected alert type down, got %s", alerts[0].AlertType)
	}
}

func TestNotificationManagerSetStorageNil(t *testing.T) {
	nm := NewNotificationManager(0)
	nm.SetStorage(nil) // Should not panic

	// Should still work without storage
	sent := nm.CheckAndNotify("http://example.com", false)
	if sent {
		t.Error("expected no notification with no notifiers")
	}
}
