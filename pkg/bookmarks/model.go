package bookmarks

type BookmarkRequest struct {
	VoteID string `json:"vote_id" binding:"required"`
	Flag   bool   `json:"flag" binding:"required"`
}

type Bookmark struct {
	VoteID int64 `json:"vote_id,string" bson:"vote_id"`
	UserID int64 `json:"user_id,string" bson:"user_id"`
}
