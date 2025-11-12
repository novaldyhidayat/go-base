package validation

import (
	"github.com/go-playground/validator/v10"
)

// Validator wraps go-playground/validator for dependency injection.
type Validator interface {
	Struct(any) error
}

type defaultValidator struct {
	validate *validator.Validate
}

// New returns a Validator instance.
func New() Validator {
	return &defaultValidator{validate: validator.New()}
}

func (v *defaultValidator) Struct(s any) error {
	return v.validate.Struct(s)
}
