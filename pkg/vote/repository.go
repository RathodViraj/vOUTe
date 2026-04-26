package vote

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
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
	GetVotesByCreatorID(ctx context.Context, creatorID string, skip, take int) ([]*Vote, error)
	ListVote(ctx context.Context, skip, take int) ([]*Vote, error)
	ListLiveVote(ctx context.Context, skip, take int) ([]*Vote, error)
	InitVoteInRedis(ctx context.Context, voteID int64, optionIDs []int64) error
	AddVote(ctx context.Context, userID, voteID, optionID string, count int64) error
	GetUserVotedPolls(ctx context.Context, userID string) ([]string, error)
	GetRemainingVotes(ctx context.Context, userID string) (int64, error)
	EditTitle(ctx context.Context, voteID, newTitle string) error
	CloseVoteInMongo(ctx context.Context, voteID string) error
	DeleteVoteInRedis(ctx context.Context, voteID string) error
	GetHistoricData(ctx context.Context, voteID string) (*HistoricDataResponse, error)
	getIntervalForBucketCount(createdAT int64) int
	UpdateStatus(ctx context.Context, id int64, newStatus string) error
	HardDeleteVote(ctx context.Context, voteID int64) error
	GetPollsFromIDs(ctx context.Context, votesIDs []string) ([]Vote, error)
	GetSnapshots(ctx context.Context, pollIDs []string) ([]PollSnapshot, error)
	PopDirtyPolls(ctx context.Context) ([]string, error)
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

func (r *voteRepo) GetVotesByCreatorID(ctx context.Context, creatorID string, skip, take int) ([]*Vote, error) {
	parsedCreatorID, err := parseVoteID(creatorID)
	if err != nil {
		return nil, err
	}
	votes := []*Vote{}
	filter := bson.M{
		"created_by_id": parsedCreatorID,
		"status":        bson.M{"$in": []string{"closed", "live"}},
		"is_deleted":    false,
	}

	options := options.Find()
	options.SetSkip(int64(skip))
	options.SetLimit(int64(take))
	options.SetSort(bson.D{{Key: "created_at", Value: -1}})

	cursor, err := r.mongoDB.Collection(r.voteCollection).Find(ctx, filter, options)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var vote Vote
		if err := cursor.Decode(&vote); err != nil {
			continue
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

		votes = append(votes, &vote)
	}

	return votes, nil
}

func (r *voteRepo) ListVote(ctx context.Context, skip, take int) ([]*Vote, error) {
	votes := []*Vote{}
	filter := bson.M{
		"is_deleted": false,
		"status":     bson.M{"$in": []string{"closed", "live"}},
	}

	options := options.Find()
	options.SetSkip(int64(skip))
	options.SetLimit(int64(take))
	options.SetSort(bson.D{{Key: "created_at", Value: -1}})

	cursor, err := r.mongoDB.Collection(r.voteCollection).Find(ctx, filter, options)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var vote Vote
		if err := cursor.Decode(&vote); err != nil {
			continue
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
		votes = append(votes, &vote)
	}

	return votes, nil
}

func (r *voteRepo) ListLiveVote(ctx context.Context, skip, take int) ([]*Vote, error) {
	votes := []*Vote{}
	filter := bson.M{
		"is_deleted": false,
		"status":     "live",
	}

	options := options.Find()
	options.SetSkip(int64(skip))
	options.SetLimit(int64(take))
	options.SetSort(bson.D{{Key: "created_at", Value: -1}})

	cursor, err := r.mongoDB.Collection(r.voteCollection).Find(ctx, filter, options)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var vote Vote
		if err := cursor.Decode(&vote); err != nil {
			continue
		}

		options, err := r.getOptionsByVoteID(ctx, vote.ID)
		if err != nil {
			continue
		}
		vote.Options = options

		voteCount, err := r.getVoteCount(ctx, vote.ID)
		if err != nil {
			continue
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

	return votes, nil
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
	// Increment the actual vote count
	pipe.HIncrBy(ctx, voteKey(parsedVoteID), strconv.FormatInt(parsedOptionID, 10), count)
	// Mark this poll as dirty for the WS broadcaster
	pipe.SAdd(ctx, dirtyPollsKey, voteID)

	_, err = pipe.Exec(ctx)
	if err != nil {
		return err
	}

	// Vote mutation is already committed in Redis at this point.
	// Snapshot persistence should not fail the request, otherwise clients can
	// see a successful live count update but receive an API error.
	_ = r.saveSnapshot(ctx, parsedVoteID)
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

func (r *voteRepo) GetUserVotedPolls(ctx context.Context, userID string) ([]string, error) {
	parsedUserId, err := parseVoteID(userID)
	if err != nil {
		return nil, err
	}

	cutoff := strconv.FormatInt(time.Now().Add(-24*time.Hour).Unix(), 10)
	return r.rdb.ZRangeByScore(
		ctx,
		userVotedPollsKey(parsedUserId),
		&redis.ZRangeBy{
			Min: cutoff,
			Max: "+inf",
		},
	).Result()
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

func (r *voteRepo) GetHistoricData(ctx context.Context, voteID string) (*HistoricDataResponse, error) {
	parsedID, err := parseVoteID(voteID)
	if err != nil {
		return nil, err
	}
	query := `
		SELECT
		time_bucket_gapfill(make_interval(mins => $1), created_at) AS bucket,
		option_id,
		locf(MAX(vote_count)) AS votes
		FROM vote_snapshots
		WHERE vote_id = $2
		AND created_at >= $3
		AND created_at <= $4
		GROUP BY bucket, option_id
		ORDER BY bucket;
 	`

	end := time.Now().UTC()
	start := end.Add(-24 * time.Hour)

	// Keep chart granularity readable and stable for the "Last 24h" UI.
	// 48 buckets -> one point every 30 minutes.
	const targetBuckets = 48
	interval := int((24 * 60) / targetBuckets)
	res, err := r.timescaleDB.QueryContext(ctx, query, interval, parsedID, start, end)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("no data availble")
		}

		return nil, err
	}

	response := &HistoricDataResponse{
		VoteID:      voteID,
		OptionsData: []HistoricOptionsData{},
	}

	for res.Next() {
		var opData HistoricOptionsData
		var optionID int64
		if err := res.Scan(&opData.Timestamp, &optionID, &opData.VoteCount); err != nil {
			continue
		}
		opData.OptionID = strconv.FormatInt(optionID, 10)

		response.OptionsData = append(response.OptionsData, opData)
	}

	return response, nil
}

func (r *voteRepo) getIntervalForBucketCount(createdAT int64) int {
	now := time.Now()

	switch {
	case createdAT < now.Add(-30*time.Minute).Unix():
		return 2

	case createdAT < now.Add(-90*time.Minute).Unix():
		return 5

	case createdAT < now.Add(-5*time.Hour).Unix():
		return 10

	case createdAT < now.Add(-12*time.Hour).Unix():
		return 20

	case createdAT < now.Add(-2*24*time.Hour).Unix():
		return 60

	case createdAT < now.Add(-7*24*time.Hour).Unix():
		return 180

	default:
		return 1440
	}
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
	filter := &bson.M{
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
		var v Vote
		if err := cur.Decode(&v); err == nil {
			votes = append(votes, v)
		}
	}

	return votes, nil
}
