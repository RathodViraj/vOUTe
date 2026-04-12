package vote

import (
	"context"
	"net/http"
	"strconv"
	"time"
	"voute/pkg/response"

	"github.com/gin-gonic/gin"
)

type VoteHandler interface {
	CreateVote(c *gin.Context)
	GetVoteByID(c *gin.Context)
	GetVotesByCreatorID(c *gin.Context)
	CloseVote(c *gin.Context)
	UpdateVote(c *gin.Context)
	EditTitle(c *gin.Context)
	GetPolls(c *gin.Context)
	AddVoteRoutes(r *gin.Engine)
}

type voteHandler struct {
	service VoteService
}

func NewVoteHandler(service VoteService) VoteHandler {
	return &voteHandler{
		service: service,
	}
}

func (h *voteHandler) AddVoteRoutes(r *gin.Engine) {
	voteGroup := r.Group("/vote")
	{
		voteGroup.POST("/create", h.CreateVote)
		voteGroup.GET("/:voteID", h.GetVoteByID)
		voteGroup.GET("/:userID", h.GetVotesByCreatorID)
		voteGroup.PATCH("/:voteID", h.CloseVote)
		voteGroup.PUT("/editTitle", h.EditTitle)
		voteGroup.GET("", h.GetPolls)
		voteGroup.GET("/fromIds", h.GetFromIDs)
	}
}

func (h *voteHandler) CreateVote(c *gin.Context) {
	ctx, cancle := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancle()

	var req CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.SendResponse(c, http.StatusBadRequest, "success", "invalid request", nil)
		return
	}

	vote := &CreateVoteInMogo{
		Title:       req.Vote.Title,
		CreatedByID: req.Vote.CreatedByID,
		Status:      "created",
		IsDeleted:   false,
		CreatedAt:   time.Now().Unix(),
	}
	options := make([]Option, len(req.Options))
	for i, optionReq := range req.Options {
		options[i] = Option{
			Text: optionReq.Text,
		}
	}

	if err := h.service.CreateVote(ctx, vote, options); err != nil {
		response.SendResponse(c, http.StatusInternalServerError, "error", "failed to create vote", nil)
		return
	}
}

func (h *voteHandler) GetVoteByID(c *gin.Context) {
	ctx, cancle := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancle()

	voteID := c.Param("voteID")
	vote, err := h.service.GetVoteByID(ctx, voteID)
	if err != nil {
		response.SendResponse(c, http.StatusInternalServerError, "error", "failed to get vote", nil)
		return
	}

	response.SendResponse(c, http.StatusOK, "success", "vote retrieved successfully", vote)
}

func (h *voteHandler) GetPolls(c *gin.Context) {
	ctx, cancle := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancle()

	skip_int, take_int := 0, 10
	skip := c.Query("skip")
	if skip != "" {
		if val, err := strconv.Atoi(skip); err == nil {
			skip_int = val
		}
	}
	take := c.Query("take")
	if take != "" {
		if val, err := strconv.Atoi(take); err == nil {
			take_int = val
		}
	}

	if c.Query("status") == "live" {
		votes, err := h.service.ListLiveVote(ctx, skip_int, take_int)
		if err != nil {
			response.SendResponse(c, http.StatusInternalServerError, "error", "failed to get votes", nil)
			return
		}
		response.SendResponse(c, http.StatusOK, "success", "votes retrieved successfully", votes)
	} else {
		votes, err := h.service.ListVote(ctx, skip_int, take_int)
		if err != nil {
			response.SendResponse(c, http.StatusInternalServerError, "error", "failed to get votes", nil)
			return
		}
		response.SendResponse(c, http.StatusOK, "success", "votes retrieved successfully", votes)
	}
}

func (h *voteHandler) GetVotesByCreatorID(c *gin.Context) {
	ctx, cancle := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancle()

	creatorID := c.Param("userID")
	votes, err := h.service.GetVotesByCreatorID(ctx, creatorID, 0, 10)
	if err != nil {
		response.SendResponse(c, http.StatusInternalServerError, "error", "failed to get votes", nil)
		return
	}

	response.SendResponse(c, http.StatusOK, "success", "votes retrieved successfully", votes)
}

func (h *voteHandler) CloseVote(c *gin.Context) {
	ctx, cancle := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancle()

	voteID := c.Param("voteID")
	if err := h.service.CloseVote(ctx, voteID); err != nil {
		response.SendResponse(c, http.StatusInternalServerError, "error", "failed to close vote", nil)
		return
	}
	response.SendResponse(c, http.StatusOK, "success", "vote closed successfully", nil)
}

func (h *voteHandler) UpdateVote(c *gin.Context) {
	ctx, cancle := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancle()

	var req VoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.SendResponse(c, http.StatusBadRequest, "error", "invalid request", nil)
		return
	}

	switch req.Delta {
	case 1:
		if err := h.service.AddVote(ctx, req.ID, req.OptionID); err != nil {
			response.SendResponse(c, http.StatusInternalServerError, "error", "failed to add vote", nil)
			return
		}
	case -1:
		if err := h.service.RemoveVote(ctx, req.ID, req.OptionID); err != nil {
			response.SendResponse(c, http.StatusInternalServerError, "error", "failed to remove vote", nil)
			return
		}
	default:
		response.SendResponse(c, http.StatusBadRequest, "error", "invalid delta value", nil)
		return
	}

	response.SendResponse(c, http.StatusOK, "success", "vote updated successfully", nil)
}

func (h *voteHandler) EditTitle(c *gin.Context) {
	ctx, cancle := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancle()

	var req struct {
		VoteID   string `json:"vote_id" binding:"required"`
		NewTitle string `json:"new_title" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.SendResponse(c, http.StatusBadRequest, "error", "invalid request", nil)
		return
	}
	if err := h.service.EditTitle(ctx, req.VoteID, req.NewTitle); err != nil {
		response.SendResponse(c, http.StatusInternalServerError, "error", "failed to edit title", nil)
		return
	}

	response.SendResponse(c, http.StatusOK, "success", "title edited successfully", nil)
}

func (h *voteHandler) GetFromIDs(c *gin.Context) {
	ctx, cancle := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancle()

	var req struct {
		IDs []string `json:"ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.SendResponse(c, http.StatusBadRequest, "error", "invalid request", nil)
		return
	}

	votes, err := h.service.GetFromIDs(ctx, req.IDs)
	if err != nil {
		response.SendResponse(c, http.StatusInternalServerError, "error", "something went wrong", nil)
		return
	}

	response.SendResponse(c, http.StatusOK, "success", "successfully fetch polls from ids", votes)
}
