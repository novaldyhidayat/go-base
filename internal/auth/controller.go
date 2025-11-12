package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"go-base/pkg/response"
)

// Controller handles HTTP endpoints for authentication.
type Controller struct {
	service Service
}

// NewController builds a new Controller.
func NewController(service Service) *Controller {
	return &Controller{service: service}
}

// Register handles user registration requests.
func (c *Controller) Register(ctx *gin.Context) {
	var req RegisterRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		status, payload := response.Fail("invalid_request", "Invalid payload", err.Error())
		ctx.JSON(status, payload)
		return
	}

	user, err := c.service.Register(ctx.Request.Context(), req)
	if err != nil {
		status, payload := response.Fail("register_failed", "Could not register user", err.Error())
		ctx.JSON(status, payload)
		return
	}

	ctx.JSON(http.StatusCreated, response.JSON(gin.H{"id": user.ID, "email": user.Email}, nil))
}

// Login handles user login requests.
func (c *Controller) Login(ctx *gin.Context) {
	var req LoginRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		status, payload := response.Fail("invalid_request", "Invalid payload", err.Error())
		ctx.JSON(status, payload)
		return
	}

	token, err := c.service.Login(ctx.Request.Context(), req)
	if err != nil {
		status, payload := response.Fail("login_failed", "Invalid email or password", err.Error())
		ctx.JSON(status, payload)
		return
	}

	ctx.JSON(http.StatusOK, response.JSON(TokenResponse{AccessToken: token, TokenType: "Bearer"}, nil))
}
