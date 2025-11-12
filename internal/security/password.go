package security

import "golang.org/x/crypto/bcrypt"

// PasswordService handles password hashing and comparison.
type PasswordService interface {
	Hash(password string) (string, error)
	Compare(hash, password string) error
}

type bcryptPasswordService struct{}

// NewPasswordService returns a bcrypt-based password service.
func NewPasswordService() PasswordService {
	return &bcryptPasswordService{}
}

func (b *bcryptPasswordService) Hash(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func (b *bcryptPasswordService) Compare(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}
