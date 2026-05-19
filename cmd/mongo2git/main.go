package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/leolvt/mongo2git/internal/config"
	"github.com/leolvt/mongo2git/internal/doc"
	"github.com/leolvt/mongo2git/internal/git"
	"github.com/leolvt/mongo2git/internal/mongo"
	"github.com/leolvt/mongo2git/internal/slack"

	"go.mongodb.org/mongo-driver/v2/bson"
)

var version = "dev"
var commit = "unknown"
var date = "unknown"

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{AddSource: true})))

	cfg, err := config.ParseFlags()
	if err != nil {
		slog.Error("configuration error", "error", err)
		os.Exit(1)
	}
	if cfg.ShowVersion {
		fmt.Printf("mongo2git %s (commit %s, built %s)\n", version, commit, date)
		os.Exit(0)
	}
	if err := cfg.Validate(); err != nil {
		slog.Error("configuration error", "error", err)
		os.Exit(1)
	}
	timestamp := time.Now().UTC().Format("2006-01-02T15-04-05Z")

	slog.Info("starting", "version", version, "commit", commit, "built", date)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	dbName, err := config.DBNameFromURI(cfg.MongoURI)
	if err != nil {
		slog.Error("MongoDB URI invalid", "error", err)
		os.Exit(1)
	}

	fetcher, cleanup, err := mongo.NewMongoFetcher(
		ctx, cfg.MongoURI, cfg.CollectionName, dbName,
	)
	if err != nil {
		slog.Error("MongoDB connection failed", "error", err)
		os.Exit(1)
	}
	defer cleanup()

	repo := git.NewRepo(cfg.CloneDir, cfg.RepoURL, cfg.GitBranch, cfg.Local)
	notifier := slack.NewNotifier(cfg.SlackURL, cfg.CollectionName, cfg.Local)

	if err := run(ctx, cfg, fetcher, repo, notifier, timestamp); err != nil {
		slog.Error("fatal error", "error", err)
		notifier.Notify(false, timestamp, 0, err.Error())
		os.Exit(1)
	}
}

// run executes the backup pipeline: fetch → prepare repo → write docs →
// commit/push → notify. All side effects go through injected interfaces,
// making it testable.
func run(ctx context.Context, cfg config.Config, fetcher mongo.Fetcher, repo git.RepoOps, notifier slack.Notifier, timestamp string) error {
	if err := repo.Prepare(); err != nil {
		return fmt.Errorf("git repo preparation failed: %w", err)
	}

	var docCount int
	err := fetcher.ForEach(ctx, func(d bson.M) error {
		docID, err := mongo.IDToFilename(d)
		if err != nil {
			slog.Warn("skipping document", "error", err)
			return nil
		}
		if err := doc.WriteDocument(cfg.CloneDir, cfg.DumpDir, docID, d); err != nil {
			return err
		}
		docCount++
		return nil
	})
	if err != nil {
		return fmt.Errorf("MongoDB fetch failed: %w", err)
	}

	changed, err := repo.CommitAndPush(timestamp, docCount)
	if err != nil {
		return fmt.Errorf("git commit/push failed: %w", err)
	}

	if changed {
		notifier.Notify(true, timestamp, docCount, "")
		slog.Info("backup complete", "documents", docCount, "timestamp", timestamp)
	} else {
		notifier.Notify(true, timestamp, docCount, "no changes (already up to date)")
		slog.Info("no changes to push", "documents_checked", docCount)
	}
	return nil
}
