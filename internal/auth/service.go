package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"go-base/internal/mq"
	"go-base/internal/security"
	"go-base/internal/user"
)

// Service handles authentication and related workflows.
type Service interface {
	Register(ctx context.Context, req RegisterRequest) (*user.User, error)
	Login(ctx context.Context, req LoginRequest) (string, error)
}

type service struct {
	users      user.Service
	validator  func(interface{}) error
	password   security.PasswordService
	jwtManager security.JWTManager
	queue      mq.Client
}

// NewService creates an auth service.
func NewService(users user.Service, validator func(interface{}) error, password security.PasswordService, jwt security.JWTManager, queue mq.Client) Service {
	return &service{
		users:      users,
		validator:  validator,
		password:   password,
		jwtManager: jwt,
		queue:      queue,
	}
}

func (s *service) Register(ctx context.Context, req RegisterRequest) (*user.User, error) {
	if err := s.validator(req); err != nil {
		return nil, fmt.Errorf("validate request: %w", err)
	}

	hash, err := s.password.Hash(req.Password)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	newUser := &user.User{
		Email:    req.Email,
		Password: hash,
		FullName: req.FullName,
		Roles:    req.Roles,
	}

	if err := s.users.Create(ctx, newUser); err != nil {
		return nil, err
	}

	if s.queue != nil {
		event := struct {
			ID    string `json:"id"`
			Email string `json:"email"`
		}{
			ID:    newUser.ID,
			Email: newUser.Email,
		}
		payload, err := json.Marshal(event)
		if err == nil {
			_ = s.queue.Publish(ctx, "auth.registered", payload)
		}
	}

	return newUser, nil
}

func (s *service) Login(ctx context.Context, req LoginRequest) (string, error) {
	if err := s.validator(req); err != nil {
		return "", fmt.Errorf("validate request: %w", err)
	}

	u, err := s.users.FindByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, user.ErrNotFound) {
			return "", errors.New("invalid credentials")
		}
		return "", err
	}

	if err := s.password.Compare(u.Password, req.Password); err != nil {
		return "", errors.New("invalid credentials")
	}

	token, err := s.jwtManager.Sign(u.ID, u.Roles)
	if err != nil {
		return "", err
	}

	return token, nil
}
