package bookmarks

import (
	"context"
	"voute/pkg/utils"
)

type BookmarkService interface {
	GetBookmakrs(ctx context.Context, userID string) ([]Bookmark, error)
	AddToBookmakrs(ctx context.Context, uesrID, voteID string) error
	RemoveFromBookmarks(ctx context.Context, uesrID, voteID string) error
	RemoveAllBookmarks(ctx context.Context, userID string) error
}

type bookmarkService struct {
	repo BookmarkRepository
}

func NewBookmarkService(repo BookmarkRepository) BookmarkService {
	return &bookmarkService{
		repo: repo,
	}
}

func (s *bookmarkService) AddToBookmakrs(ctx context.Context, userID string, voteID string) error {
	parsedUserID, err := utils.ParseSnowflakeID(userID)
	if err != nil {
		return err
	}
	parsedVoteID, err := utils.ParseSnowflakeID(voteID)
	if err != nil {
		return err
	}
	b := &Bookmark{
		UserID: parsedUserID,
		VoteID: parsedVoteID,
	}
	return s.repo.AddToBookmakrs(ctx, b)
}

func (s *bookmarkService) GetBookmakrs(ctx context.Context, userID string) ([]Bookmark, error) {
	return s.repo.GetBookmakrs(ctx, userID)
}

func (s *bookmarkService) RemoveAllBookmarks(ctx context.Context, userID string) error {
	return s.repo.RemoveAllBookmarks(ctx, userID)
}

func (s *bookmarkService) RemoveFromBookmarks(ctx context.Context, userID string, voteID string) error {
	parsedUserID, err := utils.ParseSnowflakeID(userID)
	if err != nil {
		return err
	}
	parsedVoteID, err := utils.ParseSnowflakeID(voteID)
	if err != nil {
		return err
	}
	b := &Bookmark{
		UserID: parsedUserID,
		VoteID: parsedVoteID,
	}
	return s.repo.RemoveFromBookmarks(ctx, b)
}
