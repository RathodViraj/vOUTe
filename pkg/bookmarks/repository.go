package bookmarks

import (
	"context"
	"voute/pkg/utils"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type BookmarkRepository interface {
	GetBookmakrs(ctx context.Context, userID string) ([]Bookmark, error)
	AddToBookmakrs(ctx context.Context, b *Bookmark) error
	RemoveFromBookmarks(ctx context.Context, b *Bookmark) error
	RemoveAllBookmarks(ctx context.Context, userID string) error
}

type bookmarkRepo struct {
	mongoDB        *mongo.Database
	voteCollection string
}

func NewBookmarkRepository(db *mongo.Database, vc string) BookmarkRepository {
	return &bookmarkRepo{
		mongoDB:        db,
		voteCollection: vc,
	}
}

func (r *bookmarkRepo) AddToBookmakrs(ctx context.Context, b *Bookmark) error {
	_, err := r.mongoDB.Collection(r.voteCollection).InsertOne(ctx, b)
	return err
}

func (r *bookmarkRepo) GetBookmakrs(ctx context.Context, userID string) ([]Bookmark, error) {
	parsedUserID, err := utils.ParseSnowflakeID(userID)
	if err != nil {
		return nil, err
	}
	filter := &bson.M{
		"user_id": parsedUserID,
	}
	cur, err := r.mongoDB.Collection(r.voteCollection).Find(ctx, filter)
	if err != nil {
		return nil, err
	}

	bookmarks := []Bookmark{}
	for cur.Next(ctx) {
		var b Bookmark
		if err := cur.Decode(&b); err == nil {
			bookmarks = append(bookmarks, b)
		}
	}

	return bookmarks, nil
}

func (r *bookmarkRepo) RemoveAllBookmarks(ctx context.Context, userID string) error {
	parsedUserID, err := utils.ParseSnowflakeID(userID)
	if err != nil {
		return err
	}
	filter := &bson.M{
		"user_id": parsedUserID,
	}
	_, err = r.mongoDB.Collection(r.voteCollection).DeleteMany(ctx, filter)
	return err
}

func (r *bookmarkRepo) RemoveFromBookmarks(ctx context.Context, b *Bookmark) error {
	filter := &bson.M{
		"user_id": b.UserID,
		"vote_id": b.VoteID,
	}
	_, err := r.mongoDB.Collection(r.voteCollection).DeleteOne(ctx, filter)
	return err
}
