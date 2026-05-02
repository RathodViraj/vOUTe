package mailing

type GetOTPRequest struct {
	Email    string `json:"email" validate:"email" binding:"omitempty,email"`
	Username string `json:"username" validate:"omitempty"`
}

type VerifyOTPRequest struct {
	Email string `json:"email" validate:"required,email"`
	OTP   string `json:"otp" validate:"required"`
}

type SendEmailRequest struct {
	To      string `json:"to" validate:"required,email"`
	Subject string `json:"subject" validate:"required"`
	Body    string `json:"body" validate:"required"`
}

type StoredOTP struct {
	Email     string
	OTP       string
	ExpiresAt int64
}
