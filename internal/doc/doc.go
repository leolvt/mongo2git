package doc

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// WriteDocument serializes a single document as MongoDB Extended JSON and
// writes it to cloneDir/dumpDir/<docID>.json.
func WriteDocument(cloneDir, dumpDir, docID string, doc bson.M) error {
	dir := filepath.Join(cloneDir, dumpDir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("mkdir %s: %w", dir, err)
	}

	extJSON, err := bson.MarshalExtJSONIndent(SortDocument(doc), false, false, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal document %s: %w", docID, err)
	}
	extJSON = append(extJSON, '\n')

	outPath := filepath.Join(dir, docID+".json")
	if err := os.WriteFile(outPath, extJSON, 0644); err != nil {
		return fmt.Errorf("write %s: %w", outPath, err)
	}
	slog.Info("wrote document", "path", outPath)
	return nil
}
