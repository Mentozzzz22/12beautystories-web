// Command server runs the shop HTTP API.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/Mentozzzz22/12beautystories-web/backend/internal/admin"
	"github.com/Mentozzzz22/12beautystories-web/backend/internal/auth"
	"github.com/Mentozzzz22/12beautystories-web/backend/internal/catalog"
	"github.com/Mentozzzz22/12beautystories-web/backend/internal/config"
	"github.com/Mentozzzz22/12beautystories-web/backend/internal/database"
	"github.com/Mentozzzz22/12beautystories-web/backend/internal/frontend"
	"github.com/Mentozzzz22/12beautystories-web/backend/internal/httpapi"
	"github.com/Mentozzzz22/12beautystories-web/backend/internal/media"
	"github.com/Mentozzzz22/12beautystories-web/backend/internal/notify"
	"github.com/Mentozzzz22/12beautystories-web/backend/internal/order"
)

func main() {
	// The distroless image has no curl, so Docker's healthcheck calls the binary itself.
	if len(os.Args) > 1 && os.Args[1] == "-healthcheck" {
		os.Exit(healthcheck())
	}
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(log); err != nil {
		log.Error("server stopped", "err", err)
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	pool, err := database.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	if err := database.Migrate(ctx, pool); err != nil {
		return err
	}

	var notifier order.Notifier
	if cfg.TelegramEnabled() {
		notifier = notify.NewTelegram(cfg.TelegramBotToken, cfg.TelegramChatID, &http.Client{Timeout: 10 * time.Second})
	} else {
		log.Warn("telegram notifications disabled: TELEGRAM_BOT_TOKEN or TELEGRAM_CHAT_ID is empty")
	}

	orders := order.NewService(order.NewPostgres(pool), notifier, func(err error) {
		log.Error("order notification failed", "err", err)
	})

	uploads, err := media.NewStore(cfg.UploadDir, "/uploads/")
	if err != nil {
		return err
	}

	catalogRepo := catalog.NewPostgres(pool)
	var web http.Handler
	if cfg.StaticDir != "" {
		fe, err := frontend.New(cfg.StaticDir, cfg.PublicURL, catalogRepo, log)
		if err != nil {
			return err
		}
		web = fe
	}

	handler := httpapi.New(httpapi.Deps{
		Catalog:      catalogRepo,
		Orders:       orders,
		Admin:        admin.NewPostgres(pool),
		Auth:         auth.NewService(pool),
		Uploads:      uploads,
		UploadDir:    cfg.UploadDir,
		Frontend:     web,
		CookieSecure: cfg.CookieSecure,
		TrustProxy:   cfg.TrustProxy,
		Log:          log,
	})

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       60 * time.Second, // photo uploads over mobile networks
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Info("listening", "addr", cfg.Addr)
		errCh <- srv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}

func healthcheck() int {
	addr := os.Getenv("APP_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	if strings.HasPrefix(addr, ":") {
		addr = "127.0.0.1" + addr
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://"+addr+"/api/health", nil)
	if err != nil {
		return 1
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 1
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 1
	}
	return 0
}
