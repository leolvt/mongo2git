package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDBNameFromURI(t *testing.T) {
	tests := []struct {
		name    string
		uri     string
		want    string
		wantErr bool
	}{
		{"standard URI", "mongodb://localhost:27017/mydb", "mydb", false},
		{"URI with query params", "mongodb://localhost:27017/mydb?replicaSet=rs0", "mydb", false},
		{"atlas-style URI", "mongodb+srv://user:pass@cluster.abc.mongodb.net/mydb", "mydb", false},
		{"URI with authSource", "mongodb://localhost:27017/mydb?authSource=admin", "mydb", false},
		{"database with dot", "mongodb://localhost:27017/my.db", "my.db", false},
		{"no database name", "mongodb://localhost:27017/", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DBNameFromURI(tt.uri)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestResolve(t *testing.T) {
	const key = "TEST_RESOLVE_VAR"
	tests := []struct {
		name    string
		flagVal string
		envVal  string
		want    string
		wantErr bool
	}{
		{"flag takes priority", "flag-value", "env-value", "flag-value", false},
		{"env used when flag empty", "", "env-value", "env-value", false},
		{"error when both missing", "", "", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envVal != "" {
				_ = os.Setenv(key, tt.envVal)
				t.Cleanup(func() { _ = os.Unsetenv(key) })
			}
			got, err := resolve(tt.flagVal, key)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestResolveOptional(t *testing.T) {
	const key = "TEST_RESOLVE_OPT_VAR"
	tests := []struct {
		name    string
		flagVal string
		envVal  string
		defVal  string
		want    string
	}{
		{"flag takes priority", "flag-branch", "env-branch", "main", "flag-branch"},
		{"env over default", "", "env-branch", "main", "env-branch"},
		{"default when both empty", "", "", "main", "main"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envVal != "" {
				_ = os.Setenv(key, tt.envVal)
				t.Cleanup(func() { _ = os.Unsetenv(key) })
			}
			got := resolveOptional(tt.flagVal, key, tt.defVal)
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestOrDefault(t *testing.T) {
	tests := []struct {
		name string
		v    string
		def  string
		want string
	}{
		{"non-empty returns self", "hello", "default", "hello"},
		{"empty returns default", "", "default", "default"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := orDefault(tt.v, tt.def); got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFlagName(t *testing.T) {
	tests := map[string]string{
		"MONGO_URI":         "mongo-uri",
		"MONGO_COLLECTION":  "mongo-collection",
		"REPO_URL":          "repo-url",
		"SLACK_WEBHOOK_URL": "slack-webhook-url",
	}
	for envKey, want := range tests {
		t.Run(envKey, func(t *testing.T) {
			if got := flagName(envKey); got != want {
				t.Fatalf("got %q, want %q", got, want)
			}
		})
	}
}

func TestConfigValidate(t *testing.T) {
	t.Run("writable parent", func(t *testing.T) {
		cloneDir := filepath.Join(t.TempDir(), "repo")
		c := Config{CloneDir: cloneDir}
		if err := c.Validate(); err != nil {
			t.Fatalf("expected nil, got %v", err)
		}
	})

	t.Run("unwritable parent", func(t *testing.T) {
		parent := t.TempDir()
		_ = os.Chmod(parent, 0o000)
		t.Cleanup(func() { _ = os.Chmod(parent, 0o755) })
		c := Config{CloneDir: filepath.Join(parent, "repo")}
		if err := c.Validate(); err == nil {
			t.Fatal("expected error for unwritable parent")
		}
	})
}
