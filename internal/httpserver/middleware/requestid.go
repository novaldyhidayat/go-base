package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const requestIDKey = "X-Request-ID"

// RequestID injects a request ID header into each incoming request.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		reqID := c.Request.Header.Get(requestIDKey)
		if reqID == "" {
			reqID = uuid.NewString()
		}
		c.Set(requestIDKey, reqID)
		c.Writer.Header().Set(requestIDKey, reqID)
		c.Next()
	}
}
