package db

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/ibnuadam/dams-wallet-backend/config"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	client *mongo.Client
	once   sync.Once
)

func Connect() *mongo.Client {
	once.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		c, err := mongo.Connect(ctx, options.Client().ApplyURI(config.App.MongoURI))
		if err != nil {
			log.Fatalf("MongoDB connect error: %v", err)
		}

		if err = c.Ping(ctx, nil); err != nil {
			log.Fatalf("MongoDB ping error: %v", err)
		}

		log.Println("Connected to MongoDB")
		client = c
	})
	return client
}

func GetDB() *mongo.Database {
	return Connect().Database(config.App.DBName)
}

func Col(name string) *mongo.Collection {
	return GetDB().Collection(name)
}
