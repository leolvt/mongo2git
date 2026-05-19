package config

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Config holds all settings for a backup run.
type Config struct {
	MongoURI       string
	CollectionName string
	DumpDir        string
	CloneDir       string
	RepoURL        string
	GitBranch      string
	SlackURL       string
	Local          bool
	ShowVersion    bool
}

// ParseFlags reads CLI flags and env vars into a Config. Returns an error if any
// required setting is missing.
func ParseFlags() (Config, error) {
	fMongoURI := flag.String("mongo-uri", "", "MongoDB connection URI")
	fMongoColl := flag.String("mongo-collection", "", "MongoDB collection to dump")
	fDumpDir := flag.String("dump-dir", "", "Subdirectory for dumped JSON files")
	fCloneDir := flag.String("clone-dir", "", "Local directory where the git repo is cloned")
	fRepoURL := flag.String("repo-url", "", "Git repo SSH URL")
	fGitBranch := flag.String("git-branch", "", "Git branch to push to")
	fSlackURL := flag.String("slack-webhook-url", "", "Slack incoming webhook URL")
	fLocal := flag.Bool("local", false, "Commit only, no push; log notifications instead of Slack")
	fVersion := flag.Bool("version", false, "Print version and exit")

	// Short aliases
	flag.StringVar(fMongoURI, "m", "", "short for --mongo-uri")
	flag.StringVar(fMongoColl, "c", "", "short for --mongo-collection")
	flag.StringVar(fDumpDir, "d", "", "short for --dump-dir")
	flag.StringVar(fCloneDir, "g", "", "short for --clone-dir")
	flag.StringVar(fRepoURL, "r", "", "short for --repo-url")
	flag.StringVar(fGitBranch, "b", "", "short for --git-branch")
	flag.StringVar(fSlackURL, "s", "", "short for --slack-webhook-url")
	flag.BoolVar(fLocal, "L", false, "short for --local")
	flag.BoolVar(fVersion, "v", false, "short for --version")

	flag.Usage = func() {
		out := flag.CommandLine.Output()
		_, _ = fmt.Fprintf(out, "Usage: mongo2git [options]\n")
		_, _ = fmt.Fprintf(out, "\nRequired:\n")
		_, _ = fmt.Fprintf(out, "  --mongo-uri, -m         MongoDB connection URI (or MONGO_URI env)\n")
		_, _ = fmt.Fprintf(out, "  --mongo-collection, -c  MongoDB collection to dump (or MONGO_COLLECTION env)\n")
		_, _ = fmt.Fprintf(out, "  --dump-dir, -d          Subdirectory for dumped JSON files (or DUMP_DIR env)\n")
		_, _ = fmt.Fprintf(out, "  --clone-dir, -g         Local directory for cloned git repo (or CLONE_DIR env)\n")
		_, _ = fmt.Fprintf(out, "  --repo-url, -r          Git repo SSH URL (or REPO_URL env)\n")
		_, _ = fmt.Fprintf(out, "\nOptional:\n")
		_, _ = fmt.Fprintf(out, "  --git-branch, -b        Git branch to push to (or GIT_BRANCH env, default: main)\n")
		_, _ = fmt.Fprintf(out, "  --slack-webhook-url, -s Slack incoming webhook URL (or SLACK_WEBHOOK_URL env)\n")
		_, _ = fmt.Fprintf(out, "  --local, -L             Commit only, no push; log notifications instead of Slack\n")
		_, _ = fmt.Fprintf(out, "  --version, -v           Print version and exit\n")
	}

	flag.Parse()

	if *fVersion {
		return Config{ShowVersion: true}, nil
	}

	mongoURI, err := resolve(*fMongoURI, "MONGO_URI")
	if err != nil {
		return Config{}, err
	}
	collName, err := resolve(*fMongoColl, "MONGO_COLLECTION")
	if err != nil {
		return Config{}, err
	}
	dumpDir, err := resolve(*fDumpDir, "DUMP_DIR")
	if err != nil {
		return Config{}, err
	}
	repoURL, err := resolve(*fRepoURL, "REPO_URL")
	if err != nil {
		return Config{}, err
	}
	cloneDir, err := resolve(*fCloneDir, "CLONE_DIR")
	if err != nil {
		return Config{}, err
	}

	return Config{
		MongoURI:       mongoURI,
		CollectionName: collName,
		DumpDir:        dumpDir,
		RepoURL:        repoURL,
		CloneDir:       filepath.Clean(cloneDir),
		GitBranch:      resolveOptional(*fGitBranch, "GIT_BRANCH", "main"),
		SlackURL:       resolveOptional(*fSlackURL, "SLACK_WEBHOOK_URL", ""),
		Local:          *fLocal,
	}, nil
}

// DBNameFromURI extracts the database name from a MongoDB URI path.
func DBNameFromURI(uri string) (string, error) {
	if idx := strings.LastIndex(uri, "/"); idx != -1 {
		rest := uri[idx+1:]
		if q := strings.Index(rest, "?"); q != -1 {
			rest = rest[:q]
		}
		if rest != "" {
			return rest, nil
		}
	}
	return "", fmt.Errorf("no database name found in MongoDB URI")
}

// resolve returns flagVal if non-empty, otherwise the env var.
// Returns an error if neither is set.
func resolve(flagVal, envKey string) (string, error) {
	if flagVal != "" {
		return flagVal, nil
	}
	v := os.Getenv(envKey)
	if v == "" {
		return "", fmt.Errorf("%s must be set via --%s flag or %s env var", envKey, flagName(envKey), envKey)
	}
	return v, nil
}

// resolveOptional returns flagVal if non-empty, otherwise the env var,
// otherwise defaultVal.
func resolveOptional(flagVal, envKey, defaultVal string) string {
	if flagVal != "" {
		return flagVal
	}
	return orDefault(os.Getenv(envKey), defaultVal)
}

func orDefault(v, def string) string {
	if v == "" {
		return def
	}
	return v
}

func flagName(envKey string) string {
	return strings.ReplaceAll(strings.ToLower(envKey), "_", "-")
}

// Validate checks that the CloneDir parent exists and is writable.
func (c Config) Validate() error {
	parent := filepath.Dir(c.CloneDir)
	if _, err := os.Stat(parent); os.IsNotExist(err) {
		if err := os.MkdirAll(parent, 0755); err != nil {
			return fmt.Errorf("cannot create parent of CLONE_DIR %s: %w", parent, err)
		}
		return nil
	}
	f, err := os.CreateTemp(parent, ".mongo2git-check-*")
	if err != nil {
		return fmt.Errorf("parent of CLONE_DIR %s is not writable: %w", parent, err)
	}
	_ = f.Close()
	_ = os.Remove(f.Name())
	return nil
}
