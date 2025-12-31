package alerting

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

// AlertConfig holds alerting configuration.
type AlertConfig struct {
	// WebhookURL is a generic webhook endpoint (Slack, Discord, or custom)
	WebhookURL string
	// WebhookType determines the payload format: "slack", "discord", or "generic"
	WebhookType string
	// Enabled controls whether alerts are sent
	Enabled bool
	// MinFailuresBeforeAlert is the threshold before sending alerts
	MinFailuresBeforeAlert int
	// Timeout for HTTP requests
	Timeout time.Duration
}

// DefaultAlertConfig returns config from environment variables.
func DefaultAlertConfig() AlertConfig {
	cfg := AlertConfig{
		WebhookURL:             os.Getenv("ALERT_WEBHOOK_URL"),
		WebhookType:            os.Getenv("ALERT_WEBHOOK_TYPE"),
		MinFailuresBeforeAlert: 1,
		Timeout:                10 * time.Second,
	}

	cfg.Enabled = cfg.WebhookURL != ""

	if cfg.WebhookType == "" {
		// Auto-detect from URL
		if strings.Contains(cfg.WebhookURL, "slack.com") {
			cfg.WebhookType = "slack"
		} else if strings.Contains(cfg.WebhookURL, "discord.com") {
			cfg.WebhookType = "discord"
		} else {
			cfg.WebhookType = "generic"
		}
	}

	if v := os.Getenv("ALERT_MIN_FAILURES"); v != "" {
		var n int
		if _, err := fmt.Sscanf(v, "%d", &n); err == nil && n > 0 {
			cfg.MinFailuresBeforeAlert = n
		}
	}

	return cfg
}

// Alerter sends alerts to configured webhooks.
type Alerter struct {
	cfg    AlertConfig
	client *http.Client
}

// NewAlerter creates a new alerter instance.
func NewAlerter(cfg AlertConfig) *Alerter {
	return &Alerter{
		cfg: cfg,
		client: &http.Client{
			Timeout: cfg.Timeout,
		},
	}
}

// BatchAlert represents an alert about batch job results.
type BatchAlert struct {
	JobName       string
	TotalCount    int
	SuccessCount  int
	FailedCount   int
	Duration      time.Duration
	FailedDetails []ProviderFailure
	Timestamp     time.Time
}

// ProviderFailure contains details about a failed provider.
type ProviderFailure struct {
	Provider string
	Error    string
	Attempts int
}

// SendBatchAlert sends an alert about batch job failures.
func (a *Alerter) SendBatchAlert(ctx context.Context, alert BatchAlert) error {
	if !a.cfg.Enabled {
		log.Printf("alerting: alerts disabled, skipping")
		return nil
	}

	if alert.FailedCount < a.cfg.MinFailuresBeforeAlert {
		log.Printf("alerting: %d failures below threshold (%d), skipping",
			alert.FailedCount, a.cfg.MinFailuresBeforeAlert)
		return nil
	}

	var payload []byte
	var err error

	switch a.cfg.WebhookType {
	case "slack":
		payload, err = a.buildSlackPayload(alert)
	case "discord":
		payload, err = a.buildDiscordPayload(alert)
	default:
		payload, err = a.buildGenericPayload(alert)
	}

	if err != nil {
		return fmt.Errorf("build payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", a.cfg.WebhookURL, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	log.Printf("alerting: sent alert for %d failed providers", alert.FailedCount)
	return nil
}

func (a *Alerter) buildSlackPayload(alert BatchAlert) ([]byte, error) {
	// Build failure list
	var failedList strings.Builder
	for _, f := range alert.FailedDetails {
		failedList.WriteString(fmt.Sprintf("• *%s*: %s (attempts: %d)\n", f.Provider, f.Error, f.Attempts))
	}

	emoji := ":warning:"
	if alert.FailedCount == alert.TotalCount {
		emoji = ":x:"
	}

	payload := map[string]interface{}{
		"blocks": []map[string]interface{}{
			{
				"type": "header",
				"text": map[string]string{
					"type": "plain_text",
					"text": fmt.Sprintf("%s Batch Job Alert: %s", emoji, alert.JobName),
				},
			},
			{
				"type": "section",
				"fields": []map[string]string{
					{"type": "mrkdwn", "text": fmt.Sprintf("*Status:*\n%d/%d failed", alert.FailedCount, alert.TotalCount)},
					{"type": "mrkdwn", "text": fmt.Sprintf("*Duration:*\n%s", alert.Duration.Round(time.Millisecond))},
					{"type": "mrkdwn", "text": fmt.Sprintf("*Success:*\n%d", alert.SuccessCount)},
					{"type": "mrkdwn", "text": fmt.Sprintf("*Timestamp:*\n%s", alert.Timestamp.Format(time.RFC3339))},
				},
			},
			{
				"type": "section",
				"text": map[string]string{
					"type": "mrkdwn",
					"text": fmt.Sprintf("*Failed Providers:*\n%s", failedList.String()),
				},
			},
		},
	}

	return json.Marshal(payload)
}

func (a *Alerter) buildDiscordPayload(alert BatchAlert) ([]byte, error) {
	// Build failure list
	var failedList strings.Builder
	for _, f := range alert.FailedDetails {
		failedList.WriteString(fmt.Sprintf("• **%s**: %s (attempts: %d)\n", f.Provider, f.Error, f.Attempts))
	}

	color := 16776960 // Yellow
	if alert.FailedCount == alert.TotalCount {
		color = 16711680 // Red
	}

	payload := map[string]interface{}{
		"embeds": []map[string]interface{}{
			{
				"title":       fmt.Sprintf("Batch Job Alert: %s", alert.JobName),
				"description": fmt.Sprintf("%d/%d providers failed", alert.FailedCount, alert.TotalCount),
				"color":       color,
				"fields": []map[string]interface{}{
					{"name": "Success", "value": fmt.Sprintf("%d", alert.SuccessCount), "inline": true},
					{"name": "Failed", "value": fmt.Sprintf("%d", alert.FailedCount), "inline": true},
					{"name": "Duration", "value": alert.Duration.Round(time.Millisecond).String(), "inline": true},
					{"name": "Failed Providers", "value": failedList.String(), "inline": false},
				},
				"timestamp": alert.Timestamp.Format(time.RFC3339),
			},
		},
	}

	return json.Marshal(payload)
}

func (a *Alerter) buildGenericPayload(alert BatchAlert) ([]byte, error) {
	payload := map[string]interface{}{
		"alert_type":     "batch_job_failure",
		"job_name":       alert.JobName,
		"total_count":    alert.TotalCount,
		"success_count":  alert.SuccessCount,
		"failed_count":   alert.FailedCount,
		"duration_ms":    alert.Duration.Milliseconds(),
		"timestamp":      alert.Timestamp.Format(time.RFC3339),
		"failed_details": alert.FailedDetails,
	}

	return json.Marshal(payload)
}
