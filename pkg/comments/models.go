package comments

type Comment struct {
	ID        int64  `json:"id,omitempty,string" bson:"_id"`
	UserID    int64  `json:"user_id,string" bson:"user_id"`
	VoteID    int64  `json:"vote_id,string" bson:"vote_id"`
	Content   string `json:"content" bson:"content"`
	CreatedAt int64  `json:"created_at,omitempty" bson:"created_at"`
}

type CreateCommentRequest struct {
	VoteID  string `json:"vote_id" binding:"required"`
	Content string `json:"content" binding:"required"`
}
