package main

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/leolvt/mongo2git/internal/config"
	"github.com/leolvt/mongo2git/internal/git"
	"github.com/leolvt/mongo2git/internal/mongo"
	"github.com/leolvt/mongo2git/internal/slack"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type failingRepo struct{ err error }

func (r *failingRepo) Prepare() error                          { return nil }
func (r *failingRepo) CommitAndPush(string, int) (bool, error) { return false, r.err }

func TestRun_Success(t *testing.T) {
	cfg := config.Config{
		CloneDir: t.TempDir(),
		DumpDir:  "data",
	}
	fetcher := &mongo.MockFetcher{
		Docs: []bson.M{{"_id": "doc-1", "name": "test"}},
	}
	repo := &git.MockRepo{Changed: true}
	notifier := &slack.MockNotifier{}

	err := run(context.Background(), cfg, fetcher, repo, notifier, "2024-01-01T00-00-00Z")
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}
	if !notifier.SuccessCalled {
		t.Error("expected success notification")
	}
}

func TestRun_NoChanges(t *testing.T) {
	cfg := config.Config{
		CloneDir: t.TempDir(),
		DumpDir:  "data",
	}
	fetcher := &mongo.MockFetcher{
		Docs: []bson.M{{"_id": "doc-1", "name": "test"}},
	}
	repo := &git.MockRepo{Changed: false}
	notifier := &slack.MockNotifier{}

	err := run(context.Background(), cfg, fetcher, repo, notifier, "2024-01-01T00-00-00Z")
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}
	if !notifier.SuccessCalled {
		t.Error("expected success notification")
	}
	if !strings.Contains(notifier.LastDetail, "no changes") {
		t.Errorf("expected 'no changes' in detail, got: %s", notifier.LastDetail)
	}
}

func TestRun_FetchError(t *testing.T) {
	fetcher := &mongo.MockFetcher{Err: errors.New("connection refused")}
	repo := &git.MockRepo{}
	notifier := &slack.MockNotifier{}

	err := run(context.Background(), config.Config{}, fetcher, repo, notifier, "")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "MongoDB fetch failed") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRun_PrepareError(t *testing.T) {
	cfg := config.Config{CloneDir: t.TempDir()}
	fetcher := &mongo.MockFetcher{Docs: []bson.M{{"_id": "x"}}}
	repo := &git.MockRepo{PrepareErr: errors.New("clone failed")}
	notifier := &slack.MockNotifier{}

	err := run(context.Background(), cfg, fetcher, repo, notifier, "")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "git repo preparation failed") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRun_CommitPushError(t *testing.T) {
	cfg := config.Config{
		CloneDir: t.TempDir(),
		DumpDir:  "data",
	}
	fetcher := &mongo.MockFetcher{
		Docs: []bson.M{{"_id": "doc-1", "name": "test"}},
	}
	repo := &failingRepo{err: errors.New("push rejected")}
	notifier := &slack.MockNotifier{}

	err := run(context.Background(), cfg, fetcher, repo, notifier, "")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "git commit/push failed") {
		t.Errorf("unexpected error: %v", err)
	}
}
