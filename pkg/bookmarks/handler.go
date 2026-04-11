package bookmarks

import (
	"context"
	"net/http"
	"time"
	"voute/pkg/response"

	"github.com/gin-gonic/gin"
)

type BookmarksHandler interface {
	RegisterPaths(r *gin.Engine)
	GetBookMarks(c *gin.Context)
	ChangeBookMarks(c *gin.Context)
	RemoveAllBookmarks(c *gin.Context)
}

type bookmarksHandler struct {
	service BookmarkService
}

func NewBookmarksHandler(s BookmarkService) BookmarksHandler {
	return &bookmarksHandler{
		service: s,
	}
}

func (h *bookmarksHandler) RegisterPaths(r *gin.Engine) {
	bookmarksGroup := r.Group("bookmarks")
	{
		bookmarksGroup.GET("/:userID", h.GetBookMarks)
		bookmarksGroup.PUT("/change", h.ChangeBookMarks)
		bookmarksGroup.DELETE("/:userID", h.RemoveAllBookmarks)
	}
}

func (h *bookmarksHandler) ChangeBookMarks(c *gin.Context) {
	ctx, cancle := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancle()

	var req BookmarkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.SendResponse(c, http.StatusBadRequest, "error", "invalid request", nil)
		return
	}

	if req.Flag {
		if err := h.service.AddToBookmakrs(ctx, req.UserID, req.VoteID); err != nil {
			response.SendResponse(c, http.StatusInternalServerError, "error", "something went wrong", nil)
			return
		}
		response.SendResponse(c, http.StatusOK, "success", "vote added to bookmarks", nil)
	} else {
		if err := h.service.AddToBookmakrs(ctx, req.UserID, req.VoteID); err != nil {
			response.SendResponse(c, http.StatusInternalServerError, "error", "something went wrong", nil)
			return
		}
		response.SendResponse(c, http.StatusOK, "success", "vote removed from bookmarks", nil)
	}
}

func (h *bookmarksHandler) GetBookMarks(c *gin.Context) {
	ctx, cancle := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancle()

	userID := c.Param("userID")
	if userID == "" {
		response.SendResponse(c, http.StatusBadRequest, "error", "invalid user id", nil)
		return
	}

	bookmarks, err := h.service.GetBookmakrs(ctx, userID)
	if err != nil {
		response.SendResponse(c, http.StatusInternalServerError, "error", "something went wrong", nil)
		return
	}
	response.SendResponse(c, http.StatusOK, "success", "bookmarks retrived successfully", bookmarks)
}

func (h *bookmarksHandler) RemoveAllBookmarks(c *gin.Context) {
	ctx, cancle := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancle()

	userID := c.Param("userID")
	if userID == "" {
		response.SendResponse(c, http.StatusBadRequest, "error", "invalid user id", nil)
		return
	}

	if err := h.service.RemoveAllBookmarks(ctx, userID); err != nil {
		response.SendResponse(c, http.StatusInternalServerError, "error", "something went wrong", nil)
		return
	}
	response.SendResponse(c, http.StatusOK, "success", "all bookmarks removed", nil)
}
