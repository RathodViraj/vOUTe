package response

import (
	"time"

	"github.com/gin-gonic/gin"
)

type Response struct {
	Type      string `json:"type"`
	Status    int    `json:"status"`
	Message   string `json:"message"`
	Data      any    `json:"data"`
	CreatedAt int64  `json:"created_at"`
}

func SendResponse(c *gin.Context, status int, t, message string, data any) {
	res := Response{
		Type:      t,
		Status:    status,
		Message:   message,
		Data:      data,
		CreatedAt: time.Now().Unix(),
	}
	c.JSON(res.Status, res)
}
