package app

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"go-base/internal/app/buildinfo"
	"go-base/internal/cache"
	"go-base/internal/config"
	"go-base/internal/database"
	"go-base/internal/httpserver"
	"go-base/internal/logger"
	"go-base/internal/mq"
	"go-base/internal/security"
	"go-base/internal/validation"
)

// Application wires together infrastructure dependencies.
type Application struct {
	Config      *config.Config
	DB          database.Database
	Cache       cache.Cache
	MQ          mq.Client
	Validator   validation.Validator
	JWTManager  security.JWTManager
	PasswordSvc security.PasswordService
	Logger      *zap.Logger
	Router      *gin.Engine
}

// New constructs an application instance with infrastructure initialization.
func New(ctx context.Context, cfg *config.Config) (*Application, error) {
	buildinfo.ReadBuildInfo()

	logger := logger.L()

	db, err := database.NewPostgres(ctx, cfg.Database)
	if err != nil {
		return nil, fmt.Errorf("init database: %w", err)
	}

	cacheClient, err := cache.NewRedis(cfg.Redis)
	if err != nil {
		return nil, fmt.Errorf("init redis: %w", err)
	}

	mqClient, err := mq.NewRabbitMQ(cfg.RabbitMQ)
	if err != nil {
		return nil, fmt.Errorf("init rabbitmq: %w", err)
	}

	validator := validation.New()

	jwtManager, err := security.NewJWTManager(cfg.JWT)
	if err != nil {
		return nil, fmt.Errorf("init jwt manager: %w", err)
	}

	passwordSvc := security.NewPasswordService()

	router := httpserver.BuildRouter(httpserver.RouterConfig{
		Config:     cfg,
		Logger:     logger,
		DB:         db,
		Cache:      cacheClient,
		MQ:         mqClient,
		Validator:  validator,
		JWTManager: jwtManager,
		Password:   passwordSvc,
	})

	return &Application{
		Config:      cfg,
		DB:          db,
		Cache:       cacheClient,
		MQ:          mqClient,
		Validator:   validator,
		JWTManager:  jwtManager,
		PasswordSvc: passwordSvc,
		Logger:      logger,
		Router:      router,
	}, nil
}

// Close releases connections.
func (a *Application) Close(ctx context.Context) error {
	if err := a.MQ.Close(); err != nil {
		logger.L().Warn("failed to close rabbitmq", zap.Error(err))
	}

	if err := a.Cache.Close(); err != nil {
		logger.L().Warn("failed to close redis", zap.Error(err))
	}

	if err := a.DB.Close(); err != nil {
		logger.L().Warn("failed to close database", zap.Error(err))
	}

	return nil
}
