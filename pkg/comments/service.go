package comments

import "context"

type CommentService interface {
	CreateComment(ctx context.Context, comment *Comment) error
	GetCommentsByVoteID(ctx context.Context, voteID string) ([]*Comment, error)
	DeleteComment(ctx context.Context, commentID string) error
}

type commentService struct {
	repo CommentRepository
}

func NewCommentService(repo CommentRepository) CommentService {
	return &commentService{
		repo: repo,
	}
}

func (s *commentService) CreateComment(ctx context.Context, comment *Comment) error {
	return s.repo.CreateComment(ctx, comment)
}

func (s *commentService) GetCommentsByVoteID(ctx context.Context, voteID string) ([]*Comment, error) {
	return s.repo.GetCommentsByVoteID(ctx, voteID)
}

func (s *commentService) DeleteComment(ctx context.Context, commentID string) error {
	return s.repo.DeleteComment(ctx, commentID)
}
