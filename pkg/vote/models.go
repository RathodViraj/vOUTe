package vote

type Option struct {
	ID        string `json:"id,omitempty" bson:"_id"`
	Text      string `json:"text" bson:"text"`
	VoteCount int64  `json:"vote_count" bson:"vote_count"`
}

type Vote struct {
	ID          string   `json:"id,omitempty" bson:"_id"`
	CreatedByID string   `json:"created_by_id" binding:"created_by_id" bson:"created_by_id"`
	Title       string   `json:"title" bson:"title"`
	Options     []Option `json:"options" bson:"options"`
	Status      string   `json:"status" binding:"oneof=created live closed" bson:"status"`
	IsDeleted   bool     `json:"is_deleted" bson:"is_deleted"`
	CreatedAt   int64    `json:"created_at" bson:"created_at"`
}

type CreateRequest struct {
	Vote    CreateVoteRequest     `json:"vote" binding:"required"`
	Options []CreateOptionRequest `json:"options" binding:"required,dive"`
}

type CreateVoteRequest struct {
	Title       string `json:"title" binding:"required"`
	CreatedByID string `json:"created_by_id" binding:"created_by_id"`
}
type CreateOptionRequest struct {
	VoteID string `json:"vote_id" binding:"required"`
	Text   string `json:"text" binding:"required"`
}

type CreateVoteInMogo struct {
	ID          string `json:"id,omitempty" bson:"_id"`
	Title       string `json:"title" bson:"title"`
	CreatedByID string `json:"created_by_id" bson:"created_by_id"`
	Status      string `json:"status" bson:"status"`
	IsDeleted   bool   `json:"is_deleted" bson:"is_deleted"`
	CreatedAt   int64  `json:"created_at" bson:"created_at"`
}

type VoteRequest struct {
	ID       string `json:"id,omitempty" bson:"_id"`
	OptionID string `json:"option_id" binding:"required"`
	Delta    int8   `json:"delta" binding:"required,oneof=1 -1"`
}

type HistoricDataResponse struct {
	VoteID      string
	OptionsData []HistoricOptionsData
}

type HistoricOptionsData struct {
	Timestamp int64
	OptionID  string
	VoteCount int
}
