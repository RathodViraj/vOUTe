package middleware

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
	"voute/pkg/mailing"
	"voute/pkg/response"
	"voute/pkg/utils"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

var db *MiddleWareDB

func AddDBInMiddleware(mongoDB *mongo.Database, name string) {
	db = &MiddleWareDB{
		mongoDatabase:     mongoDB,
		userCollectioName: name,
	}
}

func AddMailingServiceInMiddleware(mailService mailing.EmailService) {
	if db != nil {
		db.emailService = mailService
	}
}

type Claims struct {
	UserID   string `json:"user_id"`
	UserName string `json:"user_name"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

type LoginWithEmailRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type LoginWithUsernameRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type SignupWithOTPRequest struct {
	Username          string `json:"username" binding:"required"`
	Email             string `json:"email" binding:"required,email"`
	Password          string `json:"password" binding:"required,min=6"`
	VerificationToken string `json:"verification_token" binding:"required"`
}

type LoginWithOTPRequest struct {
	Email string `json:"email" binding:"required,email"`
	OTP   string `json:"otp" binding:"required"`
}

type LoginWithOTPUsernameRequest struct {
	Username string `json:"username" binding:"required"`
	OTP      string `json:"otp" binding:"required"`
}

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefershToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}

var (
	accessSecret  = []byte(utils.GetEnv("JWT_ACCESS_SECRET", "give-me-access"))
	refreshSecret = []byte(utils.GetEnv("JWT_REFRESH_SECRET", "refresh-my-token"))

	accessTTL  = 2 * 24 * time.Hour
	refershTTL = 7 * 24 * time.Hour
)

func useSecureCookies() bool {
	return strings.EqualFold(utils.GetEnv("COOKIE_SECURE", ""), "true")
}

func generateTokePair(userID, userName, role string) (*TokenPair, error) {
	now := time.Now()

	accessClaims := &Claims{
		UserID:   userID,
		UserName: userName,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(accessTTL)),
			Issuer:    "backend",
		},
	}
	accessToken, err := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		accessClaims,
	).SignedString(accessSecret)
	if err != nil {
		return nil, err
	}

	refershClaims := &Claims{
		UserID:   userID,
		UserName: userName,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(refershTTL)),
			Issuer:    "api-gateway",
		},
	}
	refershToken, err := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		refershClaims,
	).SignedString(refreshSecret)
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefershToken: refershToken,
		ExpiresIn:    int64(accessTTL.Seconds()),
	}, nil
}

func ParseAccessToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(
		tokenStr,
		&Claims{},
		func(t *jwt.Token) (any, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return accessSecret, nil
		},
	)
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}
	return nil, jwt.ErrInvalidKey
}

func ParseRefershToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(
		tokenStr,
		&Claims{},
		func(t *jwt.Token) (any, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return refreshSecret, nil
		},
	)
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}
	return nil, jwt.ErrInvalidKey
}

func Login(c *gin.Context) {
	ctx, cancle := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancle()

	var pair *TokenPair
	var err error
	if c.Query("type") == "username" {
		var req LoginWithUsernameRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.SendResponse(c, http.StatusBadRequest, "error", "invalid request body", nil)
			return
		}
		pair, err = LoginWithUsername(ctx, req)

	} else if c.Query("type") == "email" {
		var req LoginWithEmailRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.SendResponse(c, http.StatusBadRequest, "error", "invalid request body", nil)
			return
		}
		pair, err = LoginWithEmail(ctx, req)
	} else {
		response.SendResponse(c, http.StatusBadRequest, "error", "invalid login request type", nil)
		return
	}

	if err != nil {
		if err.Error() == "invalid credntials" {
			response.SendResponse(c, http.StatusUnauthorized, "error", "invalid credntials", nil)
			return
		}
		response.SendResponse(c, http.StatusInternalServerError, "error", "please try again later", nil)
		return
	}

	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("refresh_token", pair.RefershToken, int(refershTTL.Seconds()), "/", "", useSecureCookies(), true)
	response.SendResponse(c, http.StatusOK, "success", "loged in successfully", pair)
}

func LoginWithUsername(ctx context.Context, req LoginWithUsernameRequest) (*TokenPair, error) {
	userID, hashPwd, role, err := db.FetchUserByUsername(ctx, req.Username)
	if err != nil {
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hashPwd), []byte(req.Password)); err != nil {
		return nil, errors.New("invalid credntials")
	}

	pair, err := generateTokePair(strconv.FormatInt(userID, 10), req.Username, role)
	if err != nil {
		return nil, errors.New("could not generate tokens")
	}

	return pair, nil
}

func LoginWithEmail(ctx context.Context, req LoginWithEmailRequest) (*TokenPair, error) {
	userID, username, hashPwd, role, err := db.FetchUserByEmail(ctx, req.Email)
	if err != nil {
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hashPwd), []byte(req.Password)); err != nil {
		return nil, errors.New("invalid credntials")
	}

	pair, err := generateTokePair(strconv.FormatInt(userID, 10), username, role)
	if err != nil {
		return nil, errors.New("could not generate tokens")
	}

	return pair, nil
}

func RefershToken(c *gin.Context) {
	tokenStr, err := c.Cookie("refresh_token")
	if err != nil {
		tokenStr, _ = c.Cookie("refersh_token")
	}
	if err != nil {
		tokenStr = extractBearer(c)
	}
	if tokenStr == "" {
		response.SendResponse(c, http.StatusUnauthorized, "error", "refersh token required", nil)
		return
	}

	claims, err := ParseRefershToken(tokenStr)
	if err != nil {
		response.SendResponse(c, http.StatusUnauthorized, "error", "invalid or expried refersh token", nil)
		return
	}

	// TODO: optionally check a token blacklist / rotation store here

	pair, err := generateTokePair(claims.UserID, claims.UserName, claims.Role)
	if err != nil {
		response.SendResponse(c, http.StatusInternalServerError, "error", "could not genrate tokens", nil)
		return
	}

	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("refresh_token", pair.RefershToken, int(refershTTL.Seconds()), "/", "", useSecureCookies(), true)
	response.SendResponse(c, http.StatusOK, "success", "token refreshed", pair)
}

func extractBearer(c *gin.Context) string {
	header := c.GetHeader("Authorization")
	if header == "" {
		return ""
	}

	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

func Logout(c *gin.Context) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("refresh_token", "", -1, "/", "", useSecureCookies(), true)
	c.SetCookie("refersh_token", "", -1, "/", "", useSecureCookies(), true)
	response.SendResponse(c, http.StatusOK, "success", "logged out successfully", nil)
}

func ResetPassword(c *gin.Context) {
	var req struct {
		Email       string `json:"email" binding:"required,email"`
		NewPassword string `json:"new_password" binding:"required,min=6"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.SendResponse(c, http.StatusBadRequest, "error", "invalid request body", nil)
		return
	}

	if err := db.ResetPassword(c.Request.Context(), req.Email, req.NewPassword); err != nil {
		response.SendResponse(c, http.StatusInternalServerError, "error", "failed to reset password", nil)
		return
	}

	response.SendResponse(c, http.StatusOK, "success", "password reset successfully", nil)
}

