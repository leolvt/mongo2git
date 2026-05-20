package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// initTestRepo creates a temporary git repo on "main" with an initial commit.
func initTestRepo(t *testing.T) string {
	return initTestRepoWithBranch(t, "main")
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

	// Configure git identity in the clone (needed in CI where no global config exists).
	_ = runGit(cloneDir, "config", "user.email", "test@test.com")
	_ = runGit(cloneDir, "config", "user.name", "Test")

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

// initTestRepoWithBranch creates a repo on a given branch and returns its path.
func initTestRepoWithBranch(t *testing.T, branch string) string {
	t.Helper()
	dir := t.TempDir()
	_ = runGit(dir, "init", "-b", branch)
	_ = runGit(dir, "config", "user.email", "test@test.com")
	_ = runGit(dir, "config", "user.name", "Test")
	_ = os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Test"), 0644)
	_ = runGit(dir, "add", "README.md")
	_ = runGit(dir, "commit", "-m", "initial")
	_ = runGit(dir, "config", "receive.denyCurrentBranch", "ignore")
	return dir
}

// initTestRepoWithBranches creates a repo with multiple branches. Returns the
// repo path. The active branch after creation is the one passed as 'defaultBranch'.
func initTestRepoWithBranches(t *testing.T, defaultBranch string, branches ...string) string {
	t.Helper()
	dir := initTestRepoWithBranch(t, defaultBranch)
	for _, b := range branches {
		_ = runGit(dir, "checkout", "-b", b)
		fname := "file-" + b + ".txt"
		_ = os.WriteFile(filepath.Join(dir, fname), []byte(b), 0644)
		_ = runGit(dir, "add", fname)
		_ = runGit(dir, "commit", "-m", "add "+fname)
	}
	// Switch back to default branch.
	_ = runGit(dir, "checkout", defaultBranch)
	return dir
}

// TestPrepare_CloneNonDefaultBranch verifies that cloning with a non-"main"
// branch sets up the working tree on that branch.
func TestPrepare_CloneNonDefaultBranch(t *testing.T) {
	remote := initTestRepoWithBranches(t, "main", "staging")
	cloneDir := filepath.Join(t.TempDir(), "clone")

	r := NewRepo(cloneDir, remote, "staging", true)
	if err := r.Prepare(); err != nil {
		t.Fatalf("Prepare() = %v", err)
	}

	out, err := exec.Command("git", "-C", cloneDir, "branch", "--show-current").Output()
	if err != nil {
		t.Fatalf("git branch --show-current: %v", err)
	}
	got := strings.TrimSpace(string(out))
	if got != "staging" {
		t.Fatalf("expected branch %q, got %q", "staging", got)
	}

	// Verify the file that only exists on staging is present.
	if _, err := os.Stat(filepath.Join(cloneDir, "file-staging.txt")); os.IsNotExist(err) {
		t.Fatal("file-staging.txt not found — wrong branch cloned?")
	}
}

// TestPrepare_SwitchBranchOnExistingClone checks that Prepare() switches to
// the configured branch when the existing clone is on a different branch.
func TestPrepare_SwitchBranchOnExistingClone(t *testing.T) {
	remote := initTestRepoWithBranches(t, "main", "develop")
	cloneDir := filepath.Join(t.TempDir(), "clone")

	// First clone with main (simulating an old run or default).
	rMain := NewRepo(cloneDir, remote, "main", true)
	if err := rMain.Prepare(); err != nil {
		t.Fatalf("first Prepare() = %v", err)
	}

	// Now create a new GitRepo that wants the develop branch.
	rDev := NewRepo(cloneDir, remote, "develop", true)
	if err := rDev.Prepare(); err != nil {
		t.Fatalf("second Prepare() = %v", err)
	}

	out, err := exec.Command("git", "-C", cloneDir, "branch", "--show-current").Output()
	if err != nil {
		t.Fatalf("git branch --show-current: %v", err)
	}
	got := strings.TrimSpace(string(out))
	if got != "develop" {
		t.Fatalf("expected branch %q, got %q", "develop", got)
	}

	// The file unique to develop must exist.
	if _, err := os.Stat(filepath.Join(cloneDir, "file-develop.txt")); os.IsNotExist(err) {
		t.Fatal("file-develop.txt not found — branch switch failed?")
	}
}

// TestCommitAndPush_NonDefaultBranch pushes to a non-"main" branch and
// verifies the commit lands on the correct remote branch.
func TestCommitAndPush_NonDefaultBranch(t *testing.T) {
	remote := initTestRepoWithBranches(t, "main", "staging")
	cloneDir := filepath.Join(t.TempDir(), "clone")

	r := NewRepo(cloneDir, remote, "staging", false)
	if err := r.Prepare(); err != nil {
		t.Fatalf("Prepare() = %v", err)
	}

	// Configure git identity.
	_ = runGit(cloneDir, "config", "user.email", "test@test.com")
	_ = runGit(cloneDir, "config", "user.name", "Test")

	// Write a document.
	_ = os.MkdirAll(filepath.Join(cloneDir, "data"), 0755)
	_ = os.WriteFile(filepath.Join(cloneDir, "data", "doc.json"), []byte(`{"x":1}`), 0644)

	changed, err := r.CommitAndPush("2024-06-01T00-00-00Z", 1)
	if err != nil {
		t.Fatalf("CommitAndPush() = %v", err)
	}
	if !changed {
		t.Fatal("expected changes to be detected")
	}

	// The commit must show up on the staging branch of the remote.
	out, err := exec.Command("git", "-C", remote, "log", "--oneline", "staging").Output()
	if err != nil {
		t.Fatalf("git log staging: %v", err)
	}
	if len(out) == 0 {
		t.Fatal("no commits on staging branch after push")
	}
	if !strings.Contains(string(out), "2024-06-01T00-00-00Z") {
		t.Fatalf("backup commit not found on staging branch:\n%s", out)
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
