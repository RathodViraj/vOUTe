package user

import (
	"context"
	"net/http"
	"time"
	"voute/pkg/bloom"
	"voute/pkg/middleware"
	"voute/pkg/response"

	"github.com/gin-gonic/gin"
)

type UserHandler interface {
	AddUserRoutes(r *gin.Engine)
	checkUsernameExists(c *gin.Context)
	createUser(c *gin.Context)
	getUserByEmail(c *gin.Context)
	getUserByID(c *gin.Context)
	updateUser(c *gin.Context)
	updatePassword(c *gin.Context)
	deleteUser(c *gin.Context)
}

type userHandler struct {
	userService UserService
	bloom       *bloom.Filter
}

func NewHandler(srv UserService, bf *bloom.Filter) UserHandler {
	return &userHandler{
		userService: srv,
		bloom:       bf,
	}
}

func (h *userHandler) AddUserRoutes(r *gin.Engine) {
	userGroup := r.Group("/users")
	{
		userGroup.GET("/check", h.checkUsernameExists)
		userGroup.POST("/create", h.createUser)
		userGroup.GET("/email/:email", h.getUserByEmail)
		userGroup.GET("/me", middleware.AuthMiddleware(), h.getUserByID)
		userGroup.PUT("/update", middleware.AuthMiddleware(), h.updateUser)
		userGroup.PUT("/updatePassword", h.updatePassword)
		userGroup.DELETE("/delete", middleware.AuthMiddleware(), h.deleteUser)
	}
}

func (h *userHandler) checkUsernameExists(c *gin.Context) {
	username := c.Query("username")
	if username == "" {
		response.SendResponse(c, http.StatusBadRequest, "error", "username is required", nil)
		return
	}

	if h.bloom.MightExist(username) {
		response.SendResponse(c, http.StatusOK, "success", "username might exist", nil)
		return
	}

	response.SendResponse(c, http.StatusOK, "success", "username does not exist", nil)
}

func (h *userHandler) createUser(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.SendResponse(c, http.StatusBadRequest, "error", "invalid data", nil)
		return
	}

	if h.bloom.MightExist(req.Username) {
		response.SendResponse(c, http.StatusBadRequest, "error", "username already exists", nil)
		return
	}

	user, err := h.userService.CreateUser(ctx, req.Username, req.Email, req.Password, req.Role)
	if err != nil {
		response.SendResponse(c, http.StatusInternalServerError, "error", "failed to create user", err.Error())
		return
	}

	h.bloom.Add(req.Username)
	user.Password = ""
	response.SendResponse(c, http.StatusOK, "success", "user created successfully", user)
}

func (h *userHandler) getUserByEmail(c *gin.Context) {
	email := c.Param("email")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	user, err := h.userService.GetUserByEmail(ctx, email)
	if err != nil {
		response.SendResponse(c, http.StatusInternalServerError, "error", "failed to get user", err.Error())
		return
	}

	if user == nil {
		response.SendResponse(c, http.StatusNotFound, "error", "user not found", nil)
		return
	}
	user.Password = ""
	response.SendResponse(c, http.StatusOK, "success", "user found", user)
}

func (h *userHandler) getUserByID(c *gin.Context) {
	claims, ok := middleware.GetClaims(c)
	if !ok || claims.UserID == "" {
		response.SendResponse(c, http.StatusUnauthorized, "error", "invalid auth claims", nil)
		return
	}

	id := claims.UserID
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	user, err := h.userService.GetUserByID(ctx, id)
	if err != nil {
		response.SendResponse(c, http.StatusInternalServerError, "error", "failed to get user", nil)
		return
	}

	if user == nil {
		response.SendResponse(c, http.StatusNotFound, "error", "user not found", nil)
		return
	}

	user.Password = ""
	response.SendResponse(c, http.StatusOK, "success", "user found", user)
}
func (h *userHandler) updateUser(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.SendResponse(c, http.StatusBadRequest, "error", "invalid data", nil)
		return
	}

	claims, ok := middleware.GetClaims(c)
	if !ok || claims.UserID == "" {
		response.SendResponse(c, http.StatusUnauthorized, "error", "invalid auth claims", nil)
		return
	}

	if err := h.userService.UpdateUser(ctx, req.Username, req.Email, claims.UserID); err != nil {
		response.SendResponse(c, http.StatusInternalServerError, "error", "failed to update user", nil)
		return
	}

	response.SendResponse(c, http.StatusOK, "success", "user updated successfully", nil)
}

func (h *userHandler) updatePassword(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var req UpdatePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.SendResponse(c, http.StatusBadRequest, "error", "invalid data", nil)
		return
	}

	if err := h.userService.UpdatePassword(ctx, req.Email, req.NewPassword); err != nil {
		response.SendResponse(c, http.StatusInternalServerError, "error", "failed to update password", nil)
		return
	}

	response.SendResponse(c, http.StatusOK, "success", "password updated successfully", nil)
}

func (h *userHandler) deleteUser(c *gin.Context) {
	claims, ok := middleware.GetClaims(c)
	if !ok || claims.UserID == "" {
		response.SendResponse(c, http.StatusUnauthorized, "error", "invalid auth claims", nil)
		return
	}

	id := claims.UserID
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := h.userService.DeleteUser(ctx, id); err != nil {
		response.SendResponse(c, http.StatusInternalServerError, "error", "failed to delete user", nil)
		return
	}

	response.SendResponse(c, http.StatusOK, "success", "user deleted successfully", nil)
}