func getGoogleOAuthConfig() (*oauth2.Config, error) {
	clientID := strings.TrimSpace(utils.GetEnv("GOOGLE_CLIENT_ID", ""))
	if clientID == "" {
		clientID = strings.TrimSpace(utils.GetEnv("OAUTH_CLIENT_ID", ""))
	}
	clientSecret := strings.TrimSpace(utils.GetEnv("GOOGLE_CLIENT_SECRET", ""))
	if clientSecret == "" {
		clientSecret = strings.TrimSpace(utils.GetEnv("OAUTH_CLIENT_SECRET", ""))
	}
	redirectURL := strings.TrimSpace(utils.GetEnv("GOOGLE_REDIRECT_URL", "http://localhost:8080/auth/google/callback"))

	if clientID == "" || clientSecret == "" {
		return nil, errors.New("google oauth is not configured")
	}

	return &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes:       []string{"openid", "email", "profile"},
		Endpoint:     google.Endpoint,
	}, nil
}

func generateRandomState() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func sanitizeUsername(input string) string {
	re := regexp.MustCompile(`[^a-zA-Z0-9_]`)
	trimmed := re.ReplaceAllString(strings.ToLower(strings.TrimSpace(input)), "")
	if trimmed == "" {
		return "voter"
	}
	if len(trimmed) < 3 {
		return trimmed + "_user"
	}
	return trimmed
}

