package seed

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"go-base/internal/cache"
	"go-base/internal/config"
	"go-base/internal/database"
	"go-base/internal/logger"
	"go-base/internal/modules/user"
	"go-base/internal/security"
)

// Run executes database seeders based on configuration.
func Run(ctx context.Context, cfg *config.Config, db database.Database, cache cache.Cache, password security.PasswordService) error {
	if !cfg.Seed.Enabled {
		logger.L().Info("seeding disabled")
		return nil
	}

	log := logger.L()
	log.Info("running seeders")

	repo := user.NewRepository(db.DB())
	service := user.NewService(repo, cache, cfg.Redis.TTL)

	adminEmail := cfg.Seed.AdminEmail
	adminPassword := cfg.Seed.AdminPassword
	if adminEmail == "" || adminPassword == "" {
		return fmt.Errorf("admin credentials must be provided in seed config")
	}

	if _, err := service.FindByEmail(ctx, adminEmail); err == nil {
		log.Info("admin user already exists", zap.String("email", adminEmail))
		return nil
	}

	hash, err := password.Hash(adminPassword)
	if err != nil {
		return fmt.Errorf("hash admin password: %w", err)
	}

	admin := &user.User{
		Email:    adminEmail,
		FullName: "Administrator",
		Password: hash,
		Roles:    []string{"admin"},
	}

	if err := service.Create(ctx, admin); err != nil {
		return fmt.Errorf("create admin user: %w", err)
	}

	log.Info("admin user seeded", zap.String("email", adminEmail))

	return nil
}
