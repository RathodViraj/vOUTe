package vote

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	maxVotes         = 24
	refillRatePerSec = 3600
	dirtyPollsKey    = "ws:dirty:polls"
)

var ErrVoteLimitReached = errors.New("vote limit reached")

var tokenBucketScript = redis.NewScript(`
	local key = KEYS[1]
	local now = tonumber(ARGV[1])
	local max_tokens = tonumber(ARGV[2])
	local rate_secs = tonumber(ARGV[3])
	local consume = tonumber(ARGV[4])

	if consume == nil or consume <= 0 or consume > max_tokens then
		return 0
	end

	local data = redis.call("HMGET", key, "remaining", "last_refill")
	local remaining = tonumber(data[1])
	local last_refill = tonumber(data[2])

	if remaining == nil then
		remaining = max_tokens
		last_refill = now
	end

	local elapsed = now - last_refill
	local tokens_to_add = math.floor(elapsed / rate_secs)

	if tokens_to_add > 0 then
		remaining = math.min(max_tokens, remaining + tokens_to_add)
		last_refill = last_refill + (tokens_to_add * rate_secs)
	end

	if remaining < consume then
		redis.call("HMSET", key, "remaining", remaining, "last_refill", last_refill)
		redis.call("EXPIRE", key, 172800)
		return 0
 	end

	remaining = remaining - consume
	redis.call("HMSET", key, "remaining", remaining, "last_refill", last_refill)
	redis.call("EXPIRE", key, 172800)
	return 1
`)

type VoteRepository interface {
	CreateVoteInMongo(ctx context.Context, vote *CreateVoteInMogo) error
	AddOptionsInMongo(ctx context.Context, voteID int64, options []Option) error
	GetVoteByID(ctx context.Context, id string) (*Vote, error)
	ListLiveVotePage(ctx context.Context, cursor string, take int) ([]*Vote, string, error)
	ListVotePage(ctx context.Context, cursor string, take int) ([]*Vote, string, error)
	GetVotesByCreatorIDPage(ctx context.Context, creatorID, cursor string, take int) ([]*Vote, string, error)
	InitVoteInRedis(ctx context.Context, voteID int64, optionIDs []int64) error
	AddVote(ctx context.Context, userID, voteID, optionID string, count int64) error
	GetUserVotedPolls(ctx context.Context, userID string) ([]UserVotedPoll, error)
	GetRemainingVotes(ctx context.Context, userID string) (int64, error)
	EditTitle(ctx context.Context, voteID, newTitle string) error
	CloseVoteInMongo(ctx context.Context, voteID string) error
	DeleteVoteInRedis(ctx context.Context, voteID string) error
	GetPollHistory(ctx context.Context, voteID string, rangeVal string) ([]HistoricOptionsData, error)
	UpdateStatus(ctx context.Context, id int64, newStatus string) error
	HardDeleteVote(ctx context.Context, voteID int64) error
	GetPollsFromIDs(ctx context.Context, votesIDs []string) ([]Vote, error)
	GetSnapshots(ctx context.Context, pollIDs []string) ([]PollSnapshot, error)
	PopDirtyPolls(ctx context.Context) ([]string, error)
}

type pageCursor struct {
	CreatedAt int64
	ID        int64
}

func parsePageCursor(cursor string) (*pageCursor, error) {
	if cursor == "" {
		return nil, nil
	}
	parts := strings.Split(cursor, ":")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid cursor")
	}
	createdAt, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return nil, err
	}
	id, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return nil, err
	}
	return &pageCursor{CreatedAt: createdAt, ID: id}, nil
}

func encodePageCursor(createdAt, id int64) string {
	return fmt.Sprintf("%d:%d", createdAt, id)
}

type voteRepo struct {
	mongoDB           *mongo.Database
	voteCollection    string
	optionsCollection string
	rdb               *redis.Client
	timescaleDB       *sql.DB
}

func NewVoteRepository(mongoDB *mongo.Database, voteCollection string, optionsCollection string, rdb *redis.Client, timescaleDB *sql.DB) VoteRepository {
	return &voteRepo{
		mongoDB:           mongoDB,
		voteCollection:    voteCollection,
		optionsCollection: optionsCollection,
		rdb:               rdb,
		timescaleDB:       timescaleDB,
	}
}

