package status

import (
	"bytes"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"sync"
	"time"
)

// NotifyError implements error signifying notification failures.
var (
	ErrNotifyFailed    = errors.New("notify: notification delivery failed")
	ErrInvalidNotifier = errors.New("notify: invalid notifier type")
)

// isSuccessStatus checks if an HTTP status code indicates success (2xx range).
func isSuccessStatus(code int) bool {
	return code >= 200 && code < 300
}

// AlertType represents the type of alert being sent.
type AlertType string

const (
	AlertTypeDown     AlertType = "down"
	AlertTypeRecovery AlertType = "recovery"
)

// Alert represents a notification event.
type Alert struct {
	ServiceURL string
	AlertType  AlertType
	Message    string
	Timestamp  time.Time
}

// Notifier is an interface that describes how to send notifications.
type Notifier interface {
	Notify(alert Alert) error
	Type() string
}

// NotifierFactory is a single method interface that describes
// how to create a Notifier object.
type NotifierFactory interface {
	Create(config NotifierConfig) (Notifier, error)
}

// NotifierConfig holds configuration for a notification channel.
type NotifierConfig struct {
	Type       string `json:"type"`
	WebhookURL string `json:"webhook_url,omitempty"`
	Channel    string `json:"channel,omitempty"`
	Username   string `json:"username,omitempty"`
}

// NotificationManager manages notification state and cooldowns.
type NotificationManager struct {
	notifiers    []Notifier
	cooldown     time.Duration
	lastAlert    map[string]time.Time
	serviceState map[string]bool // true = up, false = down
	mu           sync.RWMutex
}

// NewNotificationManager creates a new NotificationManager with the given cooldown period.
func NewNotificationManager(cooldown time.Duration) *NotificationManager {
	return &NotificationManager{
		notifiers:    nil,
		cooldown:     cooldown,
		lastAlert:    make(map[string]time.Time),
		serviceState: make(map[string]bool),
	}
}

// AddNotifier adds a notifier to the manager.
func (nm *NotificationManager) AddNotifier(n Notifier) {
	nm.mu.Lock()
	defer nm.mu.Unlock()
	nm.notifiers = append(nm.notifiers, n)
}

// CheckAndNotify checks if a service status changed and sends notifications.
// Returns true if notifications were sent.
func (nm *NotificationManager) CheckAndNotify(serviceURL string, isUp bool) bool {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	prevState, exists := nm.serviceState[serviceURL]
	nm.serviceState[serviceURL] = isUp

	// No change in state, no notification needed
	if exists && prevState == isUp {
		return false
	}

	// Check cooldown
	if lastTime, ok := nm.lastAlert[serviceURL]; ok {
		if time.Since(lastTime) < nm.cooldown {
			return false
		}
	}

	// Determine alert type
	var alertType AlertType
	var message string
	if isUp {
		alertType = AlertTypeRecovery
		message = "Service recovered: " + serviceURL
	} else {
		alertType = AlertTypeDown
		message = "Service down: " + serviceURL
	}

	alert := Alert{
		ServiceURL: serviceURL,
		AlertType:  alertType,
		Message:    message,
		Timestamp:  time.Now(),
	}

	// Send to all notifiers
	sent := false
	for _, notifier := range nm.notifiers {
		if err := notifier.Notify(alert); err != nil {
			log.Printf("notification error (%s): %v", notifier.Type(), err)
		} else {
			sent = true
		}
	}

	if sent {
		nm.lastAlert[serviceURL] = time.Now()
	}

	return sent
}

// GetServiceState returns the current state of all tracked services.
func (nm *NotificationManager) GetServiceState() map[string]bool {
	nm.mu.RLock()
	defer nm.mu.RUnlock()

	state := make(map[string]bool)
	for k, v := range nm.serviceState {
		state[k] = v
	}
	return state
}

// WebhookNotifier sends notifications to a generic webhook URL.
type WebhookNotifier struct {
	webhookURL string
	client     *http.Client
}

// WebhookPayload is the JSON payload sent to webhooks.
type WebhookPayload struct {
	ServiceURL string `json:"service_url"`
	AlertType  string `json:"alert_type"`
	Message    string `json:"message"`
	Timestamp  string `json:"timestamp"`
}

// NewWebhookNotifier creates a new WebhookNotifier.
func NewWebhookNotifier(webhookURL string) *WebhookNotifier {
	return &WebhookNotifier{
		webhookURL: webhookURL,
		client:     &http.Client{Timeout: 10 * time.Second},
	}
}

