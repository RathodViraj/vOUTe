package vote

import "time"

type UserVote struct {
	OptionID  int64 `json:"option_id,string"`
	VoteCount int64 `json:"vote_count"`
}

type Option struct {
	ID        int64  `json:"id,omitempty,string" bson:"_id"`
	VoteID    int64  `json:"vote_id,string" bson:"vote_id"`
	Text      string `json:"text" bson:"text"`
	VoteCount int64  `json:"vote_count" bson:"vote_count"`
}

type Vote struct {
	ID          int64     `json:"id,omitempty,string" bson:"_id"`
	CreatedByID int64     `json:"created_by_id,string" binding:"required" bson:"created_by_id"`
	Title       string    `json:"title" bson:"title"`
	Options     []Option  `json:"options" bson:"options"`
	Status      string    `json:"status" binding:"oneof=created live closed" bson:"status"`
	IsDeleted   bool      `json:"is_deleted" bson:"is_deleted"`
	CreatedAt   int64     `json:"created_at" bson:"created_at"`
	UserVote    *UserVote `json:"user_vote,omitempty" bson:"-"`
}

type UserVotedPoll struct {
	VoteID    int64
	OptionID  int64
	VoteCount int64
}

type CreateRequest struct {
	Vote    CreateVoteRequest     `json:"vote" binding:"required"`
	Options []CreateOptionRequest `json:"options" binding:"required,dive"`
}

type CreateVoteRequest struct {
	Title string `json:"title" binding:"required"`
}
type CreateOptionRequest struct {
	Text string `json:"text" binding:"required"`
}

type CreateVoteInMogo struct {
	ID          int64  `json:"id,omitempty,string" bson:"_id"`
	Title       string `json:"title" bson:"title"`
	CreatedByID int64  `json:"created_by_id,string" bson:"created_by_id"`
	Status      string `json:"status" bson:"status"`
	IsDeleted   bool   `json:"is_deleted" bson:"is_deleted"`
	CreatedAt   int64  `json:"created_at" bson:"created_at"`
}

type CastVoteRequest struct {
	ID       string `json:"id,omitempty" bson:"_id,omitempty"`
	OptionID string `json:"option_id" binding:"required"`
	Count    int64  `json:"count" binding:"required,min=1"`
}

type VoteRequest = CastVoteRequest

type HistoricDataResponse struct {
	VoteID      string
	OptionsData []HistoricOptionsData
}

type HistoricOptionsData struct {
	Timestamp time.Time
	OptionID  string
	VoteCount int
}

type PollSnapshot struct {
	PollId  string           `json:"poll_id"`
	Options []OptionSnapshot `json:"options"`
}

type OptionSnapshot struct {
	OptionId  string `json:"option_id"`
	VoteCount int64  `json:"vote_count"`
}
