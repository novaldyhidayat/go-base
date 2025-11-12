package httpserver

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"go-base/internal/config"
)

// Server wraps an HTTP server with graceful shutdown.
type Server struct {
	engine *gin.Engine
	cfg    config.ServerConfig
	log    *zap.Logger
}

// NewServer constructs a Server.
func NewServer(engine *gin.Engine, cfg config.ServerConfig, log *zap.Logger) *Server {
	return &Server{engine: engine, cfg: cfg, log: log}
}

// Run starts the HTTP server and blocks until context cancellation.
func (s *Server) Run(ctx context.Context) error {
	addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)

	httpServer := &http.Server{
		Addr:         addr,
		Handler:      s.engine,
		ReadTimeout:  s.cfg.ReadTimeout,
		WriteTimeout: s.cfg.WriteTimeout,
		IdleTimeout:  s.cfg.IdleTimeout,
	}

	errCh := make(chan error, 1)

	go func() {
		s.log.Info("HTTP server starting", zap.String("addr", addr))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), s.cfg.GracefulTimeout)
		defer cancel()

		s.log.Info("HTTP server shutting down")
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown server: %w", err)
		}
	case err := <-errCh:
		return err
	}

	return nil
}
