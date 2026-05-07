package mailing

import (
	"context"
	"net/http"
	"time"
	"voute/pkg/response"

	"github.com/gin-gonic/gin"
)

type MailingHandler interface {
	GetOTP(c *gin.Context)
	VerifyOTP(c *gin.Context)
	RegisterRoutes(router *gin.Engine)
}

type mailingHandler struct {
	service EmailService
}

func NewMailingHandler(service EmailService) MailingHandler {
	return &mailingHandler{service: service}
}

func (h *mailingHandler) RegisterRoutes(router *gin.Engine) {
	mailingGroup := router.Group("/mailing")
	{
		mailingGroup.POST("/otp", h.GetOTP)
		mailingGroup.POST("/verify-otp", h.VerifyOTP)
	}
}

func (h *mailingHandler) GetOTP(c *gin.Context) {
	// No timeout here: for demo reliability we let SMTP complete naturally.
	ctx := context.Background()

	var req GetOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.SendResponse(c, http.StatusBadRequest, "error", "invalid request body - provide either email or username", nil)
		return
	}

	// Validate that either email or username is provided
	if req.Email == "" && req.Username == "" {
		response.SendResponse(c, http.StatusBadRequest, "error", "either email or username is required", nil)
		return
	}

	var err error
	if req.Email != "" {
		err = h.service.SendOTPEmail(ctx, req.Email)
	} else {
		err = h.service.SendOTPEmailByUsername(ctx, req.Username)
	}

	if err != nil {
		response.SendResponse(c, http.StatusInternalServerError, "error", "failed to send OTP email: "+err.Error(), nil)
		return
	}
	response.SendResponse(c, http.StatusOK, "success", "OTP email sent successfully", nil)
}

func (h *mailingHandler) VerifyOTP(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	var req VerifyOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.SendResponse(c, http.StatusBadRequest, "error", "invalid request", nil)
		return
	}

	isValid, err := h.service.VerifyOTP(ctx, req.Email, req.OTP)
	if err != nil {
		response.SendResponse(c, http.StatusInternalServerError, "error", "failed to verify OTP: "+err.Error(), nil)
		return
	}
	if !isValid {
		response.SendResponse(c, http.StatusUnauthorized, "error", "invalid OTP", nil)
		return
	}
	// create a short-lived verification token and return it to the client
	token, err := h.service.CreateVerificationToken(ctx, req.Email)
	if err != nil {
		response.SendResponse(c, http.StatusInternalServerError, "error", "failed to create verification token: "+err.Error(), nil)
		return
	}
	response.SendResponse(c, http.StatusOK, "success", "OTP verified successfully", map[string]string{"verification_token": token})
}
