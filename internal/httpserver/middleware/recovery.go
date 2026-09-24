package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"go-base/pkg/response"
)

// Recovery handles panics and returns a 500 error while logging.
func Recovery(log *zap.Logger) gin.HandlerFunc {
	if log == nil {
		log = zap.NewNop()
	}

	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		log.Error("panic recovered", zap.Any("error", recovered))
		status, payload := response.WithStatus(http.StatusInternalServerError, "internal_error", "Internal server error", nil)
		c.AbortWithStatusJSON(status, payload)
	})
}
