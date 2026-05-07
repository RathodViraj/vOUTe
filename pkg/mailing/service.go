package mailing

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"time"
	"voute/pkg/utils"

	"github.com/wneessen/go-mail"
)

const expiredTimeOUT = 2 * time.Minute

type EmailService interface {
	SendOTPEmail(ctx context.Context, to string) error
	SendOTPEmailByUsername(ctx context.Context, username string) error
	VerifyOTP(ctx context.Context, email, otp string) (bool, error)
	CreateVerificationToken(ctx context.Context, email string) (string, error)
	GetEmailByVerificationToken(ctx context.Context, token string) (string, error)
}

type emailService struct {
	repo        MailingRepository
	smtpHost    string
	smtpPort    int
	smtpUser    string
	smtpPass    string
	senderEmail string
	senderName  string
}

func NewEmailService(repo MailingRepository) EmailService {
	smtpHost := utils.GetEnvOrPanic("SMTP_HOST")
	smtpPort := utils.GetEnvOrPanicInt("SMTP_PORT")
	smtpUser := utils.GetEnvOrPanic("SMTP_USER")
	smtpPass := utils.GetEnvOrPanic("SMTP_PASSWORD")
	senderEmail := utils.GetEnvOrPanic("SMTP_FROM")
	senderName := utils.GetEnv("SMTP_FROM_NAME", "VOuTE")
	return &emailService{
		repo:        repo,
		smtpHost:    smtpHost,
		smtpPort:    smtpPort,
		smtpUser:    smtpUser,
		smtpPass:    smtpPass,
		senderEmail: senderEmail,
		senderName:  senderName,
	}
}

func (s *emailService) SendOTPEmail(ctx context.Context, to string) error {
	code := generateOTP(6)
	if err := s.repo.StoreOTP(ctx, to, code); err != nil {
		return err
	}

	htmlContent := generateOTPEmailContent(code)
	plainText := generateOTPPlainText(code)

	return s.sendEmailViaSMTP(to, "Your VOuTE OTP Code", plainText, htmlContent)
}

func (s *emailService) SendOTPEmailByUsername(ctx context.Context, username string) error {
	email, err := s.repo.GetEmailByUsername(ctx, username)
	if err != nil {
		return err
	}

	code := generateOTP(6)
	if err := s.repo.StoreOTP(ctx, email, code); err != nil {
		return err
	}

	htmlContent := generateOTPEmailContent(code)
	plainText := generateOTPPlainText(code)

	return s.sendEmailViaSMTP(email, "Your VOuTE OTP Code", plainText, htmlContent)
}

func (s *emailService) CreateVerificationToken(ctx context.Context, email string) (string, error) {
	token := generateVerificationToken(32)
	// store token in redis with TTL (10 minutes)
	if err := s.repo.StoreVerificationToken(ctx, token, email, 10*time.Minute); err != nil {
		return "", err
	}
	return token, nil
}

func (s *emailService) GetEmailByVerificationToken(ctx context.Context, token string) (string, error) {
	return s.repo.GetEmailByVerificationToken(ctx, token)
}

func generateOTP(length int) string {
	digits := "0123456789"
	otp := make([]byte, length)
	for i := range otp {
		otp[i] = digits[rand.Intn(len(digits))]
	}
	return string(otp)
}

func generateOTPPlainText(code string) string {
	return "Your VOuTE OTP code is: " + code + "\n\nThis code will expire in 2 minutes.\nIf you didn't request this code, please ignore this email."
}

