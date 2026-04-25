package middleware

import (
	"net/http"
	"voute/pkg/response"

	"github.com/gin-gonic/gin"
)

const claimsKey = "jwt_claims"

func GetClaims(c *gin.Context) (*Claims, bool) {
	v, ok := c.Get(claimsKey)
	if !ok {
		return nil, false
	}
	claims, ok := v.(*Claims)
	return claims, ok
}

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenStr := extractBearer(c)
		if tokenStr == "" {
			response.SendResponse(c, http.StatusUnauthorized, "error", "access token required", nil)
			c.Abort()
			return
		}

		claims, err := ParseAccessToken(tokenStr)
		if err != nil {
			response.SendResponse(c, http.StatusUnauthorized, "error", "invalid or expired access token", nil)
			c.Abort()
			return
		}

		c.Set(claimsKey, claims)
		c.Next()
	}
}
