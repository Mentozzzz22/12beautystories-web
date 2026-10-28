// Package dbtest provides an isolated, migrated database for integration tests.
package dbtest

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Mentozzzz22/12beautystories-web/backend/internal/database"
)

// EnvURL names the variable with a Postgres URL whose user may create databases.
const EnvURL = "TEST_DATABASE_URL"

// New creates a fresh database, applies migrations and drops it when the test ends.
// The test is skipped when TEST_DATABASE_URL is not set.
func New(t *testing.T) *pgxpool.Pool {
	t.Helper()
	adminURL := os.Getenv(EnvURL)
	if adminURL == "" {
		t.Skipf("%s is not set", EnvURL)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	admin, err := pgx.Connect(ctx, adminURL)
	if err != nil {
		t.Fatalf("connect admin: %v", err)
	}
	defer func() { _ = admin.Close(ctx) }()

	name := "test_" + randomSuffix(t)
	if _, err := admin.Exec(ctx, "CREATE DATABASE "+name); err != nil {
		t.Fatalf("create database: %v", err)
	}

	u, err := url.Parse(adminURL)
	if err != nil {
		t.Fatalf("parse url: %v", err)
	}
	u.Path = "/" + name

	pool, err := database.Open(ctx, u.String())
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	t.Cleanup(func() {
		pool.Close()
		cctx, ccancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer ccancel()
		conn, err := pgx.Connect(cctx, adminURL)
		if err != nil {
			t.Errorf("connect admin for cleanup: %v", err)
			return
		}
		defer func() { _ = conn.Close(cctx) }()
		if _, err := conn.Exec(cctx, "DROP DATABASE IF EXISTS "+name+" WITH (FORCE)"); err != nil {
			t.Errorf("drop database: %v", err)
		}
	})

	if err := database.Migrate(ctx, pool); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return pool
}

// Exec runs setup SQL and fails the test on error.
func Exec(t *testing.T, pool *pgxpool.Pool, sql string, args ...any) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), sql, args...); err != nil {
		t.Fatalf("exec %q: %v", sql, err)
	}
}

// Seed loads backend/db/seed.sql, the same demo data used in development.
func Seed(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	_, file, _, _ := runtime.Caller(0)
	sql, err := os.ReadFile(filepath.Join(filepath.Dir(file), "..", "..", "..", "db", "seed.sql"))
	if err != nil {
		t.Fatalf("read seed: %v", err)
	}
	Exec(t, pool, string(sql))
}

func randomSuffix(t *testing.T) string {
	t.Helper()
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		t.Fatalf("random: %v", err)
	}
	return hex.EncodeToString(b)
}