func generateRandomPassword() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func getOrCreateGoogleUser(ctx context.Context, email, name string) (int64, string, string, error) {
	userID, username, _, role, err := db.FetchUserByEmail(ctx, email)
	if err == nil {
		return userID, username, role, nil
	}
	if err.Error() != "invalid credntials" {
		return 0, "", "", err
	}

	base := strings.Split(email, "@")[0]
	if strings.TrimSpace(name) != "" {
		base = strings.ReplaceAll(name, " ", "_")
	}
	base = sanitizeUsername(base)

	password, err := generateRandomPassword()
	if err != nil {
		return 0, "", "", err
	}

	for i := 0; i < 50; i++ {
		candidate := base
		if i > 0 {
			candidate = fmt.Sprintf("%s%d", base, i)
		}

		createdID, createErr := db.CreateUser(ctx, candidate, email, password)
		if createErr == nil {
			return createdID, candidate, "user", nil
		}

		if strings.Contains(createErr.Error(), "email already exists") {
			uid, uname, _, r, fetchErr := db.FetchUserByEmail(ctx, email)
			if fetchErr == nil {
				return uid, uname, r, nil
			}
		}

		if !strings.Contains(createErr.Error(), "username already exists") {
			return 0, "", "", createErr
		}
	}

	return 0, "", "", errors.New("failed to allocate username for google account")
}

func GoogleLogin(c *gin.Context) {
	conf, err := getGoogleOAuthConfig()
	if err != nil {
		response.SendResponse(c, http.StatusInternalServerError, "error", err.Error(), nil)
		return
	}

	state, err := generateRandomState()
	if err != nil {
		response.SendResponse(c, http.StatusInternalServerError, "error", "failed to start google auth", nil)
		return
	}

	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("google_oauth_state", state, 300, "/", "", useSecureCookies(), true)
	c.Redirect(http.StatusTemporaryRedirect, conf.AuthCodeURL(state))
}

func GoogleCallback(c *gin.Context) {
	state := c.Query("state")
	code := c.Query("code")
	if state == "" || code == "" {
		response.SendResponse(c, http.StatusBadRequest, "error", "missing google callback params", nil)
		return
	}

	savedState, err := c.Cookie("google_oauth_state")
	if err != nil || savedState == "" || savedState != state {
		response.SendResponse(c, http.StatusUnauthorized, "error", "invalid oauth state", nil)
		return
	}

	conf, err := getGoogleOAuthConfig()
	if err != nil {
		response.SendResponse(c, http.StatusInternalServerError, "error", err.Error(), nil)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	tok, err := conf.Exchange(ctx, code)
	if err != nil {
		response.SendResponse(c, http.StatusUnauthorized, "error", "failed to exchange google code", nil)
		return
	}

	client := conf.Client(ctx, tok)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v3/userinfo")
	if err != nil {
		response.SendResponse(c, http.StatusUnauthorized, "error", "failed to fetch google profile", nil)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		response.SendResponse(c, http.StatusUnauthorized, "error", "google profile request failed", nil)
		return
	}

	var profile struct {
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
		Name          string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&profile); err != nil {
		response.SendResponse(c, http.StatusUnauthorized, "error", "failed to parse google profile", nil)
		return
	}

	if profile.Email == "" || !profile.EmailVerified {
		response.SendResponse(c, http.StatusUnauthorized, "error", "google email is not verified", nil)
		return
	}

	userID, username, role, err := getOrCreateGoogleUser(ctx, profile.Email, profile.Name)
	if err != nil {
		response.SendResponse(c, http.StatusInternalServerError, "error", "failed to login with google", nil)
		return
	}

	pair, err := generateTokePair(strconv.FormatInt(userID, 10), username, role)
	if err != nil {
		response.SendResponse(c, http.StatusInternalServerError, "error", "could not generate tokens", nil)
		return
	}

	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("refresh_token", pair.RefershToken, int(refershTTL.Seconds()), "/", "", useSecureCookies(), true)

	frontendURL := strings.TrimRight(utils.GetEnv("FRONTEND_URL", "http://localhost:5173"), "/")
	redirectURL := frontendURL + "/auth/google/callback?access_token=" + url.QueryEscape(pair.AccessToken)
	c.Redirect(http.StatusTemporaryRedirect, redirectURL)
}

func SignupWithOTP(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var req SignupWithOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.SendResponse(c, http.StatusBadRequest, "error", "invalid request body", nil)
		return
	}

	if db.emailService == nil {
		response.SendResponse(c, http.StatusInternalServerError, "error", "email service not configured", nil)
		return
	}

	email, err := db.emailService.GetEmailByVerificationToken(ctx, req.VerificationToken)
	if err != nil {
		response.SendResponse(c, http.StatusInternalServerError, "error", "failed to validate verification token: "+err.Error(), nil)
		return
	}
	if email == "" || email != req.Email {
		response.SendResponse(c, http.StatusUnauthorized, "error", "invalid or expired verification token", nil)
		return
	}

	userID, err := db.CreateUser(ctx, req.Username, req.Email, req.Password)
	if err != nil {
		response.SendResponse(c, http.StatusInternalServerError, "error", err.Error(), nil)
		return
	}

	pair, err := generateTokePair(strconv.FormatInt(userID, 10), req.Username, "user")
	if err != nil {
		response.SendResponse(c, http.StatusInternalServerError, "error", "could not generate tokens", nil)
		return
	}

	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("refresh_token", pair.RefershToken, int(refershTTL.Seconds()), "/", "", useSecureCookies(), true)
	response.SendResponse(c, http.StatusOK, "success", "signup successful", pair)
}

