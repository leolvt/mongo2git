package doc

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestWriteDocument(t *testing.T) {
	cloneDir := t.TempDir()
	dumpDir := "data"
	docID := "507f1f77bcf86cd799439011"
	doc := bson.M{"_id": bson.NewObjectID(), "name": "test doc", "count": int32(42)}

	if err := WriteDocument(cloneDir, dumpDir, docID, doc); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(cloneDir, dumpDir, docID+".json"))
	if err != nil {
		t.Fatalf("file not written: %v", err)
	}

	var parsed interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("not valid JSON: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "$oid") {
		t.Errorf("expected Extended JSON with $oid, got: %s", content)
	}
	if !strings.HasSuffix(content, "\n") {
		t.Errorf("expected trailing newline")
	}
}

func TestWriteDocumentNested(t *testing.T) {
	cloneDir := t.TempDir()
	doc := bson.M{
		"_id":     "nested-doc",
		"profile": bson.M{"name": "Alice", "email": "alice@example.com"},
		"tags":    bson.A{"go", "mongodb", "git"},
	}

	if err := WriteDocument(cloneDir, "dumps", "nested-doc", doc); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(cloneDir, "dumps", "nested-doc.json"))
	if err != nil {
		t.Fatalf("file not written: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("not valid JSON: %v", err)
	}
}

func TestWriteDocument_Deterministic(t *testing.T) {
	// Same document written twice must produce byte-for-byte identical output.
	cloneDir := t.TempDir()
	doc := bson.M{
		"zebra": 1,
		"alpha": "first",
		"mike":  bson.M{"nested2": 2, "nested1": 1},
		"_id":   "det-test",
		"arr":   bson.A{bson.M{"b": 2, "a": 1}, "scalar"},
	}

	if err := WriteDocument(cloneDir, "data", "det", doc); err != nil {
		t.Fatalf("first write: %v", err)
	}
	first, err := os.ReadFile(filepath.Join(cloneDir, "data", "det.json"))
	if err != nil {
		t.Fatalf("read first: %v", err)
	}

	if err := WriteDocument(cloneDir, "data", "det", doc); err != nil {
		t.Fatalf("second write: %v", err)
	}
	second, err := os.ReadFile(filepath.Join(cloneDir, "data", "det.json"))
	if err != nil {
		t.Fatalf("read second: %v", err)
	}

	if string(first) != string(second) {
		t.Errorf("non‑deterministic output:\n--- first ---\n%s\n--- second ---\n%s", first, second)
	}

	// Verify keys appear in alphabetical order at the top level.
	content := string(first)
	idIdx := strings.Index(content, "\"_id\"")
	alphaIdx := strings.Index(content, "\"alpha\"")
	arrIdx := strings.Index(content, "\"arr\"")
	mikeIdx := strings.Index(content, "\"mike\"")
	zebraIdx := strings.Index(content, "\"zebra\"")
	if idIdx >= alphaIdx || alphaIdx >= arrIdx || arrIdx >= mikeIdx || mikeIdx >= zebraIdx {
		t.Errorf("keys not in alphabetical order:\n%s", content)
	}
}

func TestWriteDocument_UnwritablePath(t *testing.T) {
	parent := t.TempDir()
	_ = os.Chmod(parent, 0o000)
	t.Cleanup(func() { _ = os.Chmod(parent, 0o755) })

	doc := bson.M{"_id": "doc-a", "value": 1}
	err := WriteDocument(parent, "data", "doc-a", doc)
	if err == nil {
		t.Fatal("expected error")
	}
}
