package mongo

import (
	"context"
	"fmt"
	"log/slog"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
)

// Fetcher retrieves documents from MongoDB.
type Fetcher interface {
	ForEach(ctx context.Context, fn func(doc bson.M) error) error
}

// MongoFetcher implements Fetcher against a real MongoDB collection.
type MongoFetcher struct {
	client *mongo.Client
	coll   *mongo.Collection
}

// NewMongoFetcher connects to MongoDB, pings, and returns a fetcher plus a
// cleanup function. The caller must defer the cleanup.
func NewMongoFetcher(ctx context.Context, uri, collName, dbName string) (*MongoFetcher, func(), error) {
	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		return nil, nil, fmt.Errorf("connect: %w", err)
	}

	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		if dErr := client.Disconnect(context.Background()); dErr != nil {
			slog.Warn("failed to disconnect after ping failure", "error", dErr)
		}
		return nil, nil, fmt.Errorf("ping: %w", err)
	}

	db := client.Database(dbName)
	coll := db.Collection(collName)

	cleanup := func() {
		if err := client.Disconnect(context.Background()); err != nil {
			slog.Warn("failed to disconnect MongoDB client", "error", err)
		}
	}

	return &MongoFetcher{client: client, coll: coll}, cleanup, nil
}

// ForEach iterates all documents in the collection, calling fn for each.
// The driver handles internal batching; memory usage is O(1).
func (f *MongoFetcher) ForEach(ctx context.Context, fn func(bson.M) error) error {
	cursor, err := f.coll.Find(ctx, bson.D{})
	if err != nil {
		return fmt.Errorf("find: %w", err)
	}
	defer func() {
		if err := cursor.Close(context.Background()); err != nil {
			slog.Warn("failed to close MongoDB cursor", "error", err)
		}
	}()

	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			return fmt.Errorf("decode: %w", err)
		}
		if err := fn(doc); err != nil {
			return err
		}
	}
	return cursor.Err()
}
