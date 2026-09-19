package permission

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// MongoDBStore returns a Store backed by a MongoDB collection.
// dsn is a mongodb:// or mongodb+srv:// URI.
// database defaults to "gocraft", collection defaults to "permissions".
func MongoDBStore(dsn, database, collection string) (Store, error) {
	if database == "" {
		database = "gocraft"
	}
	if collection == "" {
		collection = "permissions"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(options.Client().ApplyURI(dsn))
	if err != nil {
		return nil, fmt.Errorf("mongodb connect: %w", err)
	}
	if err := client.Ping(ctx, nil); err != nil {
		_ = client.Disconnect(context.Background())
		return nil, fmt.Errorf("mongodb ping: %w", err)
	}
	coll := client.Database(database).Collection(collection)
	return &mongoStore{client: client, coll: coll}, nil
}

type mongoStore struct {
	client *mongo.Client
	coll   *mongo.Collection
}

func (s *mongoStore) Load() (Document, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var result struct {
		Data json.RawMessage `bson:"data"`
	}
	err := s.coll.FindOne(ctx, bson.M{"_id": "singleton"}).Decode(&result)
	if err == mongo.ErrNoDocuments {
		doc := DefaultDocument()
		return doc, s.Save(doc)
	}
	if err != nil {
		return Document{}, fmt.Errorf("mongodb load: %w", err)
	}
	var doc Document
	if err := json.Unmarshal(result.Data, &doc); err != nil {
		return Document{}, fmt.Errorf("mongodb decode: %w", err)
	}
	return doc, nil
}

func (s *mongoStore) Save(doc Document) error {
	data, err := json.Marshal(doc)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = s.coll.UpdateOne(ctx,
		bson.M{"_id": "singleton"},
		bson.M{"$set": bson.M{"data": bson.Raw(data)}},
		options.UpdateOne().SetUpsert(true),
	)
	return err
}

func (s *mongoStore) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s.client.Disconnect(ctx)
}
