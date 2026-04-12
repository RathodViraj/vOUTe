package bloom

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
)

type InitOption struct {
	SeedTimeout time.Duration
}

func defualtOptions() InitOption {
	return InitOption{
		SeedTimeout: 30 * time.Second,
	}
}

// Strategy:
//  1. Try to restore the bit array from Redis — O(1) network call.
//  2. On a cold start (nothing in Redis), stream all usernames from MongoDB,
//     build the filter, then persist the snapshot back to Redis.
func InitBloomFilter(ctx context.Context, rdb *redis.Client, col *mongo.Collection, opts ...InitOption) (*Filter, error) {
	options := defualtOptions()
	if len(opts) > 0 {
		options = opts[0]
	}

	f := NewBloomFilter(rdb)

	loaded, err := f.LoadFromRedis(ctx)
	if err != nil {
		log.Printf("bloom: load from redis: %w", err)
	}

	if loaded {
		log.Printf("[bloom] restored from Redis (%d bits, k=%d)", f.m, f.k)
		return f, nil
	}

	log.Printf("[bloom] cold start — seeding from MongoDB (this may take a moment)…")

	seedCtx, cancel := context.WithTimeout(ctx, options.SeedTimeout)
	defer cancel()

	start := time.Now()
	if err := f.SeedFromMongoDB(seedCtx, col); err != nil {
		return nil, fmt.Errorf("bloom: seed from MongoDB: %w", err)
	}
	log.Printf("[bloom] seeded %d usernames in %s", f.Count(), time.Since(start).Round(time.Millisecond))

	if err := f.PersistToRedis(ctx); err != nil {
		log.Printf("[bloom] warning: could not persist to Redis: %v", err)
	}

	return f, nil
}
