package middleware

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MiddleWareDB struct {
	mongoDatabase     *mongo.Database
	userCollectioName string
}

func (d *MiddleWareDB) FetchUserByUsername(ctx context.Context, username string) (string, string, error) {
	filter := bson.M{
		"username":   username,
		"is_deleted": false,
	}

	projection := bson.D{
		{Key: "username", Value: 1},
		{Key: "role", Value: 1},
		{Key: "password", Value: 1},
		{Key: "_id", Value: 0},
		{Key: "id_deleted", Value: 0},
		{Key: "email", Value: 0},
		{Key: "created_at", Value: 0},
		{Key: "deleted_at", Value: 0},
	}

	opts := options.FindOne().SetProjection(projection)
	result := d.mongoDatabase.Collection(d.userCollectioName).FindOne(ctx, filter, opts)
	if result.Err() != nil {
		if errors.Is(result.Err(), mongo.ErrNoDocuments) {
			return "", "", errors.New("invalid credntials")
		}
		return "", "", result.Err()
	}

	u := struct {
		Username string `bson:"username"`
		Role     string `bson:"role"`
		Password string `bson:"password"`
	}{}

	if err := result.Decode(&u); err != nil {
		return "", "", err
	}

	return u.Password, u.Role, nil
}

func (d *MiddleWareDB) FetchUserByEmail(ctx context.Context, email string) (string, string, string, error) {
	filter := bson.M{
		"email":      email,
		"is_deleted": false,
	}

	projection := bson.D{
		{Key: "username", Value: 1},
		{Key: "role", Value: 1},
		{Key: "password", Value: 1},
		{Key: "_id", Value: 0},
		{Key: "id_deleted", Value: 0},
		{Key: "email", Value: 0},
		{Key: "created_at", Value: 0},
		{Key: "deleted_at", Value: 0},
	}

	opts := options.FindOne().SetProjection(projection)
	result := d.mongoDatabase.Collection(d.userCollectioName).FindOne(ctx, filter, opts)
	if result.Err() != nil {
		if errors.Is(result.Err(), mongo.ErrNoDocuments) {
			return "", "", "", errors.New("invalid credntials")
		}
		return "", "", "", result.Err()
	}

	u := struct {
		Username string `bson:"username"`
		Role     string `bson:"role"`
		Password string `bson:"password"`
	}{}

	if err := result.Decode(&u); err != nil {
		return "", "", "", err
	}

	return u.Username, u.Password, u.Role, nil
}
