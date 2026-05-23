package slack

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

var httpClient = &http.Client{Timeout: 30 * time.Second}

// Notifier sends backup result notifications.
type Notifier interface {
	Notify(success bool, timestamp string, docCount int, detail string)
}

// SlackNotifier sends notifications to a Slack incoming webhook.
type SlackNotifier struct {
	webhookURL     string
	collectionName string
	repoURL        string
	branch         string
	hostname       string
	local          bool
}

// NewNotifier creates a SlackNotifier with the given hostname. If local is
// true, notifications are logged instead of sent.
func NewNotifier(webhookURL, collectionName, repoURL, branch, hostname string, local bool) *SlackNotifier {
	return &SlackNotifier{
		webhookURL:     webhookURL,
		collectionName: collectionName,
		repoURL:        repoURL,
		branch:         branch,
		hostname:       hostname,
		local:          local,
	}
}

// Notify sends or logs a backup result notification.
func (n *SlackNotifier) Notify(success bool, timestamp string, docCount int, detail string) {
	emoji := "✅"
	status := "succeeded"
	if !success {
		emoji = "❌"
		status = "failed"
	}

	body := fmt.Sprintf("%s *%s backup* %s\n• %d documents\n• Timestamp: `%s`\n• Host: `%s`\n• Repository: `%s` (branch: `%s`)",
		emoji, n.collectionName, status, docCount, timestamp, n.hostname, n.repoURL, n.branch)
	if detail != "" {
		body += "\n• " + detail
	}

	if n.local {
		slog.Info("notification", "body", body)
		return
	}

	if n.webhookURL == "" {
		return
	}

	payload := slackPayload{
		Text: body,
		Blocks: []slackBlock{
			{Type: "section", Text: slackText{Type: "mrkdwn", Text: body}},
		},
	}

	b, err := json.Marshal(payload)
	if err != nil {
		slog.Warn("failed to marshal Slack payload", "error", err)
		return
	}
	resp, err := httpClient.Post(n.webhookURL, "application/json", bytes.NewReader(b))
	if err != nil {
		slog.Warn("Slack notification failed", "error", err)
		return
	}
	if err := resp.Body.Close(); err != nil {
		slog.Warn("failed to close Slack response body", "error", err)
	}
	if resp.StatusCode >= 300 {
		slog.Warn("Slack returned non-OK status", "status", resp.StatusCode)
	}
}

type slackPayload struct {
	Text   string       `json:"text"`
	Blocks []slackBlock `json:"blocks,omitempty"`
}

type slackBlock struct {
	Type string    `json:"type"`
	Text slackText `json:"text"`
}

type slackText struct {
	Type string `json:"type"`
	Text string `json:"text"`
}
