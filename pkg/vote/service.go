package vote

import (
	"context"
	"time"
	"voute/pkg/utils"
)

type VoteService interface {
	CreateVote(ctx context.Context, vote *CreateVoteInMogo, options []Option) error
	ListLiveVotePage(ctx context.Context, cursor string, take int) ([]*Vote, string, error)
	ListVotePage(ctx context.Context, cursor string, take int) ([]*Vote, string, error)
	GetVoteByID(ctx context.Context, id string) (*Vote, error)
	GetVotesByCreatorIDPage(ctx context.Context, creatorID, cursor string, take int) ([]*Vote, string, error)
	GetUserVotedPolls(ctx context.Context, userID string) ([]UserVotedPoll, error)
	GetRemainingVotes(ctx context.Context, userID string) (int64, error)
	AddVote(ctx context.Context, userID, voteID, optionID string, count int64) error
	CloseVote(ctx context.Context, voteID string) error
	EditTitle(ctx context.Context, voteID, newTitle string) error
	GetFromIDs(ctx context.Context, voteIDs []string) ([]Vote, error)
	GetPollHistory(ctx context.Context, voteID string, cursor string) ([]HistoricOptionsData, error)
	DeleteVote(ctx context.Context, voteID string) error
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

func (s *voteService) ListVotePage(ctx context.Context, cursor string, take int) ([]*Vote, string, error) {
	if take <= 0 || take > 100 {
		take = 50
	}
	return s.repo.ListVotePage(ctx, cursor, take)
}

func (s *voteService) ListLiveVotePage(ctx context.Context, cursor string, take int) ([]*Vote, string, error) {
	if take <= 0 || take > 100 {
		take = 50
	}
	return s.repo.ListLiveVotePage(ctx, cursor, take)
}

func (s *voteService) GetVoteByID(ctx context.Context, id string) (*Vote, error) {
	return s.repo.GetVoteByID(ctx, id)
}

func (s *voteService) GetVotesByCreatorIDPage(ctx context.Context, creatorID, cursor string, take int) ([]*Vote, string, error) {
	if take <= 0 || take > 100 {
		take = 20
	}
	return s.repo.GetVotesByCreatorIDPage(ctx, creatorID, cursor, take)
}

func (s *voteService) AddVote(ctx context.Context, userID, voteID, optionID string, count int64) error {
	return s.repo.AddVote(ctx, userID, voteID, optionID, count)
}

func (s *voteService) GetUserVotedPolls(ctx context.Context, userID string) ([]UserVotedPoll, error) {
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

func (s *voteService) GetPollHistory(ctx context.Context, voteID string, cursor string) ([]HistoricOptionsData, error) {
	return s.repo.GetPollHistory(ctx, voteID, cursor)
}

func (s *voteService) DeleteVote(ctx context.Context, voteID string) error {
	// attempt to remove metadata in Redis first
	if err := s.repo.DeleteVoteInRedis(ctx, voteID); err != nil {
		// log but continue to attempt hard delete
	}

	parsedID, err := parseVoteID(voteID)
	if err != nil {
		return err
	}

	return s.repo.HardDeleteVote(ctx, parsedID)
}

func (s *voteService) GetFromIDs(ctx context.Context, voteIDs []string) ([]Vote, error) {
	return s.repo.GetPollsFromIDs(ctx, voteIDs)
}
