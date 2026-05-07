package middleware

import (
	"context"
	"errors"
	"time"
	"voute/pkg/mailing"
	"voute/pkg/utils"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MiddleWareDB struct {
	mongoDatabase     *mongo.Database
	userCollectioName string
	emailService      mailing.EmailService
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

	newHashPass, err := utils.HashPassword(newPassword)
	if err != nil {
		return err
	}

	update := bson.M{
		"$set": bson.M{
			"password": newHashPass,
		},
	}

	opts := options.Update().SetUpsert(true)
	_, err = d.mongoDatabase.Collection(d.userCollectioName).UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return err
	}

	return nil
}

func (d *MiddleWareDB) CreateUser(ctx context.Context, username, email, password string) (int64, error) {
	// Check if username already exists
	filter := bson.M{
		"username":   username,
		"is_deleted": false,
	}
	result := d.mongoDatabase.Collection(d.userCollectioName).FindOne(ctx, filter)
	if result.Err() == nil {
		return 0, errors.New("username already exists")
	} else if !errors.Is(result.Err(), mongo.ErrNoDocuments) {
		return 0, result.Err()
	}

	// Check if email already exists
	emailFilter := bson.M{
		"email":      email,
		"is_deleted": false,
	}
	emailResult := d.mongoDatabase.Collection(d.userCollectioName).FindOne(ctx, emailFilter)
	if emailResult.Err() == nil {
		return 0, errors.New("email already exists")
	} else if !errors.Is(emailResult.Err(), mongo.ErrNoDocuments) {
		return 0, emailResult.Err()
	}

	hashPass, err := utils.HashPassword(password)
	if err != nil {
		return 0, err
	}

	// Generate user ID using snowflake
	userID := d.generateID()

	user := bson.M{
		"_id":        userID,
		"username":   username,
		"email":      email,
		"password":   hashPass,
		"role":       "user",
		"created_at": time.Now().Unix(),
		"is_deleted": false,
	}

	_, err = d.mongoDatabase.Collection(d.userCollectioName).InsertOne(ctx, user)
	if err != nil {
		return 0, err
	}

	if bloomFilter != nil {
		bloomFilter.Add(username)
	}

	return userID, nil
}

func (d *MiddleWareDB) GetEmailByUsername(ctx context.Context, username string) (string, error) {
	filter := bson.M{
		"username":   username,
		"is_deleted": false,
	}

	projection := bson.D{
		{Key: "email", Value: 1},
	}
	opts := options.FindOne().SetProjection(projection)
	result := d.mongoDatabase.Collection(d.userCollectioName).FindOne(ctx, filter, opts)
	if result.Err() != nil {
		if errors.Is(result.Err(), mongo.ErrNoDocuments) {
			return "", errors.New("user not found")
		}
		return "", result.Err()
	}

	var user bson.M
	if err := result.Decode(&user); err != nil {
		return "", err
	}

	email, ok := user["email"].(string)
	if !ok {
		return "", errors.New("email not found")
	}

	return email, nil
}

func (d *MiddleWareDB) generateID() int64 {
	// Using current time in milliseconds as simple ID (in production, use proper snowflake ID generator)
	return primitive.NewObjectID().Timestamp().Unix() * 1000
}
