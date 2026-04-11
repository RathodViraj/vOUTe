package comments

import (
	"context"
	"net/http"
	"time"
	"voute/pkg/response"

	"github.com/gin-gonic/gin"
)

type CommentHandler interface {
	RegisterRoutes(r *gin.Engine)
	CreateComment(c *gin.Context)
	GetCommentsByVoteID(c *gin.Context)
	DeleteComment(c *gin.Context)
}

type commentHandler struct {
	service CommentService
}

func NewCommentHandler(service CommentService) CommentHandler {
	return &commentHandler{
		service: service,
	}
}

func (h *commentHandler) RegisterRoutes(r *gin.Engine) {
	comments := r.Group("/comments")
	{
		comments.POST("/", h.CreateComment)
		comments.GET("/vote/:vote_id", h.GetCommentsByVoteID)
		comments.DELETE("/:comment_id", h.DeleteComment)
	}
}

func (h *commentHandler) CreateComment(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	var req CreateCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.SendResponse(c, http.StatusBadRequest, "error", "invalid request", nil)
		return
	}
	comment := &Comment{
		UserID:    req.UserID,
		VoteID:    req.VoteID,
		Content:   req.Content,
		CreatedAt: time.Now().Unix(),
	}

	if err := h.service.CreateComment(ctx, comment); err != nil {
		response.SendResponse(c, http.StatusInternalServerError, "error", "failed to create comment", nil)
		return
	}
	response.SendResponse(c, http.StatusOK, "success", "comment created successfully", nil)
}

func (h *commentHandler) GetCommentsByVoteID(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	voteID := c.Param("vote_id")
	comments, err := h.service.GetCommentsByVoteID(ctx, voteID)
	if err != nil {
		response.SendResponse(c, http.StatusInternalServerError, "error", "failed to get comments", nil)
		return
	}
	response.SendResponse(c, http.StatusOK, "success", "comments retrieved successfully", comments)
}

func (h *commentHandler) DeleteComment(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	commentID := c.Param("comment_id")
	if err := h.service.DeleteComment(ctx, commentID); err != nil {
		response.SendResponse(c, http.StatusInternalServerError, "error", "failed to delete comment", nil)
		return
	}

	response.SendResponse(c, http.StatusOK, "success", "comment deleted successfully", nil)
}
