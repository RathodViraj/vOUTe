package mailing

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MailingRepository interface {
	StoreOTP(ctx context.Context, email, otp string) error
	GetOTP(ctx context.Context, email string) (*StoredOTP, error)
	VerifyOTP(ctx context.Context, email, otp string) (bool, error)
	IsUserExist(ctx context.Context, email string) (bool, error)
	IsUserExistByUsername(ctx context.Context, username string) (bool, error)
	GetEmailByUsername(ctx context.Context, username string) (string, error)
	StoreVerificationToken(ctx context.Context, token, email string, ttl time.Duration) error
	GetEmailByVerificationToken(ctx context.Context, token string) (string, error)
	DeleteVerificationToken(ctx context.Context, token string) error
}

type mailingRepository struct {
	rdb                *redis.Client
	mongoDB            *mongo.Database
	UserCollectionName string
}

func NewMailingRepository(rdb *redis.Client, mongoDB *mongo.Database, userCollectionName string) MailingRepository {
	return &mailingRepository{
		rdb:                rdb,
		mongoDB:            mongoDB,
		UserCollectionName: userCollectionName,
	}
}

func (r *mailingRepository) StoreOTP(ctx context.Context, email, otp string) error {
	return r.rdb.Set(ctx, "otp:"+email, otp, expiredTimeOUT).Err()
}

func (r *mailingRepository) GetOTP(ctx context.Context, email string) (*StoredOTP, error) {
	otp, err := r.rdb.Get(ctx, "otp:"+email).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &StoredOTP{Email: email, OTP: otp}, nil
}

func (r *mailingRepository) VerifyOTP(ctx context.Context, email, otp string) (bool, error) {
	storedOTP, err := r.GetOTP(ctx, email)
	if err != nil {
		return false, err
	}
	if storedOTP == nil {
		return false, nil
	}

	return storedOTP.OTP == otp, nil
}

func (r *mailingRepository) IsUserExist(ctx context.Context, email string) (bool, error) {
	filter := bson.M{
		"email":      email,
		"is_deleted": false,
	}

	projection := bson.D{
		{Key: "_id", Value: 1},
	}
	opts := options.FindOne().SetProjection(projection)
	result := r.mongoDB.Collection(r.UserCollectionName).FindOne(ctx, filter, opts)
	if result.Err() != nil {
		if errors.Is(result.Err(), mongo.ErrNoDocuments) {
			return false, nil
		}
		return false, result.Err()
	}

	return true, nil
}

func (r *mailingRepository) IsUserExistByUsername(ctx context.Context, username string) (bool, error) {
	filter := bson.M{
		"username":   username,
		"is_deleted": false,
	}

	projection := bson.D{
		{Key: "_id", Value: 1},
	}
	opts := options.FindOne().SetProjection(projection)
	result := r.mongoDB.Collection(r.UserCollectionName).FindOne(ctx, filter, opts)
	if result.Err() != nil {
		if errors.Is(result.Err(), mongo.ErrNoDocuments) {
			return false, nil
		}
		return false, result.Err()
	}

	return true, nil
}

func (r *mailingRepository) GetEmailByUsername(ctx context.Context, username string) (string, error) {
	filter := bson.M{
		"username":   username,
		"is_deleted": false,
	}

	projection := bson.D{
		{Key: "email", Value: 1},
	}
	opts := options.FindOne().SetProjection(projection)
	result := r.mongoDB.Collection(r.UserCollectionName).FindOne(ctx, filter, opts)
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

func (r *mailingRepository) StoreVerificationToken(ctx context.Context, token, email string, ttl time.Duration) error {
	return r.rdb.Set(ctx, "otp_verified:"+token, email, ttl).Err()
}

func (r *mailingRepository) GetEmailByVerificationToken(ctx context.Context, token string) (string, error) {
	val, err := r.rdb.Get(ctx, "otp_verified:"+token).Result()
	if err == redis.Nil {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return val, nil
}

func (r *mailingRepository) DeleteVerificationToken(ctx context.Context, token string) error {
	return r.rdb.Del(ctx, "otp_verified:"+token).Err()
}