func generateOTPEmailContent(code string) string {
	return `
	<!DOCTYPE html>
	<html>
	<head>
		<meta charset="utf-8">
	</head>
	<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333; margin: 0; padding: 0;">
		<div style="max-width: 600px; margin: 0 auto; padding: 20px;">
			<div style="background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); color: white; padding: 30px; border-radius: 8px 8px 0 0; text-align: center;">
				<h1 style="margin: 0; font-size: 28px;">🗳️ VOuTE</h1>
			</div>
			<div style="background: #f9f9f9; padding: 30px; border-radius: 0 0 8px 8px;">
				<h2 style="margin: 0 0 15px 0; font-size: 20px; color: #333;">Your One-Time Password (OTP)</h2>
				<p style="margin: 0 0 10px 0; color: #333;">Hello,</p>
				<p style="margin: 0 0 20px 0; color: #333;">You've requested a one-time password to complete your authentication. Use the code below:</p>
				<div style="background: white; border: 2px dashed #667eea; padding: 20px; border-radius: 8px; text-align: center; margin: 20px 0;">
					<div style="font-size: 36px; font-weight: bold; color: #667eea; letter-spacing: 5px; font-family: 'Courier New', monospace;">` + code + `</div>
				</div>
				<p style="margin: 0 0 10px 0; color: #666;">This code will expire in <strong>2 minutes</strong>.</p>
				<p style="margin: 0; color: #999; font-size: 14px;">If you didn't request this code, please ignore this email and do not share it with anyone.</p>
			</div>
			<div style="text-align: center; color: #999; font-size: 12px; margin-top: 20px; padding-top: 20px; border-top: 1px solid #ddd;">
				<p style="margin: 0;">&copy; 2026 VOuTE. All rights reserved.</p>
			</div>
		</div>
	</body>
	</html>
	`
}

func generateVerificationToken(length int) string {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return string(b)
}

func (s *emailService) VerifyOTP(ctx context.Context, email, otp string) (bool, error) {
	return s.repo.VerifyOTP(ctx, email, otp)
}

func (s *emailService) sendEmailViaSMTP(to, subject, plainText, htmlText string) error {
	timeout := smtpTimeout()

	msg := mail.NewMsg()
	if err := msg.From(s.senderEmail); err != nil {
		fmt.Printf("[mailing] Failed to set From: %v\n", err)
		return err
	}
	if err := msg.To(to); err != nil {
		fmt.Printf("[mailing] Failed to set To: %v\n", err)
		return err
	}
	msg.Subject(subject)
	// Set HTML as the primary body (not alternative) to ensure it renders properly
	msg.SetBodyString(mail.TypeTextHTML, htmlText)

	options := []mail.Option{
		mail.WithPort(s.smtpPort),
		mail.WithSMTPAuth(mail.SMTPAuthPlain),
		mail.WithUsername(s.smtpUser),
		mail.WithPassword(s.smtpPass),
		mail.WithSSL(),
	}
	if timeout > 0 {
		options = append(options, mail.WithTimeout(timeout))
	}

	client, err := mail.NewClient(s.smtpHost, options...)
	if err != nil {
		fmt.Printf("[mailing] SMTP client creation failed: %v\n", err)
		return errors.New("failed to create SMTP client: " + err.Error())
	}

	if err := client.DialAndSend(msg); err != nil {
		// If port 465 fails, try fallback to port 587 with TLS before
		// reporting an error to avoid noisy logs on transient SSL failures.
		if s.smtpPort == 465 {
			fmt.Printf("[mailing] Primary SMTP route failed, trying TLS fallback on 587...\n")
			fallbackOptions := []mail.Option{
				mail.WithPort(587),
				mail.WithSMTPAuth(mail.SMTPAuthPlain),
				mail.WithUsername(s.smtpUser),
				mail.WithPassword(s.smtpPass),
				mail.WithTLSPolicy(mail.TLSMandatory),
			}
			if timeout > 0 {
				fallbackOptions = append(fallbackOptions, mail.WithTimeout(timeout))
			}

			client2, err2 := mail.NewClient(s.smtpHost, fallbackOptions...)
			if err2 == nil {
				if err3 := client2.DialAndSend(msg); err3 == nil {
					fmt.Printf("[mailing] Email sent successfully to %s (via TLS fallback)\n", to)
					return nil
				}
				return errors.New("failed to send email via SMTP: primary route failed and fallback failed")
			}
			return errors.New("failed to create SMTP fallback client: " + err2.Error())
		}
		return errors.New("failed to send email via SMTP: " + err.Error())
	}

	fmt.Printf("[mailing] Email sent successfully to %s\n", to)
	return nil
}

func smtpTimeout() time.Duration {
	value := utils.GetEnv("SMTP_TIMEOUT_SECONDS", "0")
	seconds, err := strconv.Atoi(value)
	if err != nil || seconds <= 0 {
		return 0
	}
	return time.Duration(seconds) * time.Second
}