func LoginWithOTP(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var req LoginWithOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.SendResponse(c, http.StatusBadRequest, "error", "invalid request body", nil)
		return
	}

	if db.emailService == nil {
		response.SendResponse(c, http.StatusInternalServerError, "error", "email service not configured", nil)
		return
	}

	isValid, err := db.emailService.VerifyOTP(ctx, req.Email, req.OTP)
	if err != nil {
		response.SendResponse(c, http.StatusInternalServerError, "error", "failed to verify OTP: "+err.Error(), nil)
		return
	}

	if !isValid {
		response.SendResponse(c, http.StatusUnauthorized, "error", "invalid or expired OTP", nil)
		return
	}

	userID, username, _, role, err := db.FetchUserByEmail(ctx, req.Email)
	if err != nil {
		response.SendResponse(c, http.StatusUnauthorized, "error", "user not found", nil)
		return
	}

	pair, err := generateTokePair(strconv.FormatInt(userID, 10), username, role)
	if err != nil {
		response.SendResponse(c, http.StatusInternalServerError, "error", "could not generate tokens", nil)
		return
	}

	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("refresh_token", pair.RefershToken, int(refershTTL.Seconds()), "/", "", useSecureCookies(), true)
	response.SendResponse(c, http.StatusOK, "success", "login successful", pair)
}

func LoginWithOTPUsername(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var req LoginWithOTPUsernameRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.SendResponse(c, http.StatusBadRequest, "error", "invalid request body", nil)
		return
	}

	if db.emailService == nil {
		response.SendResponse(c, http.StatusInternalServerError, "error", "email service not configured", nil)
		return
	}

	email, err := db.GetEmailByUsername(ctx, req.Username)
	if err != nil {
		response.SendResponse(c, http.StatusUnauthorized, "error", "user not found", nil)
		return
	}

	isValid, err := db.emailService.VerifyOTP(ctx, email, req.OTP)
	if err != nil {
		response.SendResponse(c, http.StatusInternalServerError, "error", "failed to verify OTP: "+err.Error(), nil)
		return
	}

	if !isValid {
		response.SendResponse(c, http.StatusUnauthorized, "error", "invalid or expired OTP", nil)
		return
	}

	userID, _, role, err := db.FetchUserByUsername(ctx, req.Username)
	if err != nil {
		response.SendResponse(c, http.StatusUnauthorized, "error", "user not found", nil)
		return
	}

	pair, err := generateTokePair(strconv.FormatInt(userID, 10), req.Username, role)
	if err != nil {
		response.SendResponse(c, http.StatusInternalServerError, "error", "could not generate tokens", nil)
		return
	}

	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("refresh_token", pair.RefershToken, int(refershTTL.Seconds()), "/", "", useSecureCookies(), true)
	response.SendResponse(c, http.StatusOK, "success", "login successful", pair)
}
