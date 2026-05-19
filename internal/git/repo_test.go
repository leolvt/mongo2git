package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// initTestRepo creates a temporary git repo with an initial commit.
func initTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	_ = runGit(dir, "init")
	_ = runGit(dir, "config", "user.email", "test@test.com")
	_ = runGit(dir, "config", "user.name", "Test")
	_ = os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Test"), 0644)
	_ = runGit(dir, "add", "README.md")
	_ = runGit(dir, "commit", "-m", "initial")
	// Allow pushing to this non-bare repo during tests.
	_ = runGit(dir, "config", "receive.denyCurrentBranch", "ignore")
	return dir
}

func TestPrepare_Clone(t *testing.T) {
	remote := initTestRepo(t)
	cloneDir := filepath.Join(t.TempDir(), "clone")

	r := NewRepo(cloneDir, remote, "main", true)
	if err := r.Prepare(); err != nil {
		t.Fatalf("Prepare() = %v", err)
	}

	if _, err := os.Stat(filepath.Join(cloneDir, ".git")); os.IsNotExist(err) {
		t.Fatal(".git directory not found after clone")
	}
	if _, err := os.Stat(filepath.Join(cloneDir, "README.md")); os.IsNotExist(err) {
		t.Fatal("README.md not found after clone")
	}
}

func TestPrepare_FetchReset(t *testing.T) {
	remote := initTestRepo(t)
	cloneDir := filepath.Join(t.TempDir(), "clone")

	// First clone
	r := NewRepo(cloneDir, remote, "main", true)
	if err := r.Prepare(); err != nil {
		t.Fatalf("first Prepare() = %v", err)
	}

	// Add a new file to the remote
	_ = os.WriteFile(filepath.Join(remote, "newfile.txt"), []byte("hello"), 0644)
	_ = runGit(remote, "add", "newfile.txt")
	_ = runGit(remote, "commit", "-m", "add newfile")

	// Second Prepare should fetch and reset
	if err := r.Prepare(); err != nil {
		t.Fatalf("second Prepare() = %v", err)
	}

	if _, err := os.Stat(filepath.Join(cloneDir, "newfile.txt")); os.IsNotExist(err) {
		t.Fatal("newfile.txt not found after fetch+reset")
	}
}

func TestCommitAndPush_NoChanges(t *testing.T) {
	remote := initTestRepo(t)
	cloneDir := filepath.Join(t.TempDir(), "clone")

	r := NewRepo(cloneDir, remote, "main", true)
	if err := r.Prepare(); err != nil {
		t.Fatalf("Prepare() = %v", err)
	}

	changed, err := r.CommitAndPush("2024-01-01T00-00-00Z", 0)
	if err != nil {
		t.Fatalf("CommitAndPush() = %v", err)
	}
	if changed {
		t.Error("expected no changes")
	}
}

func TestCommitAndPush_WithChanges(t *testing.T) {
	remote := initTestRepo(t)
	cloneDir := filepath.Join(t.TempDir(), "clone")

	r := NewRepo(cloneDir, remote, "main", true)
	if err := r.Prepare(); err != nil {
		t.Fatalf("Prepare() = %v", err)
	}

	// Write a document file
	_ = os.MkdirAll(filepath.Join(cloneDir, "data"), 0755)
	_ = os.WriteFile(filepath.Join(cloneDir, "data", "doc.json"), []byte(`{"hello":"world"}`), 0644)

	changed, err := r.CommitAndPush("2024-01-01T00-00-00Z", 1)
	if err != nil {
		t.Fatalf("CommitAndPush() = %v", err)
	}
	if !changed {
		t.Error("expected changes to be detected")
	}

	// Verify commit was created
	out, err := exec.Command("git", "-C", cloneDir, "log", "--oneline").Output()
	if err != nil {
		t.Fatalf("git log: %v", err)
	}
	if len(out) == 0 {
		t.Fatal("expected at least one commit")
	}
}

func TestCommitAndPush_NotAGitRepo(t *testing.T) {
	cloneDir := t.TempDir()
	r := &GitRepo{cloneDir: cloneDir, local: true}

	changed, err := r.CommitAndPush("2024-01-01T00-00-00Z", 0)
	if err == nil {
		t.Fatal("expected error from non-git directory")
	}
	if changed {
		t.Error("expected no changes")
	}
}
