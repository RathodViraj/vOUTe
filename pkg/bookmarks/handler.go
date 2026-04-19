package bookmarks

import (
	"context"
	"net/http"
	"time"
	"voute/pkg/middleware"
	"voute/pkg/response"

	"github.com/gin-gonic/gin"
)

type BookmarksHandler interface {
	AddBookmarksRoutes(r *gin.Engine)
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

func (h *bookmarksHandler) AddBookmarksRoutes(r *gin.Engine) {
	bookmarksGroup := r.Group("bookmarks")
	{
		bookmarksGroup.GET("", middleware.AuthMiddleware(), h.GetBookMarks)
		bookmarksGroup.PUT("/change", middleware.AuthMiddleware(), h.ChangeBookMarks)
		bookmarksGroup.DELETE("", middleware.AuthMiddleware(), h.RemoveAllBookmarks)
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

	claims, ok := middleware.GetClaims(c)
	if !ok || claims.UserID == "" {
		response.SendResponse(c, http.StatusUnauthorized, "error", "invalid auth claims", nil)
		return
	}

	if req.Flag {
		if err := h.service.AddToBookmakrs(ctx, claims.UserID, req.VoteID); err != nil {
			response.SendResponse(c, http.StatusInternalServerError, "error", "something went wrong", nil)
			return
		}
		response.SendResponse(c, http.StatusOK, "success", "vote added to bookmarks", nil)
	} else {
		if err := h.service.RemoveFromBookmarks(ctx, claims.UserID, req.VoteID); err != nil {
			response.SendResponse(c, http.StatusInternalServerError, "error", "something went wrong", nil)
			return
		}
		response.SendResponse(c, http.StatusOK, "success", "vote removed from bookmarks", nil)
	}
}

func (h *bookmarksHandler) GetBookMarks(c *gin.Context) {
	ctx, cancle := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancle()

	claims, ok := middleware.GetClaims(c)
	if !ok || claims.UserID == "" {
		response.SendResponse(c, http.StatusBadRequest, "error", "invalid user id", nil)
		return
	}

	bookmarks, err := h.service.GetBookmakrs(ctx, claims.UserID)
	if err != nil {
		response.SendResponse(c, http.StatusInternalServerError, "error", "something went wrong", nil)
		return
	}
	response.SendResponse(c, http.StatusOK, "success", "bookmarks retrived successfully", bookmarks)
}

func (h *bookmarksHandler) RemoveAllBookmarks(c *gin.Context) {
	ctx, cancle := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancle()

	claims, ok := middleware.GetClaims(c)
	if !ok || claims.UserID == "" {
		response.SendResponse(c, http.StatusBadRequest, "error", "invalid user id", nil)
		return
	}

	if err := h.service.RemoveAllBookmarks(ctx, claims.UserID); err != nil {
		response.SendResponse(c, http.StatusInternalServerError, "error", "something went wrong", nil)
		return
	}
	response.SendResponse(c, http.StatusOK, "success", "all bookmarks removed", nil)
}
