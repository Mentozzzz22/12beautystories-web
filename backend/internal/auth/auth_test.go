package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Mentozzzz22/12beautystories-web/backend/internal/apperr"
	"github.com/Mentozzzz22/12beautystories-web/backend/internal/database/dbtest"
)

func TestAuthFlow(t *testing.T) {
	svc := NewService(dbtest.New(t))
	ctx := context.Background()

	if _, err := svc.CreateUser(ctx, " Mama@Example.com ", "secret-password"); err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}

	if _, err := svc.Login(ctx, "mama@example.com", "wrong-password"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("Login(wrong password) error = %v, want ErrInvalidCredentials", err)
	}
	if _, err := svc.Login(ctx, "nobody@example.com", "secret-password"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("Login(unknown) error = %v, want ErrInvalidCredentials", err)
	}

	sess, err := svc.Login(ctx, "MAMA@example.com", "secret-password")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	u, err := svc.Authenticate(ctx, sess.Token)
	if err != nil || u.Email != "mama@example.com" {
		t.Fatalf("Authenticate() = %+v, %v", u, err)
	}

	if err := svc.Logout(ctx, sess.Token); err != nil {
		t.Fatalf("Logout() error = %v", err)
	}
	if _, err := svc.Authenticate(ctx, sess.Token); !errors.Is(err, ErrUnauthenticated) {
		t.Errorf("Authenticate(after logout) error = %v, want ErrUnauthenticated", err)
	}
}

func TestSessionExpires(t *testing.T) {
	svc := NewService(dbtest.New(t))
	ctx := context.Background()
	if _, err := svc.CreateUser(ctx, "mama@example.com", "secret-password"); err != nil {
		t.Fatal(err)
	}
	sess, err := svc.Login(ctx, "mama@example.com", "secret-password")
	if err != nil {
		t.Fatal(err)
	}

	svc.now = func() time.Time { return time.Now().Add(SessionTTL + time.Hour) }
	if _, err := svc.Authenticate(ctx, sess.Token); !errors.Is(err, ErrUnauthenticated) {
		t.Errorf("Authenticate(expired) error = %v, want ErrUnauthenticated", err)
	}
}

func TestCreateUserValidation(t *testing.T) {
	svc := NewService(dbtest.New(t))
	ctx := context.Background()

	_, err := svc.CreateUser(ctx, "not-an-email", "short")
	var verr *apperr.ValidationError
	if !errors.As(err, &verr) || verr.Fields["email"] == "" || verr.Fields["password"] == "" {
		t.Fatalf("CreateUser() error = %v, want email and password errors", err)
	}

	if _, err := svc.CreateUser(ctx, "mama@example.com", "secret-password"); err != nil {
		t.Fatal(err)
	}
	_, err = svc.CreateUser(ctx, "MAMA@example.com", "another-password")
	if !errors.As(err, &verr) || verr.Fields["email"] != apperr.Taken {
		t.Errorf("CreateUser(duplicate) error = %v, want email taken", err)
	}
}
