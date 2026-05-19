package mongo

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// MockFetcher is a Fetcher that returns pre-configured data, for use in tests.
type MockFetcher struct {
	Docs []bson.M
	Err  error
}

// ForEach returns the configured error or iterates the configured documents.
func (m *MockFetcher) ForEach(_ context.Context, fn func(bson.M) error) error {
	if m.Err != nil {
		return m.Err
	}
	for _, doc := range m.Docs {
		if err := fn(doc); err != nil {
			return err
		}
	}
	return nil
}
