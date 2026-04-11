package comments

type Comment struct {
	ID        string `json:"id" bson:"_id,omitempty"`
	UserID    string `json:"user_id" bson:"user_id"`
	VoteID    string `json:"vote_id" bson:"vote_id"`
	Content   string `json:"content" bson:"content"`
	CreatedAt int64  `json:"created_at,omitempty" bson:"created_at"`
}

type CreateCommentRequest struct {
	UserID  string `json:"user_id" binding:"required"`
	VoteID  string `json:"vote_id" binding:"required"`
	Content string `json:"content" binding:"required"`
}