func voteKey(id int64) string {
	return "vote:" + strconv.FormatInt(id, 10)
}

func parseVoteID(id string) (int64, error) {
	return strconv.ParseInt(id, 10, 64)
}

func (r *voteRepo) CreateVoteInMongo(ctx context.Context, vote *CreateVoteInMogo) error {
	_, err := r.mongoDB.Collection(r.voteCollection).InsertOne(ctx, vote)
	return err
}

func (r *voteRepo) getOptionsByVoteID(ctx context.Context, voteID int64) ([]Option, error) {
	cur, err := r.mongoDB.Collection(r.optionsCollection).Find(ctx, bson.M{"vote_id": voteID})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	options := make([]Option, 0)
	for cur.Next(ctx) {
		var option Option
		if err := cur.Decode(&option); err != nil {
			return nil, err
		}
		options = append(options, option)
	}

	return options, nil
}

func (r *voteRepo) AddOptionsInMongo(ctx context.Context, voteID int64, options []Option) error {
	var optionDocs []any
	for _, option := range options {
		optionDocs = append(optionDocs, option)
	}

	_, err := r.mongoDB.Collection(r.optionsCollection).InsertMany(ctx, optionDocs)
	return err
}

func (r *voteRepo) InitVoteInRedis(ctx context.Context, voteID int64, optionIDs []int64) error {
	pipe := r.rdb.Pipeline()
	for _, optionID := range optionIDs {
		pipe.HSet(ctx, voteKey(voteID), optionID, 0)
	}

	_, err := pipe.Exec(ctx)
	return err
}

func (r *voteRepo) UpdateStatus(ctx context.Context, id int64, newStatus string) error {
	_, err := r.mongoDB.Collection(r.voteCollection).UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": bson.M{"status": newStatus}})
	return err
}

func (r *voteRepo) GetVoteByID(ctx context.Context, id string) (*Vote, error) {
	parsedID, err := parseVoteID(id)
	if err != nil {
		return nil, err
	}
	var vote Vote
	filter := bson.M{
		"_id":        parsedID,
		"is_deleted": false,
	}

	err = r.mongoDB.Collection(r.voteCollection).FindOne(ctx, filter).Decode(&vote)
	if err != nil {
		return nil, err
	}

	options, err := r.getOptionsByVoteID(ctx, vote.ID)
	if err != nil {
		return nil, err
	}
	vote.Options = options

	voteCount, err := r.getVoteCount(ctx, vote.ID)
	if err != nil {
		return nil, err
	}

	for i := range vote.Options {
		if cnt, ok := voteCount[vote.Options[i].ID]; ok {
			vote.Options[i].VoteCount = cnt
		} else {
			vote.Options[i].VoteCount = 0
		}
	}

	return &vote, nil
}

func (r *voteRepo) ListVotePage(ctx context.Context, cursor string, take int) ([]*Vote, string, error) {
	return r.listVotePage(ctx, bson.M{
		"is_deleted": false,
		"status":     bson.M{"$in": []string{"closed", "live"}},
	}, cursor, take)
}

func (r *voteRepo) ListLiveVotePage(ctx context.Context, cursor string, take int) ([]*Vote, string, error) {
	return r.listVotePage(ctx, bson.M{
		"is_deleted": false,
		"status":     "live",
	}, cursor, take)
}

func (r *voteRepo) GetVotesByCreatorIDPage(ctx context.Context, creatorID, cursor string, take int) ([]*Vote, string, error) {
	parsedCreatorID, err := parseVoteID(creatorID)
	if err != nil {
		return nil, "", err
	}
	return r.listVotePage(ctx, bson.M{
		"created_by_id": parsedCreatorID,
		"status":        bson.M{"$in": []string{"closed", "live"}},
		"is_deleted":    false,
	}, cursor, take)
}

