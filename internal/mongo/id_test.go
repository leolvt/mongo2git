package mongo

import (
	"strings"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestIDToFilename(t *testing.T) {
	objID := bson.NewObjectID()

	tests := []struct {
		name     string
		doc      bson.M
		want     string
		wantErr  bool
		errMatch string
	}{
		{
			name:    "ObjectID",
			doc:     bson.M{"_id": objID},
			want:    objID.Hex(),
			wantErr: false,
		},
		{
			name: "UUID binary (subtype 4, 16 bytes)",
			doc: bson.M{"_id": bson.Binary{
				Subtype: 4,
				Data:    []byte{0x00, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99, 0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff},
			}},
			want:    "00112233-4455-6677-8899-aabbccddeeff",
			wantErr: false,
		},
		{
			name: "non-UUID binary (subtype 0)",
			doc:  bson.M{"_id": bson.Binary{Subtype: 0, Data: []byte{0xde, 0xad, 0xbe, 0xef}}},
			want: "binary-deadbeef",
		},
		{
			name: "string _id",
			doc:  bson.M{"_id": "my-custom-id"},
			want: "my-custom-id",
		},
		{
			name: "string _id with path traversal (../)",
			doc:  bson.M{"_id": "../../../etc/passwd"},
			want: ".._.._.._etc_passwd",
		},
		{
			name: "string _id with backslash",
			doc:  bson.M{"_id": "foo\\bar"},
			want: "foo_bar",
		},
		{
			name: "integer _id (fallback)",
			doc:  bson.M{"_id": int32(42)},
			want: "42",
		},
		{
			name:     "missing _id",
			doc:      bson.M{"name": "no-id"},
			wantErr:  true,
			errMatch: "no _id field",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := IDToFilename(tt.doc)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if !strings.Contains(err.Error(), tt.errMatch) {
					t.Fatalf("expected error containing %q, got %q", tt.errMatch, err.Error())
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

func TestFormatUUID(t *testing.T) {
	tests := []struct {
		name string
		b    []byte
		want string
	}{
		{
			name: "standard UUID",
			b:    []byte{0x55, 0x0e, 0x84, 0x00, 0xe2, 0x9b, 0x41, 0xd4, 0xa7, 0x16, 0x44, 0x66, 0x55, 0x44, 0x00, 0x00},
			want: "550e8400-e29b-41d4-a716-446655440000",
		},
		{
			name: "all zeros",
			b:    []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			want: "00000000-0000-0000-0000-000000000000",
		},
		{
			name: "short input fallback",
			b:    []byte{0xde, 0xad, 0xbe, 0xef},
			want: "deadbeef",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FormatUUID(tt.b); got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}
