package auth

import (
	"context"
	"errors"
	"testing"

	"github.com/rabbitmq/amqp091-go"

	"go-base/internal/modules/user"
	"go-base/internal/security"
)

type fakeUsers struct {
	created *user.User
	found   *user.User
}

func (f *fakeUsers) Create(_ context.Context, u *user.User) error {
	f.created = u
	return nil
}

func (f *fakeUsers) FindByEmail(_ context.Context, _ string) (*user.User, error) {
	if f.found == nil {
		return nil, user.ErrNotFound
	}
	return f.found, nil
}

type fakePassword struct{}

func (fakePassword) Hash(string) (string, error)  { return "hashed", nil }
func (fakePassword) Compare(string, string) error { return nil }

type fakeJWT struct{}

func (fakeJWT) Sign(string, []string) (string, error)   { return "token", nil }
func (fakeJWT) Verify(string) (*security.Claims, error) { return nil, errors.New("unused") }

type fakeQueue struct{}

func (fakeQueue) Publish(context.Context, string, []byte) error  { return nil }
func (fakeQueue) Subscribe(string, func(amqp091.Delivery)) error { return nil }
func (fakeQueue) Close() error                                   { return nil }

func TestRegisterNormalizesEmailAndAssignsUserRole(t *testing.T) {
	users := &fakeUsers{}
	svc := NewService(users, func(any) error { return nil }, fakePassword{}, fakeJWT{}, fakeQueue{})

	created, err := svc.Register(context.Background(), RegisterRequest{
		Email:    "  ADMIN@Example.COM ",
		Password: "password",
		FullName: "Admin",
	})
	if err != nil {
		t.Fatal(err)
	}

	if created.Email != "admin@example.com" {
		t.Fatalf("email = %q, want normalized email", created.Email)
	}
	if len(created.Roles) != 1 || created.Roles[0] != "user" {
		t.Fatalf("roles = %#v, want [user]", created.Roles)
	}
	if users.created != created {
		t.Fatal("created user was not passed to repository service")
	}
}

func TestLoginReturnsInvalidCredentialsForUnknownUser(t *testing.T) {
	svc := NewService(&fakeUsers{}, func(any) error { return nil }, fakePassword{}, fakeJWT{}, nil)
	_, err := svc.Login(context.Background(), LoginRequest{Email: "unknown@example.com", Password: "password"})
	if err == nil || err.Error() != "invalid credentials" {
		t.Fatalf("error = %v, want invalid credentials", err)
	}
}
