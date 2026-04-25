package mailing

import (
	"context"
	"errors"

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
