package vote

import (
	"context"
	"time"
	"voute/pkg/utils"
)

type VoteService interface {
	CreateVote(ctx context.Context, vote *CreateVoteInMogo, options []Option) error
	ListLiveVote(ctx context.Context, skip, take int) ([]*Vote, error)
	ListVote(ctx context.Context, skip, take int) ([]*Vote, error)
	GetVoteByID(ctx context.Context, id string) (*Vote, error)
	GetVotesByCreatorID(ctx context.Context, creatorID string, skip, take int) ([]*Vote, error)
	GetUserVotedPolls(ctx context.Context, userID string) ([]string, error)
	GetRemainingVotes(ctx context.Context, userID string) (int64, error)
	AddVote(ctx context.Context, userID, voteID, optionID string, count int64) error
	CloseVote(ctx context.Context, voteID string) error
	EditTitle(ctx context.Context, voteID, newTitle string) error
	GetFromIDs(ctx context.Context, voteIDs []string) ([]Vote, error)
	GetHistoricData(ctx context.Context, voteIDs []string) (*[]HistoricDataResponse, error)
}

type voteService struct {
	repo VoteRepository
}

func NewVoteService(repo VoteRepository) VoteService {
	return &voteService{
		repo: repo,
	}
}

func (s *voteService) CreateVote(ctx context.Context, vote *CreateVoteInMogo, options []Option) error {
	if vote.ID == 0 {
		vote.ID = utils.GenerateSnowflakeID()
	}
	vote.CreatedAt = time.Now().Unix()
	for i := range options {
		options[i].ID = utils.GenerateSnowflakeID()
		options[i].VoteID = vote.ID
	}
	if err := s.repo.CreateVoteInMongo(ctx, vote); err != nil {
		return err
	}

	optionIDs := make([]int64, len(options))
	for i := range options {
		optionIDs[i] = options[i].ID
	}
	if err := s.repo.AddOptionsInMongo(ctx, vote.ID, options); err != nil {
		s.repo.HardDeleteVote(ctx, vote.ID)
		return err
	}

	if err := s.repo.InitVoteInRedis(ctx, vote.ID, optionIDs); err != nil {
		s.repo.HardDeleteVote(ctx, vote.ID)
		return err
	}

	if err := s.repo.UpdateStatus(ctx, vote.ID, "live"); err != nil {
		s.repo.HardDeleteVote(ctx, vote.ID)
		return err
	}

	return nil
}

func (s *voteService) ListLiveVote(ctx context.Context, skip, take int) ([]*Vote, error) {
	if skip < 0 {
		skip = 0
	}
	if take <= 0 || take > 100 {
		take = 10
	}

	return s.repo.ListLiveVote(ctx, skip, take)
}

func (s *voteService) ListVote(ctx context.Context, skip, take int) ([]*Vote, error) {
	if skip < 0 {
		skip = 0
	}
	if take <= 0 || take > 100 {
		take = 10
	}
	return s.repo.ListVote(ctx, skip, take)
}

func (s *voteService) GetVoteByID(ctx context.Context, id string) (*Vote, error) {
	return s.repo.GetVoteByID(ctx, id)
}

func (s *voteService) GetVotesByCreatorID(ctx context.Context, creatorID string, skip, take int) ([]*Vote, error) {
	if skip < 0 {
		skip = 0
	}
	if take <= 0 || take > 100 {
		take = 10
	}
	return s.repo.GetVotesByCreatorID(ctx, creatorID, skip, take)
}

func (s *voteService) AddVote(ctx context.Context, userID, voteID, optionID string, count int64) error {
	return s.repo.AddVote(ctx, userID, voteID, optionID, count)
}

func (s *voteService) GetUserVotedPolls(ctx context.Context, userID string) ([]string, error) {
	return s.repo.GetUserVotedPolls(ctx, userID)
}

func (s *voteService) GetRemainingVotes(ctx context.Context, userID string) (int64, error) {
	return s.repo.GetRemainingVotes(ctx, userID)
}

func (s *voteService) CloseVote(ctx context.Context, voteID string) error {
	if err := s.repo.CloseVoteInMongo(ctx, voteID); err != nil {
		return err
	}
	if err := s.repo.DeleteVoteInRedis(ctx, voteID); err != nil {
		return err
	}
	return nil
}

func (s *voteService) EditTitle(ctx context.Context, voteID, newTitle string) error {
	return s.repo.EditTitle(ctx, voteID, newTitle)
}

func (s *voteService) GetHistoricData(ctx context.Context, voteIDs []string) (*[]HistoricDataResponse, error) {
	var result []HistoricDataResponse

	for _, voteID := range voteIDs {
		data, err := s.repo.GetHistoricData(ctx, voteID)
		if err == nil {
			result = append(result, *data)
		}
	}

	return &result, nil
}

func (s *voteService) GetFromIDs(ctx context.Context, voteIDs []string) ([]Vote, error) {
	return s.repo.GetPollsFromIDs(ctx, voteIDs)
}
