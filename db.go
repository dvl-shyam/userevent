package main

import (
	"context"
	"fmt"
	"os"
	"time"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var client *mongo.Client

func ConnectDB() (*mongo.Client, error) {
	var err error
	if client == nil {
		uri := os.Getenv("MONGO_URI")
		if uri == "" {
			return nil, fmt.Errorf("missing required MongoDB environment variables")
	}

		clientOptions := options.Client().ApplyURI(uri)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		client, err = mongo.Connect(ctx, clientOptions)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
		}

		if err = client.Ping(ctx, nil); err != nil {
			return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
		}
		fmt.Println("Connected to MongoDB!")
	}
	return client, nil
}
