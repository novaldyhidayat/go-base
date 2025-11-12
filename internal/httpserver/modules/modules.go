package modules

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"go-base/internal/cache"
	"go-base/internal/config"
	"go-base/internal/database"
	"go-base/internal/mq"
	"go-base/internal/security"
	"go-base/internal/validation"
)

// Dependencies aggregates shared resources for generated modules.
type Dependencies struct {
	Config    *config.Config
	Logger    *zap.Logger
	DB        database.Database
	Cache     cache.Cache
	MQ        mq.Client
	Validator validation.Validator
	JWT       security.JWTManager
}

// Installer registers routes for a module onto the provided router group.
type Installer func(router *gin.RouterGroup, deps Dependencies)

var installers []Installer

// Register adds an installer callback executed during router construction.
func Register(installer Installer) {
	installers = append(installers, installer)
}

// InstallAll executes all registered installers.
func InstallAll(router *gin.RouterGroup, deps Dependencies) {
	for _, installer := range installers {
		installer(router, deps)
	}
}