// Notify sends an alert to the webhook URL.
func (w *WebhookNotifier) Notify(alert Alert) error {
	payload := WebhookPayload{
		ServiceURL: alert.ServiceURL,
		AlertType:  string(alert.AlertType),
		Message:    alert.Message,
		Timestamp:  alert.Timestamp.Format(time.RFC3339),
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	resp, err := w.client.Post(w.webhookURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if !isSuccessStatus(resp.StatusCode) {
		return ErrNotifyFailed
	}

	return nil
}

// Type returns the notifier type.
func (w *WebhookNotifier) Type() string {
	return "webhook"
}

// WebhookFactory implements the NotifierFactory interface.
type WebhookFactory struct{}

// Create returns a pointer to a WebhookNotifier.
func (f *WebhookFactory) Create(config NotifierConfig) (Notifier, error) {
	if config.Type != "webhook" {
		return nil, ErrInvalidNotifier
	}
	return NewWebhookNotifier(config.WebhookURL), nil
}

// SlackNotifier sends notifications to Slack via incoming webhook.
type SlackNotifier struct {
	webhookURL string
	channel    string
	username   string
	client     *http.Client
}

// SlackPayload is the JSON payload sent to Slack webhooks.
type SlackPayload struct {
	Channel     string            `json:"channel,omitempty"`
	Username    string            `json:"username,omitempty"`
	Text        string            `json:"text"`
	Attachments []SlackAttachment `json:"attachments,omitempty"`
}

// SlackAttachment represents a Slack message attachment.
type SlackAttachment struct {
	Color string `json:"color"`
	Title string `json:"title"`
	Text  string `json:"text"`
}

// NewSlackNotifier creates a new SlackNotifier.
func NewSlackNotifier(webhookURL, channel, username string) *SlackNotifier {
	return &SlackNotifier{
		webhookURL: webhookURL,
		channel:    channel,
		username:   username,
		client:     &http.Client{Timeout: 10 * time.Second},
	}
}

// Notify sends an alert to Slack.
func (s *SlackNotifier) Notify(alert Alert) error {
	color := "#FF0000" // red for down
	if alert.AlertType == AlertTypeRecovery {
		color = "#00FF00" // green for recovery
	}

	payload := SlackPayload{
		Channel:  s.channel,
		Username: s.username,
		Attachments: []SlackAttachment{
			{
				Color: color,
				Title: string(alert.AlertType),
				Text:  alert.Message,
			},
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	resp, err := s.client.Post(s.webhookURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if !isSuccessStatus(resp.StatusCode) {
		return ErrNotifyFailed
	}

	return nil
}

// Type returns the notifier type.
func (s *SlackNotifier) Type() string {
	return "slack"
}

// SlackFactory implements the NotifierFactory interface.
type SlackFactory struct{}

// Create returns a pointer to a SlackNotifier.
func (f *SlackFactory) Create(config NotifierConfig) (Notifier, error) {
	if config.Type != "slack" {
		return nil, ErrInvalidNotifier
	}
	return NewSlackNotifier(config.WebhookURL, config.Channel, config.Username), nil
}

// DiscordNotifier sends notifications to Discord via webhook.
type DiscordNotifier struct {
	webhookURL string
	username   string
	client     *http.Client
}

// DiscordPayload is the JSON payload sent to Discord webhooks.
type DiscordPayload struct {
	Username string         `json:"username,omitempty"`
	Content  string         `json:"content,omitempty"`
	Embeds   []DiscordEmbed `json:"embeds,omitempty"`
}

// DiscordEmbed represents a Discord embed.
type DiscordEmbed struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Color       int    `json:"color"`
}

// NewDiscordNotifier creates a new DiscordNotifier.
func NewDiscordNotifier(webhookURL, username string) *DiscordNotifier {
	return &DiscordNotifier{
		webhookURL: webhookURL,
		username:   username,
		client:     &http.Client{Timeout: 10 * time.Second},
	}
}

// Notify sends an alert to Discord.
func (d *DiscordNotifier) Notify(alert Alert) error {
	color := 16711680 // red (0xFF0000) for down
	if alert.AlertType == AlertTypeRecovery {
		color = 65280 // green (0x00FF00) for recovery
	}

	payload := DiscordPayload{
		Username: d.username,
		Embeds: []DiscordEmbed{
			{
				Title:       string(alert.AlertType),
				Description: alert.Message,
				Color:       color,
			},
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	resp, err := d.client.Post(d.webhookURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if !isSuccessStatus(resp.StatusCode) {
		return ErrNotifyFailed
	}

	return nil
}

// Type returns the notifier type.
func (d *DiscordNotifier) Type() string {
	return "discord"
}

// DiscordFactory implements the NotifierFactory interface.
type DiscordFactory struct{}

// Create returns a pointer to a DiscordNotifier.
func (f *DiscordFactory) Create(config NotifierConfig) (Notifier, error) {
	if config.Type != "discord" {
		return nil, ErrInvalidNotifier
	}
	return NewDiscordNotifier(config.WebhookURL, config.Username), nil
}

// LogNotifier sends notifications to the standard logger (useful for testing/debugging).
type LogNotifier struct{}

// NewLogNotifier creates a new LogNotifier.
func NewLogNotifier() *LogNotifier {
	return &LogNotifier{}
}

// Notify logs the alert.
func (l *LogNotifier) Notify(alert Alert) error {
	log.Printf("[%s] %s: %s at %s",
		alert.AlertType,
		alert.ServiceURL,
		alert.Message,
		alert.Timestamp.Format(time.RFC3339))
	return nil
}

// Type returns the notifier type.
func (l *LogNotifier) Type() string {
	return "log"
}

// LogFactory implements the NotifierFactory interface.
type LogFactory struct{}

// Create returns a pointer to a LogNotifier.
func (f *LogFactory) Create(config NotifierConfig) (Notifier, error) {
	if config.Type != "log" {
		return nil, ErrInvalidNotifier
	}
	return NewLogNotifier(), nil
}

// CreateNotifier creates a notifier based on the configuration type.
func CreateNotifier(config NotifierConfig) (Notifier, error) {
	switch config.Type {
	case "webhook":
		return NewWebhookNotifier(config.WebhookURL), nil
	case "slack":
		return NewSlackNotifier(config.WebhookURL, config.Channel, config.Username), nil
	case "discord":
		return NewDiscordNotifier(config.WebhookURL, config.Username), nil
	case "log":
		return NewLogNotifier(), nil
	default:
		return nil, ErrInvalidNotifier
	}
}
