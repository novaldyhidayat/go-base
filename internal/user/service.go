package user

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go-base/internal/cache"
)

// Service exposes user operations.
type Service interface {
	Create(ctx context.Context, user *User) error
	FindByEmail(ctx context.Context, email string) (*User, error)
}

type service struct {
	repo     Repository
	cache    cache.Cache
	cacheTTL time.Duration
}

// NewService builds a user service with the given repository and cache.
func NewService(repo Repository, cache cache.Cache, cacheTTL time.Duration) Service {
	return &service{repo: repo, cache: cache, cacheTTL: cacheTTL}
}

func (s *service) cacheKey(email string) string {
	return fmt.Sprintf("user:email:%s", email)
}

func (s *service) Create(ctx context.Context, user *User) error {
	if err := s.repo.Create(ctx, user); err != nil {
		return err
	}

	if s.cache != nil {
		payload, err := json.Marshal(user)
		if err == nil {
			ttl := s.cacheTTL
			if ttl <= 0 {
				ttl = 5 * time.Minute
			}
			_ = s.cache.Set(ctx, s.cacheKey(user.Email), payload, ttl)
		}
	}

	return nil
}

func (s *service) FindByEmail(ctx context.Context, email string) (*User, error) {
	if s.cache != nil {
		if cached, err := s.cache.Get(ctx, s.cacheKey(email)); err == nil && cached != "" {
			var u User
			if err := json.Unmarshal([]byte(cached), &u); err == nil {
				return &u, nil
			}
		}
	}

	u, err := s.repo.FindByEmail(ctx, email)
	if err != nil {
		return nil, err
	}

	if s.cache != nil {
		payload, err := json.Marshal(u)
		if err == nil {
			ttl := s.cacheTTL
			if ttl <= 0 {
				ttl = 5 * time.Minute
			}
			_ = s.cache.Set(ctx, s.cacheKey(email), payload, ttl)
		}
	}

	return u, nil
}
