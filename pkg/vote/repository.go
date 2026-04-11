package vote

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type VoteRepository interface {
	CreateVoteInMongo(ctx context.Context, vote *CreateVoteInMogo) error
	AddOptionsInMongo(ctx context.Context, voteID string, options []Option) error
	GetVoteByID(ctx context.Context, id string) (*Vote, error)
	GetVotesByCreatorID(ctx context.Context, creatorID string, skip, take int) ([]*Vote, error)
	ListVote(ctx context.Context, skip, take int) ([]*Vote, error)
	ListLiveVote(ctx context.Context, skip, take int) ([]*Vote, error)
	InitVoteInRedis(ctx context.Context, voteID string, optionIDs []string) error
	AddVote(ctx context.Context, voteID, optionID string) error
	RemoveVote(ctx context.Context, voteID, optionID string) error
	EditTitle(ctx context.Context, voteID, newTitle string) error
	CloseVoteInMongo(ctx context.Context, voteID string) error
	DeleteVoteInRedis(ctx context.Context, voteID string) error
	GetHistoricData(ctx context.Context, voteID string) (*HistoricDataResponse, error)
	getIntervalForBucketCount(createdAT int64) int
	UpdateStatus(ctx context.Context, id, newStatus string) error
	HardDeleteVote(ctx context.Context, voteID string) error
	GetPollsFromIDs(ctx context.Context, votesIDs []string) ([]Vote, error)
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

func (r *voteRepo) CreateVoteInMongo(ctx context.Context, vote *CreateVoteInMogo) error {
	_, err := r.mongoDB.Collection(r.voteCollection).InsertOne(ctx, vote)
	return err
}

func (r *voteRepo) AddOptionsInMongo(ctx context.Context, voteID string, options []Option) error {
	var optionDocs []any
	for _, option := range options {
		optionDocs = append(optionDocs, option)
	}

	_, err := r.mongoDB.Collection(r.optionsCollection).InsertMany(ctx, optionDocs)
	return err
}

func (r *voteRepo) InitVoteInRedis(ctx context.Context, voteID string, optionIDs []string) error {
	pipe := r.rdb.Pipeline()
	for _, optionID := range optionIDs {
		pipe.HSet(ctx, "vote:"+voteID, optionID, 0)
	}

	_, err := pipe.Exec(ctx)
	return err
}

func (r *voteRepo) UpdateStatus(ctx context.Context, id, newStatus string) error {
	_, err := r.mongoDB.Collection(r.voteCollection).UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": bson.M{"status": newStatus}})
	return err
}

func (r *voteRepo) GetVoteByID(ctx context.Context, id string) (*Vote, error) {
	var vote Vote
	filter := bson.M{
		"_id":        id,
		"is_deleted": false,
	}

	err := r.mongoDB.Collection(r.voteCollection).FindOne(ctx, filter).Decode(&vote)
	if err != nil {
		return nil, err
	}

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
	votes := []*Vote{}
	filter := bson.M{
		"created_by_id": creatorID,
		"status":        bson.M{"$in": []string{"closed", "live"}},
		"is_deleted":    false,
	}

	options := options.Find()
	options.SetSkip(int64(skip))
	options.SetLimit(int64(take))

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

func (r *voteRepo) getVoteCount(ctx context.Context, voteID string) (map[string]int64, error) {
	result, err := r.rdb.HGetAll(ctx, "vote:"+voteID).Result()
	if err != nil {
		return nil, err
	}
	voteCount := make(map[string]int64)
	for optionID := range result {
		count, err := r.rdb.HGet(ctx, "vote:"+voteID, optionID).Int64()
		if err != nil {
			continue
		}
		voteCount[optionID] = count
	}
	return voteCount, nil
}

func (r *voteRepo) AddVote(ctx context.Context, voteID, optionID string) error {
	return r.rdb.HIncrBy(ctx, "vote:"+voteID, optionID, 1).Err()
}

func (r *voteRepo) RemoveVote(ctx context.Context, voteID, optionID string) error {
	return r.rdb.HIncrBy(ctx, "vote:"+voteID, optionID, -1).Err()
}

func (r *voteRepo) EditTitle(ctx context.Context, voteID, newTitle string) error {
	filter := bson.M{
		"_id":        voteID,
		"is_deleted": false,
	}

	update := bson.M{
		"title": newTitle,
	}

	_, err := r.mongoDB.Collection(r.voteCollection).UpdateOne(ctx, filter, update)
	return err
}

func (r *voteRepo) GetHistoricData(ctx context.Context, voteID string) (*HistoricDataResponse, error) {
	query := `
		SELECT
		time_bucket($1, created_at) AS bucket,
		option_id,
		MAX(vote_count) AS votes
		FROM vote_snapshots
		WHERE vote_id = $2
		GROUP BY bucket, option_id
		ORDER BY bucket;
 	`

	interval := r.getIntervalForBucketCount(time.Now().Unix())
	res, err := r.timescaleDB.QueryContext(ctx, query, interval, voteID)
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
		var op_data HistoricOptionsData
		if err := res.Scan(&op_data.Timestamp, &op_data.OptionID, &op_data.VoteCount); err != nil {
			response.OptionsData = append(response.OptionsData, HistoricOptionsData{
				Timestamp: -1,
			})
			continue
		}

		response.OptionsData = append(response.OptionsData, op_data)
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
	filter := bson.M{
		"_id":        voteID,
		"is_deleted": false,
	}

	update := bson.M{
		"closed": true,
	}

	_, err := r.mongoDB.Collection(r.voteCollection).UpdateOne(ctx, filter, update)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return fmt.Errorf("document not found")
		}
		return err
	}
	return nil
}

func (r *voteRepo) DeleteVoteInRedis(ctx context.Context, voteID string) error {
	return r.rdb.Del(ctx, "vote:"+voteID).Err()
}

func (r *voteRepo) HardDeleteVote(ctx context.Context, voteID string) error {
	filter := bson.M{
		"_id": voteID,
	}
	_, err := r.mongoDB.Collection(r.voteCollection).DeleteOne(ctx, filter)
	return err
}

func (r *voteRepo) GetPollsFromIDs(ctx context.Context, IDs []string) ([]Vote, error) {
	filter := &bson.M{
		"vote_id": bson.M{
			"$in": IDs,
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
