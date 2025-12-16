package httpserver

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"go-base/internal/cache"
	"go-base/internal/config"
	"go-base/internal/database"
	"go-base/internal/httpserver/middleware"
	"go-base/internal/httpserver/modules"
	"go-base/internal/logger"
	"go-base/internal/modules/auth"
	"go-base/internal/modules/user"
	"go-base/internal/mq"
	"go-base/internal/security"
	"go-base/internal/validation"
	"go-base/pkg/response"
)

// RouterConfig bundles dependencies required to build the HTTP router.
type RouterConfig struct {
	Config     *config.Config
	Logger     *zap.Logger
	DB         database.Database
	Cache      cache.Cache
	MQ         mq.Client
	Validator  validation.Validator
	JWTManager security.JWTManager
	Password   security.PasswordService
}

// BuildRouter creates the Gin engine with registered routes and middleware.
func BuildRouter(cfg RouterConfig) *gin.Engine {
	mode := gin.ReleaseMode
	if cfg.Config.App.Env != "production" {
		mode = gin.DebugMode
	}
	gin.SetMode(mode)

	r := gin.New()

	log := cfg.Logger
	if log == nil {
		log = logger.L()
	}

	r.Use(gin.Recovery())
	r.Use(middleware.RequestID())
	r.Use(middleware.Logger(log))
	r.Use(middleware.Recovery(log))
	r.Use(middleware.CORS(cfg.Config.Security))

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, response.JSON(gin.H{
			"status":  "ok",
			"version": cfg.Config.App.Version,
		}, nil))
	})

	userRepo := user.NewRepository(cfg.DB.DB())
	userService := user.NewService(userRepo, cfg.Cache, cfg.Config.Redis.TTL)

	apiGroup := r.Group("/api/v1")

	authModule := auth.BuildModule(auth.ModuleConfig{
		Config:    cfg.Config,
		Router:    apiGroup,
		Validator: cfg.Validator,
		Password:  cfg.Password,
		JWT:       cfg.JWTManager,
		Users:     userService,
		MQ:        cfg.MQ,
	})
	authModule.RegisterRoutes()

	modules.InstallAll(apiGroup, modules.Dependencies{
		Config:    cfg.Config,
		Logger:    log,
		DB:        cfg.DB,
		Cache:     cfg.Cache,
		MQ:        cfg.MQ,
		Validator: cfg.Validator,
		JWT:       cfg.JWTManager,
	})

	return r
}

// Application is the interface exposing router dependencies.
type Application interface {
	Run() error
}
