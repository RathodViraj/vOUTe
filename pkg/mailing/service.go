package mailing

import (
	"context"
	"errors"
	"math/rand"
	"time"
	"voute/pkg/utils"

	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

const expiredTimeOUT = 2 * time.Minute

type EmailService interface {
	SendOTPEmail(ctx context.Context, to string) error
	VerifyOTP(ctx context.Context, email, otp string) (bool, error)
}

type emailService struct {
	repo        MailingRepository
	client      *sendgrid.Client
	senderEmail string
	senderName  string
}

func NewEmailService(repo MailingRepository) EmailService {
	apiKey := utils.GetEnvOrPanic("SENDGRID_API_KEY")
	senderEmail := utils.GetEnvOrPanic("SENDGRID_SENDER_EMAIL")
	senderName := utils.GetEnvOrPanic("SENDGRID_SENDER_NAME")
	return &emailService{
		repo:        repo,
		client:      sendgrid.NewSendClient(apiKey),
		senderEmail: senderEmail,
		senderName:  senderName,
	}
}

func (s *emailService) SendOTPEmail(ctx context.Context, to string) error {
	isUserExist, err := s.repo.IsUserExist(ctx, to)
	if err != nil {
		return err
	}
	if !isUserExist {
		return errors.New("user does not exist")
	}

	code := generateOTP(6)
	if err := s.repo.StoreOTP(ctx, to, code); err != nil {
		return err
	}

	emailContent := generateOTPEmailContent(code)
	from := mail.NewEmail(s.senderName, s.senderEmail)
	toEmail := mail.NewEmail("", to)
	message := mail.NewSingleEmail(from, "Your OTP Code", toEmail, emailContent, emailContent)
	_, err = s.client.Send(message)
	return err
}

func generateOTP(length int) string {
	digits := "0123456789"
	otp := make([]byte, length)
	for i := range otp {
		otp[i] = digits[rand.Intn(len(digits))]
	}
	return string(otp)
}

func generateOTPEmailContent(code string) string {
	return "Your OTP code is: " + code
}

func (s *emailService) VerifyOTP(ctx context.Context, email, otp string) (bool, error) {
	return s.repo.VerifyOTP(ctx, email, otp)
}
