package middleware

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"
	"voute/pkg/response"
	"voute/pkg/utils"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"
)

var db *MiddleWareDB

func AddDBInMiddleware(mongoDB *mongo.Database, name string) {
	db = &MiddleWareDB{
		mongoDatabase:     mongoDB,
		userCollectioName: name,
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

	c.SetCookie("refresh_token", pair.RefershToken, int(refershTTL.Seconds()), "/", "", true, true)
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

	c.SetCookie("refresh_token", pair.RefershToken, int(refershTTL.Seconds()), "/", "", true, true)
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
	c.SetCookie("refresh_token", "", -1, "/", "", true, true)
	c.SetCookie("refersh_token", "", -1, "/", "", true, true)
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
