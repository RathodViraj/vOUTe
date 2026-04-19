package user

import (
	"context"
	"errors"
	"time"
	"voute/pkg/config"
	"voute/pkg/utils"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type UserRepository interface {
	CreateUser(ctx context.Context, user *User) error
	GetUserByID(ctx context.Context, id string) (*User, error)
	GetUsersByUsername(ctx context.Context, username string, skip, take int) ([]*User, error)
	UpdateUser(ctx context.Context, username, email, id string) error
	DeleteUser(ctx context.Context, id string) error
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	UpdatePassword(ctx context.Context, email, hashPwd string) error
}

type userRepo struct {
	mongoDB     *mongo.Database
	collenction string
}

func NewUserRepository(mongoDB *mongo.Database) UserRepository {
	return &userRepo{
		mongoDB:     mongoDB,
		collenction: config.GetEnvWithDefault("MONGO_USER_COLLECTION", "users"),
	}
}

func (r *userRepo) CreateUser(ctx context.Context, user *User) error {
	res, err := r.mongoDB.Collection(r.collenction).InsertOne(ctx, user)
	if err != nil {
		return err
	}

	id, ok := res.InsertedID.(int64)
	if !ok {
		return errors.New("failed to convert inserted ID to int64")
	}

	user.ID = id
	return nil
}

func (r *userRepo) GetUserByID(ctx context.Context, id string) (*User, error) {
	parsedID, err := utils.ParseSnowflakeID(id)
	if err != nil {
		return nil, err
	}

	var u User
	filter := bson.M{
		"_id":        parsedID,
		"is_deleted": false,
	}
	err = r.mongoDB.Collection(r.collenction).FindOne(ctx, filter).Decode(&u)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}

	return &u, nil
}

func (r *userRepo) GetUsersByUsername(ctx context.Context, username string, skip, take int) ([]*User, error) {
	users := []*User{}
	filter := bson.M{
		"username":   username,
		"is_deleted": false,
	}

	findOptions := options.Find()
	findOptions.SetSkip(int64(skip))
	findOptions.SetLimit(int64(take))
	cur, err := r.mongoDB.Collection(r.collenction).Find(ctx, filter, findOptions)
	if err != nil {
		return users, err
	}
	defer cur.Close(ctx)

	for cur.Next(ctx) {
		var u User
		if err := cur.Decode(&u); err != nil {
			return users, err
		}
		users = append(users, &u)
	}

	return users, err
}

func (r *userRepo) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	var u User
	filter := bson.M{
		"email":      email,
		"is_deleted": false,
	}
	err := r.mongoDB.Collection(r.collenction).FindOne(ctx, filter).Decode(&u)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errors.New("user does not exist")
		}
		return nil, err
	}

	return &u, nil
}

func (r *userRepo) UpdateUser(ctx context.Context, username, email, id string) error {
	parsedID, err := utils.ParseSnowflakeID(id)
	if err != nil {
		return err
	}

	filter := bson.M{
		"_id":        parsedID,
		"is_deleted": false,
	}
	update := bson.M{
		"$set": bson.M{
			"username": username,
			"email":    email,
		},
	}
	_, err = r.mongoDB.Collection(r.collenction).UpdateOne(ctx, filter, update)
	return err
}

func (r *userRepo) UpdatePassword(ctx context.Context, email, hashPwd string) error {
	filter := bson.M{
		"email":      email,
		"is_deleted": false,
	}
	update := bson.M{
		"$set": bson.M{
			"password": hashPwd,
		},
	}
	_, err := r.mongoDB.Collection(r.collenction).UpdateOne(ctx, filter, update)
	return err
}

func (r *userRepo) DeleteUser(ctx context.Context, id string) error {
	parsedID, err := utils.ParseSnowflakeID(id)
	if err != nil {
		return err
	}

	filter := bson.M{
		"_id":        parsedID,
		"is_deleted": false,
	}
	now := time.Now()
	update := bson.M{
		"$set": bson.M{
			"is_deleted": true,
			"deleted_at": &now,
		},
	}
	_, err = r.mongoDB.Collection(r.collenction).UpdateOne(ctx, filter, update)
	return err
}
