package user

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// User represents an application user.
type User struct {
	ID        string                      `gorm:"type:uuid;primaryKey" json:"id"`
	Email     string                      `gorm:"size:255;uniqueIndex;not null" json:"email"`
	Password  string                      `gorm:"size:255;not null" json:"-"`
	FullName  string                      `gorm:"size:255" json:"full_name"`
	Roles     datatypes.JSONSlice[string] `gorm:"type:jsonb" json:"roles"`
	CreatedAt time.Time                   `json:"created_at"`
	UpdatedAt time.Time                   `json:"updated_at"`
	DeletedAt gorm.DeletedAt              `gorm:"index" json:"-"`
}

// BeforeCreate hook ensures IDs are generated for new users.
func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == "" {
		u.ID = uuid.NewString()
	}
	return nil
}
