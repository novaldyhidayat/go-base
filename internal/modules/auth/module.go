package auth

import (
	"github.com/gin-gonic/gin"

	"go-base/internal/config"
	"go-base/internal/modules/user"
	"go-base/internal/mq"
	"go-base/internal/security"
	"go-base/internal/validation"
)

type ModuleConfig struct {
	Config    *config.Config
	Router    *gin.RouterGroup
	Validator validation.Validator
	Password  security.PasswordService
	JWT       security.JWTManager
	Users     user.Service
	MQ        mq.Client
}

type Module struct {
	controller *Controller
	router     *gin.RouterGroup
}

func BuildModule(cfg ModuleConfig) *Module {
	service := NewService(
		cfg.Users,
		func(i interface{}) error { return cfg.Validator.Struct(i) },
		cfg.Password,
		cfg.JWT,
		cfg.MQ,
	)

	controller := NewController(service)

	return &Module{
		controller: controller,
		router:     cfg.Router,
	}
}

func (m *Module) RegisterRoutes() {
	r := m.router.Group("/auth")
	r.POST("/register", m.controller.Register)
	r.POST("/login", m.controller.Login)
}
