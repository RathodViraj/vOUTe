package comments

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
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
	var comments []*Comment
	filter := bson.M{
		"vote_id":    voteID,
		"is_deleted": false,
	}

	cursor, err := r.mongoDB.Collection(r.collectionName).Find(
		ctx,
		filter,
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
	update := map[string]any{
		"$set": map[string]any{
			"is_deleted": true,
		},
	}
	_, err := r.mongoDB.Collection(r.collectionName).UpdateOne(
		ctx,
		map[string]any{
			"_id": commentID,
		},
		update,
	)
	return err
}
