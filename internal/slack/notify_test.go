package slack

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNotifier_Local(t *testing.T) {
	n := NewNotifier("https://hooks.slack.com/services/xxx", "my-coll", true)
	n.Notify(true, "2024-01-01T00-00-00Z", 42, "")
	// Should not panic or make HTTP calls.
}

func TestNotifier_NoWebhook(t *testing.T) {
	n := NewNotifier("", "my-coll", false)
	n.Notify(true, "2024-01-01T00-00-00Z", 10, "")
	// Should return early without error.
}

func TestNotifier_Success(t *testing.T) {
	var received struct {
		Text   string `json:"text"`
		Blocks []struct {
			Type string `json:"type"`
			Text struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"text"`
		} `json:"blocks"`
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", ct)
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Errorf("failed to decode payload: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	n := NewNotifier(server.URL, "my-collection", false)
	n.Notify(true, "2024-05-15T10-30-00Z", 99, "")

	if !strings.Contains(received.Text, "my-collection") {
		t.Errorf("expected collection name in payload, got: %s", received.Text)
	}
	if !strings.Contains(received.Text, "99") {
		t.Errorf("expected doc count, got: %s", received.Text)
	}
	if !strings.Contains(received.Text, "✅") {
		t.Errorf("expected success emoji, got: %s", received.Text)
	}
	if len(received.Blocks) != 1 {
		t.Errorf("expected 1 block, got %d", len(received.Blocks))
	}
}

func TestNotifier_Failure(t *testing.T) {
	var received struct{ Text string }
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&received)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	n := NewNotifier(server.URL, "failing-coll", false)
	n.Notify(false, "2024-05-15T10-30-00Z", 0, "something broke")

	if !strings.Contains(received.Text, "❌") {
		t.Errorf("expected failure emoji, got: %s", received.Text)
	}
	if !strings.Contains(received.Text, "failed") {
		t.Errorf("expected 'failed' status, got: %s", received.Text)
	}
	if !strings.Contains(received.Text, "something broke") {
		t.Errorf("expected detail text, got: %s", received.Text)
	}
}

func TestMockNotifier(t *testing.T) {
	m := &MockNotifier{}
	m.Notify(true, "ts", 5, "ok")
	if !m.SuccessCalled {
		t.Error("expected SuccessCalled")
	}
	m.Notify(false, "ts", 0, "boom")
	if !m.FailureCalled {
		t.Error("expected FailureCalled")
	}
	if m.LastDetail != "boom" {
		t.Errorf("expected LastDetail 'boom', got %q", m.LastDetail)
	}
}
