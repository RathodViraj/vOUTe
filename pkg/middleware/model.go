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

func (d *MiddleWareDB) FetchUserByUsername(ctx context.Context, username string) (int64, string, string, error) {
	filter := bson.M{
		"username":   username,
		"is_deleted": false,
	}

	projection := bson.D{
		{Key: "username", Value: 1},
		{Key: "role", Value: 1},
		{Key: "password", Value: 1},
		{Key: "_id", Value: 1},
	}

	opts := options.FindOne().SetProjection(projection)
	result := d.mongoDatabase.Collection(d.userCollectioName).FindOne(ctx, filter, opts)
	if result.Err() != nil {
		if errors.Is(result.Err(), mongo.ErrNoDocuments) {
			return 0, "", "", errors.New("invalid credntials")
		}
		return 0, "", "", result.Err()
	}

	u := struct {
		ID       int64  `bson:"_id"`
		Username string `bson:"username"`
		Role     string `bson:"role"`
		Password string `bson:"password"`
	}{}

	if err := result.Decode(&u); err != nil {
		return 0, "", "", err
	}

	return u.ID, u.Password, u.Role, nil
}

func (d *MiddleWareDB) FetchUserByEmail(ctx context.Context, email string) (int64, string, string, string, error) {
	filter := bson.M{
		"email":      email,
		"is_deleted": false,
	}

	projection := bson.D{
		{Key: "username", Value: 1},
		{Key: "role", Value: 1},
		{Key: "password", Value: 1},
		{Key: "_id", Value: 1},
	}

	opts := options.FindOne().SetProjection(projection)
	result := d.mongoDatabase.Collection(d.userCollectioName).FindOne(ctx, filter, opts)
	if result.Err() != nil {
		if errors.Is(result.Err(), mongo.ErrNoDocuments) {
			return 0, "", "", "", errors.New("invalid credntials")
		}
		return 0, "", "", "", result.Err()
	}

	u := struct {
		ID       int64  `bson:"_id"`
		Username string `bson:"username"`
		Role     string `bson:"role"`
		Password string `bson:"password"`
	}{}

	if err := result.Decode(&u); err != nil {
		return 0, "", "", "", err
	}

	return u.ID, u.Username, u.Password, u.Role, nil
}

func (d *MiddleWareDB) ResetPassword(ctx context.Context, email, newPassword string) error {
	filter := bson.M{
		"email":      email,
		"is_deleted": false,
	}

	update := bson.M{
		"$set": bson.M{
			"password": newPassword,
		},
	}

	opts := options.Update().SetUpsert(true)
	_, err := d.mongoDatabase.Collection(d.userCollectioName).UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return err
	}

	return nil
}