func (r *voteRepo) listVotePage(ctx context.Context, filter bson.M, cursor string, take int) ([]*Vote, string, error) {
	if take <= 0 {
		take = 10
	}

	queryFilter := bson.M{}
	for k, v := range filter {
		queryFilter[k] = v
	}

	if parsedCursor, err := parsePageCursor(cursor); err == nil && parsedCursor != nil {
		queryFilter["$or"] = []bson.M{
			{"created_at": bson.M{"$lt": parsedCursor.CreatedAt}},
			{"created_at": parsedCursor.CreatedAt, "_id": bson.M{"$lt": parsedCursor.ID}},
		}
	}

	options := options.Find()
	options.SetLimit(int64(take + 1))
	options.SetSort(bson.D{{Key: "created_at", Value: -1}, {Key: "_id", Value: -1}})

	cursorRes, err := r.mongoDB.Collection(r.voteCollection).Find(ctx, queryFilter, options)
	if err != nil {
		return nil, "", err
	}
	defer cursorRes.Close(ctx)

	votes := make([]*Vote, 0, take+1)
	for cursorRes.Next(ctx) {
		var vote Vote
		if err := cursorRes.Decode(&vote); err != nil {
			continue
		}

		options, err := r.getOptionsByVoteID(ctx, vote.ID)
		if err != nil {
			return nil, "", err
		}
		vote.Options = options

		voteCount, err := r.getVoteCount(ctx, vote.ID)
		if err != nil {
			return nil, "", err
		}

		for i := range vote.Options {
			if cnt, ok := voteCount[vote.Options[i].ID]; ok {
				vote.Options[i].VoteCount = cnt
			} else {
				vote.Options[i].VoteCount = 0
			}
		}

		votes = append(votes, &vote)
	}

	nextCursor := ""
	if len(votes) > take {
		nextCursor = encodePageCursor(votes[take-1].CreatedAt, votes[take-1].ID)
		votes = votes[:take]
	}

	return votes, nextCursor, nil
}

func (r *voteRepo) getVoteCount(ctx context.Context, voteID int64) (map[int64]int64, error) {
	result, err := r.rdb.HGetAll(ctx, voteKey(voteID)).Result()
	if err != nil {
		return nil, err
	}
	voteCount := make(map[int64]int64)
	for optionID := range result {
		parsedOptionID, err := strconv.ParseInt(optionID, 10, 64)
		if err != nil {
			continue
		}
		count, err := r.rdb.HGet(ctx, voteKey(voteID), optionID).Int64()
		if err != nil {
			continue
		}
		voteCount[parsedOptionID] = count
	}
	return voteCount, nil
}

