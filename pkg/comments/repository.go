package comments

import (
	"context"
	"voute/pkg/utils"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type CommentRepository interface {
	CreateComment(ctx context.Context, comment *Comment) error
	GetCommentsByVoteID(ctx context.Context, voteID string) ([]*Comment, error)
	DeleteComment(ctx context.Context, commentID string) error
}

type commentRepo struct {
	mongoDB        *mongo.Database
	collectionName string
}

func NewCommentRepository(mongoClient *mongo.Database, collectionName string) CommentRepository {
	return &commentRepo{
		mongoDB:        mongoClient,
		collectionName: collectionName,
	}
}

func (r *commentRepo) CreateComment(ctx context.Context, comment *Comment) error {
	_, err := r.mongoDB.Collection(r.collectionName).InsertOne(ctx, comment)
	return err
}

func (r *commentRepo) GetCommentsByVoteID(ctx context.Context, voteID string) ([]*Comment, error) {
	parsedVoteID, err := utils.ParseSnowflakeID(voteID)
	if err != nil {
		return nil, err
	}
	var comments []*Comment
	filter := bson.M{
		"vote_id":    parsedVoteID,
		"is_deleted": false,
	}

	options := options.Find().SetSort(bson.M{"created_at": -1})

	cursor, err := r.mongoDB.Collection(r.collectionName).Find(
		ctx,
		filter,
		options,
	)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	for cursor.Next(ctx) {
		var comment Comment
		if err := cursor.Decode(&comment); err != nil {
			return nil, err
		}
		comments = append(comments, &comment)
	}
	return comments, nil
}

func (r *commentRepo) DeleteComment(ctx context.Context, commentID string) error {
	parsedCommentID, err := utils.ParseSnowflakeID(commentID)
	if err != nil {
		return err
	}
	update := map[string]any{
		"$set": map[string]any{
			"is_deleted": true,
		},
	}
	_, err = r.mongoDB.Collection(r.collectionName).UpdateOne(
		ctx,
		map[string]any{
			"_id": parsedCommentID,
		},
		update,
	)
	return err
}
