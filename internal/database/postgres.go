package database

import (
	"context"
	"fmt"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"go-base/internal/config"
)

// Database abstracts the persistence operations needed by the app.
type Database interface {
	DB() *gorm.DB
	Close() error
}

type postgresDB struct {
	db *gorm.DB
}

// NewPostgres returns a PostgreSQL-backed database implementation.
func NewPostgres(ctx context.Context, cfg config.DatabaseConfig) (Database, error) {
	dialector := postgres.Open(cfg.DSN)

	gormCfg := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	}

	db, err := gorm.Open(dialector, gormCfg)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("db connection: %w", err)
	}

	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := sqlDB.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return &postgresDB{db: db}, nil
}

func (p *postgresDB) DB() *gorm.DB {
	return p.db
}

func (p *postgresDB) Close() error {
	sqlDB, err := p.db.DB()
	if err != nil {
		return fmt.Errorf("db connection: %w", err)
	}

	return sqlDB.Close()
}
