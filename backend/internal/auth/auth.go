// Package auth manages admin accounts and cookie sessions.
package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"github.com/Mentozzzz22/12beautystories-web/backend/internal/apperr"
)

// Errors returned by the service.
var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrUnauthenticated    = errors.New("not authenticated")
)

const (
	// SessionTTL is how long an admin stays signed in.
	SessionTTL = 30 * 24 * time.Hour
	// MinPasswordLen is the shortest accepted password.
	MinPasswordLen = 8
	tokenBytes     = 32
)

// User is an admin account.
type User struct {
	ID    int64  `json:"id"`
	Email string `json:"email"`
}

// Session is an issued login.
type Session struct {
	Token     string
	ExpiresAt time.Time
	User      User
}

// Service stores users and sessions in Postgres.
type Service struct {
	db  *pgxpool.Pool
	now func() time.Time
	// dummyHash keeps login timing equal for unknown emails.
	dummyHash []byte
}

// NewService creates an auth service.
func NewService(db *pgxpool.Pool) *Service {
	dummy, _ := bcrypt.GenerateFromPassword([]byte("timing-equalizer"), bcrypt.DefaultCost)
	return &Service{db: db, now: time.Now, dummyHash: dummy}
}

// CreateUser adds an admin. Email is normalized to lower case.
func (s *Service) CreateUser(ctx context.Context, email, password string) (User, error) {
	email = normalizeEmail(email)
	fields := apperr.Fields{}
	if _, err := mail.ParseAddress(email); err != nil || email == "" {
		fields.Add("email", apperr.Invalid)
	}
	if len(password) < MinPasswordLen {
		fields.Add("password", apperr.Invalid)
	}
	if err := fields.Err(); err != nil {
		return User{}, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return User{}, fmt.Errorf("hash password: %w", err)
	}
	u := User{Email: email}
	err = s.db.QueryRow(ctx,
		`INSERT INTO admin_users (email, password_hash) VALUES ($1, $2) RETURNING id`, email, string(hash)).
		Scan(&u.ID)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return User{}, apperr.Field("email", apperr.Taken)
	}
	if err != nil {
		return User{}, fmt.Errorf("insert user: %w", err)
	}
	return u, nil
}

// Login checks credentials and issues a new session.
func (s *Service) Login(ctx context.Context, email, password string) (Session, error) {
	var (
		u    User
		hash string
	)
	err := s.db.QueryRow(ctx,
		`SELECT id, email, password_hash FROM admin_users WHERE email = $1`, normalizeEmail(email)).
		Scan(&u.ID, &u.Email, &hash)
	if errors.Is(err, pgx.ErrNoRows) {
		_ = bcrypt.CompareHashAndPassword(s.dummyHash, []byte(password))
		return Session{}, ErrInvalidCredentials
	}
	if err != nil {
		return Session{}, fmt.Errorf("query user: %w", err)
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) != nil {
		return Session{}, ErrInvalidCredentials
	}

	token, err := newToken()
	if err != nil {
		return Session{}, err
	}
	expires := s.now().Add(SessionTTL)
	if _, err := s.db.Exec(ctx,
		`INSERT INTO admin_sessions (token_hash, user_id, expires_at) VALUES ($1, $2, $3)`,
		hashToken(token), u.ID, expires); err != nil {
		return Session{}, fmt.Errorf("insert session: %w", err)
	}
	// Opportunistic cleanup keeps the table small without a background job.
	_, _ = s.db.Exec(ctx, `DELETE FROM admin_sessions WHERE expires_at < $1`, s.now())

	return Session{Token: token, ExpiresAt: expires, User: u}, nil
}

// Authenticate resolves a session token to its user.
func (s *Service) Authenticate(ctx context.Context, token string) (User, error) {
	if token == "" {
		return User{}, ErrUnauthenticated
	}
	var u User
	err := s.db.QueryRow(ctx, `
		SELECT u.id, u.email
		FROM admin_sessions s JOIN admin_users u ON u.id = s.user_id
		WHERE s.token_hash = $1 AND s.expires_at > $2`, hashToken(token), s.now()).
		Scan(&u.ID, &u.Email)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrUnauthenticated
	}
	if err != nil {
		return User{}, fmt.Errorf("query session: %w", err)
	}
	return u, nil
}

// Logout revokes a session token.
func (s *Service) Logout(ctx context.Context, token string) error {
	if _, err := s.db.Exec(ctx, `DELETE FROM admin_sessions WHERE token_hash = $1`, hashToken(token)); err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}

func newToken() (string, error) {
	b := make([]byte, tokenBytes)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func hashToken(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:]
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
