package db

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/viper"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

type DB struct {
	Client *mongo.Client
}

func New() (*DB, error) {
	return newFromURL(viper.GetString("dbaas.mongodb.endpoint"))
}

func newFromURL(url string) (*DB, error) {
	serverAPI := options.ServerAPI(options.ServerAPIVersion1)
	maxPoolSize := viper.GetUint64("database.maxPoolSize")

	opts := options.Client().ApplyURI(url).
		SetServerAPIOptions(serverAPI).
		SetMaxPoolSize(maxPoolSize).
		SetTimeout(time.Second * 30).
		SetReadPreference(readpref.PrimaryPreferred())

	client, err := mongo.Connect(context.Background(), opts)
	if err != nil {
		return nil, fmt.Errorf("failed to connect into database: %w", err)
	}
	d := &DB{Client: client}

	if err := d.ensureIndexes(); err != nil {
		return nil, fmt.Errorf("failed to ensure indexes: %w", err)
	}

	return d, nil
}
