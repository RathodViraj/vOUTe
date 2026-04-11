package bookmarks

type BookmarkRequest struct {
	VoteID string `json:"vote_id" binding:"required"`
	UserID string `json:"user_id" binding:"required"`
	Flag   bool   `json:"flag" binding:"required"`
}

type Bookmark struct {
	VoteID string `json:"vote_id" bson:"vote_id"`
	UserID string `json:"user_id" bson:"user_id"`
}
