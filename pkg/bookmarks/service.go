package bookmarks

import (
	"context"
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
	b := &Bookmark{
		UserID: userID,
		VoteID: voteID,
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
	b := &Bookmark{
		UserID: userID,
		VoteID: voteID,
	}
	return s.repo.RemoveFromBookmarks(ctx, b)
}
