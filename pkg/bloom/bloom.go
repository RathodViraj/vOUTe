// Bloom filter for fast duplicate username detection.
//
// Design decisions:
//   - In-memory bit array for O(k) reads with zero network latency
//   - Redis persistence so the filter survives process restarts
//   - MongoDB seeding on cold start (no Redis snapshot yet)
//   - Double hashing (Kirsch-Mitzenmacher) for k hash functions from 2 SHA-256 halves
//   - Sized for 1 000 000 items at 1% false-positive rate → ~1.2 MB, 7 hashes

package bloom

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	expectedItems       = 1000000
	targetFPR           = 0.01
	redisKey            = "bloom:usernames"
	asyncPersistTimeout = 5 * time.Second
)

type Filter struct {
	bits  []uint64
	m     uint64
	k     uint64
	mu    sync.RWMutex
	count atomic.Int64
	rdb   *redis.Client
}

func NewBloomFilter(rdb *redis.Client) *Filter {
	m := optimalM(expectedItems, targetFPR)
	k := optimalK(m, expectedItems)

	return &Filter{
		bits: make([]uint64, wordsNeeded(m)),
		m:    m,
		k:    k,
		rdb:  rdb,
	}
}

func (f *Filter) Add(username string) {
	f.mu.Lock()
	for _, pos := range f.positions(username) {
		f.setBit(pos)
	}
	f.mu.Unlock()

	f.count.Add(1)

	if f.rdb != nil {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), asyncPersistTimeout)
			defer cancel()

			_ = f.PersistToRedis(ctx)
		}()
	}
}

func (f *Filter) MightExist(username string) bool {
	f.mu.RLock()
	defer f.mu.RUnlock()

	for _, pos := range f.positions(username) {
		if !f.getBit(pos) {
			return false
		}
	}

	return true
}

func (f *Filter) Count() int64 { return f.count.Load() }

func (f *Filter) PersistToRedis(ctx context.Context) error {
	if f.rdb == nil {
		return nil
	}

	f.mu.RLock()
	data, err := json.Marshal(f.bits)
	f.mu.RUnlock()
	if err != nil {
		return fmt.Errorf("bloom: marshal: %w", err)
	}

	if err := f.rdb.Set(ctx, redisKey, data, 0).Err(); err != nil {
		return fmt.Errorf("bloom: redis SET: %w", err)
	}
	return nil
}

func (f *Filter) LoadFromRedis(ctx context.Context) (bool, error) {
	if f.rdb == nil {
		return false, nil
	}

	data, err := f.rdb.Get(ctx, redisKey).Bytes()
	if err == redis.Nil {
		return false, nil // cold start — nothing persisted yet
	}
	if err != nil {
		return false, fmt.Errorf("bloom: redis GET: %w", err)
	}

	var bits []uint64
	if err := json.Unmarshal(data, &bits); err != nil {
		return false, fmt.Errorf("bloom: unmarshal: %w", err)
	}
	if uint64(len(bits)) != wordsNeeded(f.m) { // Snapshot was built with different parameters — ignore it.
		return false, nil
	}

	f.mu.Lock()
	f.bits = bits
	f.mu.Unlock()

	return true, nil
}

func (f *Filter) SeedFromMongoDB(ctx context.Context, col *mongo.Collection) error {
	projection := options.Find().SetProjection(
		bson.D{
			{Key: "username", Value: 1},
			{Key: "_id", Value: 0},
			{Key: "email", Value: 0},
		},
	)

	cur, err := col.Find(ctx, bson.D{}, projection)
	if err != nil {
		return fmt.Errorf("bloom: mongo find: %w", err)
	}
	defer cur.Close(ctx)

	type userDoc struct {
		Username string `bson:"username"`
	}

	for cur.Next(ctx) {
		var doc userDoc
		if err := cur.Decode(&doc); err != nil {
			continue
		}
		if doc.Username != "" {
			f.Add(doc.Username)
		}
	}

	return cur.Err()
}

// Hashing — Kirsch-Mitzenmacher double hashing
// position_i = (h1 + i * h2) mod m
// Two SHA-256 halves give us h1 and h2. This avoids running k independent
// hash functions while maintaining the same asymptotic false-positive rate.
func (f *Filter) positions(item string) []uint64 {
	h := sha256.Sum256([]byte(item))
	h1 := binary.BigEndian.Uint64(h[0:8])
	h2 := binary.BigEndian.Uint64(h[8:16])

	h2 |= 1

	pos := make([]uint64, f.k)
	for i := uint64(0); i < f.k; i++ {
		pos[i] = (h1 + i*h2) % f.m
	}
	return pos
}

func (f *Filter) setBit(pos uint64) {
	f.bits[pos/64] |= 1 << (pos % 64)
}

func (f *Filter) getBit(pos uint64) bool {
	return f.bits[pos/64]&(1<<(pos%64)) != 0
}

// optimalM returns the number of bits for n expected items at false-positive rate p.
// m = -n * ln(p) / ln(2)²
func optimalM(n uint64, p float64) uint64 {
	return uint64(math.Ceil(-float64(n) * math.Log(p) / (math.Log(2) * math.Log(2))))
}

// optimalK returns the optimal number of hash functions.
// k = (m/n) * ln(2)
func optimalK(m, n uint64) uint64 {
	return uint64(math.Round(float64(m) / float64(n) * math.Log(2)))
}

// wordsNeeded returns how many uint64 words hold m bits.
func wordsNeeded(m uint64) uint64 {
	return (m + 63) / 64
}
