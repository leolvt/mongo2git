package git

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
)

// RepoOps abstracts git repository operations for backup.
type RepoOps interface {
	Prepare() error
	CommitAndPush(timestamp string, docCount int) (bool, error)
}

// GitRepo implements RepoOps using local git commands.
type GitRepo struct {
	cloneDir string
	repoURL  string
	branch   string
	local    bool
}

// NewRepo creates a GitRepo with the given settings.
func NewRepo(cloneDir, repoURL, branch string, local bool) *GitRepo {
	return &GitRepo{
		cloneDir: filepath.Clean(cloneDir),
		repoURL:  repoURL,
		branch:   branch,
		local:    local,
	}
}

// Prepare ensures the repository exists and is on the configured branch.
// Clones if missing; otherwise fetches, cleans leftovers, and checks out
// the target branch.
func (r *GitRepo) Prepare() error {
	gitDir := filepath.Join(r.cloneDir, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		slog.Info("cloning repo", "url", r.repoURL, "dest", r.cloneDir)
		parent := filepath.Dir(r.cloneDir)
		if err := os.MkdirAll(parent, 0755); err != nil {
			return fmt.Errorf("mkdir %s: %w", parent, err)
		}
		return runGit(parent, "clone", "--branch", r.branch, r.repoURL, r.cloneDir)
	}

	if err := runGit(r.cloneDir, "fetch", "origin"); err != nil {
		return fmt.Errorf("git fetch: %w", err)
	}
	// Clean up any leftover dirt from a previous failed run
	// so that checkout can proceed without conflicts.
	_ = runGit(r.cloneDir, "reset", "--hard", "HEAD")
	_ = runGit(r.cloneDir, "clean", "-fd")
	// checkout -B creates the branch if new, resets it if it
	// already exists, and switches to it — all in one command.
	if err := runGit(r.cloneDir, "checkout", "-B", r.branch, "origin/"+r.branch); err != nil {
		return fmt.Errorf("git checkout %s: %w", r.branch, err)
	}
	return nil
}

// CommitAndPush stages, commits (if dirty), and optionally pushes.
func (r *GitRepo) CommitAndPush(timestamp string, docCount int) (bool, error) {
	if err := runGit(r.cloneDir, "add", "-A"); err != nil {
		return false, fmt.Errorf("git add: %w", err)
	}

	diff := exec.Command("git", "diff", "--cached", "--quiet")
	diff.Dir = r.cloneDir
	err := diff.Run()
	if err == nil {
		return false, nil // exit 0: no diff
	}
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) || exitErr.ExitCode() != 1 {
		return false, fmt.Errorf("git diff failed: %w", err)
	}
	// exit 1: diff exists, proceed to commit

	msg := fmt.Sprintf("backup: %s — %d documents", timestamp, docCount)
	if err := runGit(r.cloneDir, "commit", "-m", msg); err != nil {
		return false, fmt.Errorf("git commit: %w", err)
	}
	slog.Info("committed", "message", msg)

	if r.local {
		slog.Info("push skipped", "reason", "local mode")
		return true, nil
	}

	if err := runGit(r.cloneDir, "push", "origin", r.branch); err != nil {
		return false, fmt.Errorf("git push: %w", err)
	}
	slog.Info("pushed to origin")
	return true, nil
}

func runGit(dir string, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()
	return cmd.Run()
}
