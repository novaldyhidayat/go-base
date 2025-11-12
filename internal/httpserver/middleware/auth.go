package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"

	"go-base/internal/security"
	"go-base/pkg/response"
)

const claimsContextKey = "jwtClaims"

// Auth validates JWT tokens and injects claims into the request context.
func Auth(jwtManager security.JWTManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			status, payload := response.Fail("unauthorized", "Missing Authorization header", nil)
			c.AbortWithStatusJSON(status, payload)
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			status, payload := response.Fail("unauthorized", "Authorization header must be Bearer token", nil)
			c.AbortWithStatusJSON(status, payload)
			return
		}

		claims, err := jwtManager.Verify(parts[1])
		if err != nil {
			status, payload := response.Fail("unauthorized", "Invalid or expired token", err.Error())
			c.AbortWithStatusJSON(status, payload)
			return
		}

		c.Set(claimsContextKey, claims)
		c.Next()
	}
}

// ClaimsFromContext retrieves JWT claims from request context.
func ClaimsFromContext(c *gin.Context) (*security.Claims, bool) {
	claims, ok := c.Get(claimsContextKey)
	if !ok {
		return nil, false
	}

	claimPtr, ok := claims.(*security.Claims)
	return claimPtr, ok
}
