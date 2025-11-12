package middleware

import (
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"go-base/internal/config"
)

// CORS returns a middleware configured from the security settings.
func CORS(cfg config.SecurityConfig) gin.HandlerFunc {
	return cors.New(cors.Config{
		AllowOrigins:     cfg.AllowedOrigins,
		AllowMethods:     cfg.AllowedMethods,
		AllowHeaders:     cfg.AllowedHeaders,
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	})
}
