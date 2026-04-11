package user

import (
	"context"
	"net/http"
	"time"
	"voute/pkg/response"

	"github.com/gin-gonic/gin"
)

type UserHandler interface {
	AddUserRoute(r *gin.Engine)
}

type userHandler struct {
	userService UserService
}

func newHandler(srv UserService) UserHandler {
	return &userHandler{
		userService: srv,
	}
}

func (h *userHandler) AddUserRoute(r *gin.Engine) {
	r.POST("/user/create", h.createUser)
	r.GET("/user/:id", h.getUserByID)
	r.GET("/user/:email", h.getUserByEmail)
	r.GET("/users", h.getUsersByUsername)
	r.PUT("/user/update", h.updateUser)
	r.PUT("/user/update-password", h.updatePassword)
	r.DELETE("/user/delete", h.deleteUser)
}

func (h *userHandler) createUser(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.SendResponse(c, http.StatusBadRequest, "error", "invalid data", nil)
		return
	}

	user, err := h.userService.CreateUser(ctx, req.Username, req.Email, req.Password, req.Role)
	if err != nil {
		response.SendResponse(c, http.StatusInternalServerError, "error", "failed to create user", nil)
		return
	}

	response.SendResponse(c, http.StatusOK, "success", "user created successfully", user)
}

func (h *userHandler) getUserByEmail(c *gin.Context) {
	email := c.Param("email")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	user, err := h.userService.GetUserByEmail(ctx, email)
	if err != nil {
		response.SendResponse(c, http.StatusInternalServerError, "error", "failed to get user", nil)
		return
	}

	if user == nil {
		response.SendResponse(c, http.StatusNotFound, "error", "user not found", nil)
	}

	response.SendResponse(c, http.StatusOK, "success", "user found", user)
}

func (h *userHandler) getUserByID(c *gin.Context) {
	id := c.Param("id")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	user, err := h.userService.GetUserByID(ctx, id)
	if err != nil {
		response.SendResponse(c, http.StatusInternalServerError, "error", "failed to get user", nil)
		return
	}

	if user == nil {
		response.SendResponse(c, http.StatusNotFound, "error", "user not found", nil)
	}

	response.SendResponse(c, http.StatusOK, "success", "user found", user)
}

func (h *userHandler) getUsersByUsername(c *gin.Context) {
	username := c.Query("username")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	users, err := h.userService.GetUsersByUsername(ctx, username, 0, 10)
	if err != nil {
		response.SendResponse(c, http.StatusInternalServerError, "error", "failed to get users", nil)
		return
	}

	response.SendResponse(c, http.StatusOK, "success", "users found", users)
}

func (h *userHandler) updateUser(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.SendResponse(c, http.StatusBadRequest, "error", "invalid data", nil)
		return
	}

	if err := h.userService.UpdateUser(ctx, req.Username, req.Email, req.ID); err != nil {
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
	id := c.Query("id")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := h.userService.DeleteUser(ctx, id); err != nil {
		response.SendResponse(c, http.StatusInternalServerError, "error", "failed to delete user", nil)
		return
	}

	response.SendResponse(c, http.StatusOK, "success", "user deleted successfully", nil)
}
