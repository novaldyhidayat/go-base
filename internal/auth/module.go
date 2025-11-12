package auth

import (
	"github.com/gin-gonic/gin"

	"go-base/internal/config"
	"go-base/internal/mq"
	"go-base/internal/security"
	"go-base/internal/user"
	"go-base/internal/validation"
)

type buildAuthModuleConfig struct {
	Config    *config.Config
	Router    *gin.RouterGroup
	Validator validation.Validator
	Password  security.PasswordService
	JWT       security.JWTManager
	Users     user.Service
	MQ        mq.Client
}

type module struct {
	controller *Controller
	router     *gin.RouterGroup
}

func buildAuthModule(cfg buildAuthModuleConfig) *module {
	service := NewService(
		cfg.Users,
		func(i interface{}) error { return cfg.Validator.Struct(i) },
		cfg.Password,
		cfg.JWT,
		cfg.MQ,
	)

	controller := NewController(service)

	return &module{
		controller: controller,
		router:     cfg.Router,
	}
}

func (m *module) registerRoutes() {
	r := m.router.Group("/auth")
	r.POST("/register", m.controller.Register)
	r.POST("/login", m.controller.Login)
}
