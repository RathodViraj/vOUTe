package comments

type Comment struct {
	ID        int64  `json:"id,omitempty,string" bson:"_id"`
	Username  string `json:"username" bson:"username"`
	VoteID    int64  `json:"vote_id,string" bson:"vote_id"`
	Content   string `json:"content" bson:"content"`
	IsDeleted bool   `json:"is_deleted" bson:"is_deleted"`
	CreatedAt int64  `json:"created_at,omitempty" bson:"created_at"`
}

type CreateCommentRequest struct {
	VoteID   string `json:"vote_id" binding:"required"`
	Username string `json:"username" binding:"required"`
	Content  string `json:"content" binding:"required"`
}
