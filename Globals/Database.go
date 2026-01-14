package Globals

import (
	"context"
	"fmt"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var MongoClient *mongo.Client
var Database *mongo.Database

func InitMongoDB() error {

	MongoURI := os.Getenv("MONGO_URI")

	if MongoURI == "" {

		return fmt.Errorf("MONGO_URI environment variable is not set")

	}

	Context, Cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer Cancel()

	Client, Error := mongo.Connect(Context, options.Client().ApplyURI(MongoURI))

	if Error != nil {

		return fmt.Errorf("failed to connect to MongoDB: %w", Error)

	}

	PingError := Client.Ping(Context, nil)

	if PingError != nil {

		return fmt.Errorf("failed to ping MongoDB: %w", PingError)

	}

	MongoClient = Client
	Database = Client.Database("Synthara-Redux")

	return nil

}