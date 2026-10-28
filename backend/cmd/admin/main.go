// Command admin manages shop admin accounts.
//
//	go run ./cmd/admin create -email mama@example.com
//
// The password is read from ADMIN_PASSWORD so it does not end up in shell history.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/Mentozzzz22/12beautystories-web/backend/internal/auth"
	"github.com/Mentozzzz22/12beautystories-web/backend/internal/config"
	"github.com/Mentozzzz22/12beautystories-web/backend/internal/database"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 || args[0] != "create" {
		return errors.New("usage: admin create -email <email> (password in ADMIN_PASSWORD)")
	}
	fs := flag.NewFlagSet("create", flag.ContinueOnError)
	email := fs.String("email", "", "admin email")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	password := os.Getenv("ADMIN_PASSWORD")
	if password == "" {
		return errors.New("ADMIN_PASSWORD is empty")
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}
	ctx := context.Background()
	pool, err := database.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	if err := database.Migrate(ctx, pool); err != nil {
		return err
	}

	u, err := auth.NewService(pool).CreateUser(ctx, *email, password)
	if err != nil {
		return err
	}
	fmt.Printf("created admin %s (id %d)\n", u.Email, u.ID)
	return nil
}
