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
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	var req GetOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.SendResponse(c, http.StatusBadRequest, "success", "invalid request", nil)
		return
	}

	if err := h.service.SendOTPEmail(ctx, req.Email); err != nil {
		response.SendResponse(c, http.StatusInternalServerError, "error", "failed to send OTP email", nil)
		return
	}
	response.SendResponse(c, http.StatusOK, "success", "OTP email sent successfully", nil)
}

func (h *mailingHandler) VerifyOTP(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	var req VerifyOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.SendResponse(c, http.StatusBadRequest, "error", "invalid request", nil)
		return
	}

	isValid, err := h.service.VerifyOTP(ctx, req.Email, req.OTP)
	if err != nil {
		response.SendResponse(c, http.StatusInternalServerError, "error", "failed to verify OTP", nil)
		return
	}
	if !isValid {
		response.SendResponse(c, http.StatusUnauthorized, "error", "invalid OTP", nil)
		return
	}
	response.SendResponse(c, http.StatusOK, "success", "OTP verified successfully", nil)
}
