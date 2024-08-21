package db

import (
	"context"
	"fmt"

	"github.com/spf13/viper"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (d *DB) ensureIndexes() error {
	collection := d.Client.Database(viper.GetString("database.name")).Collection("subjects")

	indexes := []mongo.IndexModel{
		{
			Keys: bson.D{{Key: "slug", Value: 1}},
			Options: options.Index().
				SetUnique(true).
				SetName("name_unique"),
		},
		{
			Keys:    bson.D{{Key: "namespace", Value: 1}},
			Options: options.Index().SetName("namespace_index"),
		},
	}

	_, err := collection.Indexes().CreateMany(context.Background(), indexes)
	if err != nil {
		return fmt.Errorf("failed to create indexes: %w", err)
	}

	return nil
}
