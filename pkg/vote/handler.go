package vote

import (
	"context"
	"net/http"
	"strconv"
	"time"
	"voute/pkg/middleware"
	"voute/pkg/response"
	"voute/pkg/utils"

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
	HistoricData(c *gin.Context)
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
	register := func(group *gin.RouterGroup) {
		group.POST("/create", middleware.AuthMiddleware(), h.CreateVote)
		group.GET("/:voteID", h.GetVoteByID)
		group.GET("/creator", middleware.AuthMiddleware(), h.GetVotesByCreatorID)
		group.PATCH("/:voteID", h.CloseVote)
		group.PUT("/editTitle", h.EditTitle)
		group.GET("", h.GetPolls)
		group.PUT("/update", middleware.AuthMiddleware(), h.UpdateVote)
		group.GET("/getHistoricData", h.HistoricData)
	}

	register(r.Group("/polls"))
	register(r.Group("/vote"))
}

func (h *voteHandler) CreateVote(c *gin.Context) {
	ctx, cancle := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancle()

	var req CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.SendResponse(c, http.StatusBadRequest, "error", "invalid request", nil)
		return
	}
	claims, ok := middleware.GetClaims(c)
	if !ok || claims.UserID == "" {
		response.SendResponse(c, http.StatusUnauthorized, "error", "invalid auth claims", nil)
		return
	}

	createdByID, err := utils.ParseSnowflakeID(claims.UserID)
	if err != nil {
		response.SendResponse(c, http.StatusBadRequest, "error", "invalid created_by_id", nil)
		return
	}

	vote := &CreateVoteInMogo{
		Title:       req.Vote.Title,
		CreatedByID: createdByID,
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

	response.SendResponse(c, http.StatusCreated, "success", "vote created successfully", nil)
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

	claims, ok := middleware.GetClaims(c)
	if !ok || claims.UserID == "" {
		response.SendResponse(c, http.StatusUnauthorized, "error", "invalid auth claims", nil)
		return
	}

	creatorID := claims.UserID
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

	var req CastVoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.SendResponse(c, http.StatusBadRequest, "error", "invalid request", nil)
		return
	}

	claims, ok := middleware.GetClaims(c)
	if !ok || claims.UserID == "" {
		response.SendResponse(c, http.StatusUnauthorized, "error", "invalid auth claims", nil)
		return
	}

	if err := h.service.AddVote(ctx, claims.UserID, req.ID, req.OptionID, req.Count); err != nil {
		if err == ErrVoteLimitReached {
			response.SendResponse(c, http.StatusTooManyRequests, "error", "vote limit reached", nil)
			return
		}
		response.SendResponse(c, http.StatusInternalServerError, "error", "failed to add vote", nil)
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

func (h *voteHandler) HistoricData(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	var req struct {
		IDS []string `json:"ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.SendResponse(c, http.StatusBadRequest, "error", "invalid request", nil)
		return
	}
	data, err := h.service.GetHistoricData(ctx, req.IDS)
	if err != nil {
		response.SendResponse(c, http.StatusInternalServerError, "error", "failed to get historic data", nil)
		return
	}
	response.SendResponse(c, http.StatusOK, "success", "historic data retrieved successfully", data)
}
