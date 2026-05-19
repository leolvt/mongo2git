package mongo

import (
	"fmt"
	"log/slog"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// IDToFilename extracts a filesystem-safe filename from a document's _id.
func IDToFilename(doc bson.M) (string, error) {
	id, ok := doc["_id"]
	if !ok {
		return "", fmt.Errorf("no _id field")
	}
	switch v := id.(type) {
	case bson.ObjectID:
		return v.Hex(), nil
	case bson.Binary:
		if v.Subtype == 4 && len(v.Data) == 16 {
			return FormatUUID(v.Data), nil
		}
		return fmt.Sprintf("binary-%x", v.Data), nil
	case string:
		return sanitizeFilename(v), nil
	default:
		slog.Warn("unexpected _id type, using string representation", "type", fmt.Sprintf("%T", id))
		return fmt.Sprintf("%v", id), nil
	}
}

// sanitizeFilename replaces path separators in a string _id to prevent
// directory traversal when used as a filename.
func sanitizeFilename(s string) string {
	s = strings.ReplaceAll(s, "/", "_")
	s = strings.ReplaceAll(s, "\\", "_")
	if s == "" {
		return "unnamed"
	}
	return s
}

// FormatUUID formats 16 bytes as a hex UUID string (8-4-4-4-12).
// Falls back to plain hex if fewer than 16 bytes are provided.
func FormatUUID(b []byte) string {
	if len(b) < 16 {
		return fmt.Sprintf("%x", b)
	}
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