func (r *voteRepo) AddVote(ctx context.Context, userID, voteID, optionID string, count int64) error {
	if count <= 0 {
		return fmt.Errorf("count must be greater than zero")
	}

	parsedVoteID, err := parseVoteID(voteID)
	if err != nil {
		return err
	}
	parsedOptionID, err := parseVoteID(optionID)
	if err != nil {
		return err
	}
	parsedUserID, err := parseVoteID(userID)
	if err != nil {
		return err
	}

	allowed, err := tokenBucketScript.Run(
		ctx, r.rdb,
		[]string{userBucketKey(parsedUserID)},
		time.Now().Unix(), maxVotes, refillRatePerSec, count,
	).Int()
	if err != nil {
		return fmt.Errorf("rate limit check failed: %w", err)
	}
	if allowed == 0 {
		return ErrVoteLimitReached
	}

	metaKey := userVotedMetaKey(parsedUserID)
	var newTotalCount int64 = count
	if existingMeta, err := r.rdb.HGet(ctx, metaKey, voteID).Result(); err == nil {
		parts := strings.Split(existingMeta, "|")
		if len(parts) == 2 {
			if oldOpt, _ := strconv.ParseInt(parts[0], 10, 64); oldOpt == parsedOptionID {
				if oldCount, err2 := strconv.ParseInt(parts[1], 10, 64); err2 == nil {
					newTotalCount += oldCount
				}
			}
		}
	}

	now := time.Now()
	pipe := r.rdb.Pipeline()

	// Track which polls this user voted in (sorted set, score = timestamp)
	pipe.ZAdd(ctx, userVotedPollsKey(parsedUserID), redis.Z{
		Score:  float64(now.Unix()),
		Member: voteID,
	})
	// Keep at most 48 hours of voted-polls history in Redis.
	pipe.ZRemRangeByScore(ctx, userVotedPollsKey(parsedUserID),
		"-inf",
		strconv.FormatInt(now.Add(-48*time.Hour).Unix(), 10),
	)
	pipe.Expire(ctx, userVotedPollsKey(parsedUserID), 48*time.Hour)

	pipe.HSet(ctx, metaKey, voteID, fmt.Sprintf("%d|%d", parsedOptionID, newTotalCount))
	pipe.Expire(ctx, metaKey, 48*time.Hour)

	// Increment the actual vote count
	pipe.HIncrBy(ctx, voteKey(parsedVoteID), strconv.FormatInt(parsedOptionID, 10), count)
	// Mark this poll as dirty for the WS broadcaster
	pipe.SAdd(ctx, dirtyPollsKey, voteID)
	// Ensure poll is considered active for snapshot worker
	pipe.SAdd(ctx, activePollsKey(), voteID)

	_, err = pipe.Exec(ctx)
	if err != nil {
		return err
	}

	// Vote mutation is already committed in Redis at this point.
	// Snapshot persistence is now handled by a background worker; do not
	// block vote API on DB snapshot writes.
	return nil
}
func (r *voteRepo) saveSnapshot(ctx context.Context, voteID int64) error {
	voteCount, err := r.getVoteCount(ctx, voteID)
	if err != nil {
		return err
	}
	if len(voteCount) == 0 {
		return nil
	}

	tx, err := r.timescaleDB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	for optionID, count := range voteCount {
		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO vote_snapshots (vote_id, option_id, vote_count, created_at) VALUES ($1, $2, $3, NOW())`,
			voteID,
			optionID,
			count,
		); err != nil {
			_ = tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}

func userBucketKey(userID int64) string {
	return "user:vote:bucket:" + strconv.FormatInt(userID, 10)
}

func userVotedPollsKey(userID int64) string {
	return "user:voted:" + strconv.FormatInt(userID, 10)
}

func userVotedMetaKey(userID int64) string {
	return "user:voted:meta:" + strconv.FormatInt(userID, 10)
}

func activePollsKey() string {
	return "active_polls"
}

// GetPollHistory returns 1-hour bucketed aggregated snapshot data for a 24-hour slice before cursor.
// Results are cached in Redis for 5 minutes.
func (r *voteRepo) GetPollHistory(ctx context.Context, voteID string, cursor string) ([]HistoricOptionsData, error) {
	parsedVoteID, err := strconv.ParseInt(voteID, 10, 64)
	if err != nil {
		return nil, err
	}

	voteCollection := r.mongoDB.Collection(r.voteCollection)
	var pollDoc struct {
		CreatedAt int64 `bson:"created_at"`
	}
	if err := voteCollection.FindOne(ctx, bson.M{"_id": parsedVoteID}).Decode(&pollDoc); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	pollCreatedAt := time.Unix(pollDoc.CreatedAt, 0).UTC()
	maxHistoryStart := now.Add(-7 * 24 * time.Hour)

	parseCursor := func(raw string) time.Time {
		if raw == "" {
			return now
		}
		if parsed, err := time.Parse(time.RFC3339Nano, raw); err == nil {
			return parsed.UTC()
		}
		if parsed, err := time.Parse(time.RFC3339, raw); err == nil {
			return parsed.UTC()
		}
		if unix, err := strconv.ParseInt(raw, 10, 64); err == nil {
			return time.Unix(unix, 0).UTC()
		}
		return now
	}

	cursorTime := parseCursor(cursor)
	if cursorTime.After(now) {
		cursorTime = now
	}

	windowEnd := cursorTime.Truncate(time.Hour)
	if windowEnd.After(now.Truncate(time.Hour)) {
		windowEnd = now.Truncate(time.Hour)
	}
	if windowEnd.Before(maxHistoryStart) || windowEnd.Equal(maxHistoryStart) {
		return nil, nil
	}

	windowStart := windowEnd.Add(-24 * time.Hour)
	if windowStart.Before(maxHistoryStart) {
		windowStart = maxHistoryStart
	}

	if pollCreatedAt.After(windowEnd) {
		return nil, nil
	}

	ceilHour := func(t time.Time) time.Time {
		truncated := t.Truncate(time.Hour)
		if t.Equal(truncated) {
			return truncated
		}
		return truncated.Add(time.Hour)
	}
	if pollCreatedAt.After(windowStart) {
		windowStart = ceilHour(pollCreatedAt)
		if windowStart.Equal(windowEnd) {
			windowEnd = windowEnd.Add(time.Hour)
		}
	}

	if !windowStart.Before(windowEnd) {
		return nil, nil
	}

	cacheCursor := cursor
	if cacheCursor == "" {
		cacheCursor = "latest"
	}
	cacheKey := fmt.Sprintf("poll:%s:history:%s", voteID, cacheCursor)
	if cached, err := r.rdb.Get(ctx, cacheKey).Result(); err == nil && cached != "" {
		var out []HistoricOptionsData
		if err := json.Unmarshal([]byte(cached), &out); err == nil {
			return out, nil
		}
	}

	query := `SELECT time_bucket('1 hour', created_at) AS bucket, option_id, MAX(vote_count) as votes
	FROM vote_snapshots
	WHERE vote_id = $1
	AND created_at >= $2
	AND created_at < $3
	GROUP BY bucket, option_id
	ORDER BY bucket;`

	rows, err := r.timescaleDB.QueryContext(ctx, query, parsedVoteID, windowStart, windowEnd)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var resp []HistoricOptionsData
	for rows.Next() {
		var bucket time.Time
		var optionID int64
		var votes int64
		if err := rows.Scan(&bucket, &optionID, &votes); err != nil {
			continue
		}
		resp = append(resp, HistoricOptionsData{
			Timestamp: bucket,
			OptionID:  strconv.FormatInt(optionID, 10),
			VoteCount: int(votes),
		})
	}

	if len(resp) == 0 {
		return nil, nil
	}

	if data, marshalErr := json.Marshal(resp); marshalErr == nil {
		if setErr := r.rdb.Set(ctx, cacheKey, string(data), 5*time.Minute).Err(); setErr != nil {
			log.Printf("[GetPollHistory] WARNING: Failed to cache data for voteID %s: %v", voteID, setErr)
		}
	}

	return resp, nil
}

func (r *voteRepo) GetUserVotedPolls(ctx context.Context, userID string) ([]UserVotedPoll, error) {
	parsedUserId, err := parseVoteID(userID)
	if err != nil {
		return nil, err
	}

	cutoff := strconv.FormatInt(time.Now().Add(-24*time.Hour).Unix(), 10)
	pipe := r.rdb.Pipeline()
	idsCmd := pipe.ZRangeByScore(ctx, userVotedPollsKey(parsedUserId), &redis.ZRangeBy{Min: cutoff, Max: "+inf"})
	metaCmd := pipe.HGetAll(ctx, userVotedMetaKey(parsedUserId))

	if _, err := pipe.Exec(ctx); err != nil {
		return nil, err
	}

	ids := idsCmd.Val()
	meta := metaCmd.Val()
	result := make([]UserVotedPoll, 0, len(ids))
	for _, id := range ids {
		parsedID, err := parseVoteID(id)
		if err != nil {
			continue
		}

		entry := UserVotedPoll{VoteID: parsedID}
		if raw, ok := meta[id]; ok {
			parts := strings.Split(raw, "|")
			if len(parts) == 2 {
				if optionID, err := parseVoteID(parts[0]); err == nil {
					entry.OptionID = optionID
				}
				if voteCount, err := strconv.ParseInt(parts[1], 10, 64); err == nil {
					entry.VoteCount = voteCount
				}
			}
		}
		result = append(result, entry)
	}

	return result, nil
}

func (r *voteRepo) GetRemainingVotes(ctx context.Context, userID string) (int64, error) {
	parsedUserID, err := parseVoteID(userID)
	if err != nil {
		return 0, err
	}

	bucketKey := userBucketKey(parsedUserID)
	data, err := r.rdb.HGetAll(ctx, bucketKey).Result()
	if err != nil {
		return 0, err
	}
	if len(data) == 0 {
		return maxVotes, nil // never voted — full bucket
	}

	remaining, err := strconv.ParseInt(data["remaining"], 10, 64)
	if err != nil {
		remaining = maxVotes
	}
	lastRefill, err := strconv.ParseInt(data["last_refill"], 10, 64)
	if err != nil {
		lastRefill = time.Now().Unix()
	}

	elapsed := time.Now().Unix() - lastRefill
	if elapsed > 0 {
		tokensToAdd := elapsed / refillRatePerSec
		if tokensToAdd > 0 {
			remaining += tokensToAdd
			if remaining > maxVotes {
				remaining = maxVotes
			}
			lastRefill = lastRefill + (tokensToAdd * refillRatePerSec)

			_, _ = r.rdb.Pipelined(ctx, func(pipe redis.Pipeliner) error {
				pipe.HSet(ctx, bucketKey, "remaining", remaining, "last_refill", lastRefill)
				pipe.Expire(ctx, bucketKey, 48*time.Hour)
				return nil
			})
		}
	}

	return remaining, nil
}

func (r *voteRepo) PopDirtyPolls(ctx context.Context) ([]string, error) {
	pipe := r.rdb.Pipeline()
	members := pipe.SMembers(ctx, dirtyPollsKey)
	pipe.Del(ctx, dirtyPollsKey)
	if _, err := pipe.Exec(ctx); err != nil {
		return nil, err
	}
	return members.Val(), nil
}

func (r *voteRepo) EditTitle(ctx context.Context, voteID, newTitle string) error {
	parsedID, err := parseVoteID(voteID)
	if err != nil {
		return err
	}
	filter := bson.M{
		"_id":        parsedID,
		"is_deleted": false,
	}

	update := bson.M{
		"$set": bson.M{
			"title": newTitle,
		},
	}

	_, err = r.mongoDB.Collection(r.voteCollection).UpdateOne(ctx, filter, update)
	return err
}

func (r *voteRepo) GetSnapshots(ctx context.Context, pollIDs []string) ([]PollSnapshot, error) {
	snapshots := make([]PollSnapshot, 0, len(pollIDs))

	for _, pollID := range pollIDs {
		parsedID, err := parseVoteID(pollID)
		if err != nil {
			continue
		}
		counts, err := r.getVoteCount(ctx, parsedID)
		if err != nil {
			continue
		}
		options := make([]OptionSnapshot, 0, len(counts))
		for optID, count := range counts {
			options = append(options, OptionSnapshot{
				OptionId:  strconv.FormatInt(optID, 10),
				VoteCount: count,
			})
		}

		snapshots = append(snapshots, PollSnapshot{
			PollId:  pollID,
			Options: options,
		})
	}

	return snapshots, nil
}

func (r *voteRepo) CloseVoteInMongo(ctx context.Context, voteID string) error {
	parsedID, err := parseVoteID(voteID)
	if err != nil {
		return err
	}
	filter := bson.M{
		"_id":        parsedID,
		"is_deleted": false,
	}

	update := bson.M{
		"$set": bson.M{
			"status": "closed",
		},
	}

	_, err = r.mongoDB.Collection(r.voteCollection).UpdateOne(ctx, filter, update)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return fmt.Errorf("document not found")
		}
		return err
	}
	return nil
}

func (r *voteRepo) DeleteVoteInRedis(ctx context.Context, voteID string) error {
	parsedID, err := parseVoteID(voteID)
	if err != nil {
		return err
	}
	return r.rdb.Del(ctx, voteKey(parsedID)).Err()
}

func (r *voteRepo) HardDeleteVote(ctx context.Context, voteID int64) error {
	filter := bson.M{
		"_id": voteID,
	}
	_, err := r.mongoDB.Collection(r.voteCollection).DeleteOne(ctx, filter)
	return err
}

func (r *voteRepo) GetPollsFromIDs(ctx context.Context, IDs []string) ([]Vote, error) {
	parsedIDs := make([]int64, 0, len(IDs))
	for _, id := range IDs {
		parsedID, err := parseVoteID(id)
		if err != nil {
			continue
		}
		parsedIDs = append(parsedIDs, parsedID)
	}
	filter := bson.M{
		"_id": bson.M{
			"$in": parsedIDs,
		},
	}

	cur, err := r.mongoDB.Collection(r.voteCollection).Find(ctx, filter)
	if err != nil {
		return nil, err
	}

	var votes []Vote
	for cur.Next(ctx) {
		var vote Vote
		if err := cur.Decode(&vote); err == nil {
			options, err := r.getOptionsByVoteID(ctx, vote.ID)
			if err == nil {
				vote.Options = options
			}

			voteCount, err := r.getVoteCount(ctx, vote.ID)
			if err == nil {
				for i, option := range vote.Options {
					vote.Options[i].VoteCount = voteCount[option.ID]
				}
			}
			votes = append(votes, vote)
		}
	}

	return votes, nil
}
