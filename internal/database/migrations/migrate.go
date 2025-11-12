package migrations

import (
	"gorm.io/gorm"

	"go-base/internal/user"
)

// AutoMigrate runs schema migrations for domain models.
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&user.User{},
	)
}
