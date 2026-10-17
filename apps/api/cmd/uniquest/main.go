// Command uniquest is the UniQuest backend: one binary serving the API, the SSE stream and the web app.
//
//	uniquest serve                         run the server (applies migrations first)
//	uniquest migrate                       apply database migrations
//	uniquest seed [-date YYYY-MM-DD] [-activate] [-admin email]
//	                                       create/refresh the demo event from content/demo-event.json
//	uniquest gen-vapid                     print a new VAPID key pair for Web Push
//	uniquest cleanup                       delete inactive guests and expired sessions/codes
//	uniquest health                        exit 0 if the local server answers /api/v1/healthz (container healthcheck)
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/git-maks/pl-app/apps/api/internal/app"
	"github.com/git-maks/pl-app/apps/api/internal/catalog"
	"github.com/git-maks/pl-app/apps/api/internal/config"
	"github.com/git-maks/pl-app/apps/api/internal/httpapi"
	"github.com/git-maks/pl-app/apps/api/internal/mail"
	"github.com/git-maks/pl-app/apps/api/internal/push"
	"github.com/git-maks/pl-app/apps/api/internal/realtime"
	"github.com/git-maks/pl-app/apps/api/internal/store"
	"github.com/git-maks/pl-app/apps/api/internal/webui"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func newLogger(level string) *slog.Logger {
	var l slog.Level
	_ = l.UnmarshalText([]byte(level))
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: l}))
}

func run(args []string) error {
	cmd := "serve"
	if len(args) > 0 {
		cmd, args = args[0], args[1:]
	}
	if cmd == "health" {
		addr := os.Getenv("UQ_ADDR")
		if addr == "" || addr[0] == ':' {
			addr = "127.0.0.1" + addr
		}
		if addr == "127.0.0.1" {
			addr += ":8080"
		}
		c := http.Client{Timeout: 3 * time.Second}
		res, err := c.Get("http://" + addr + "/api/v1/healthz")
		if err != nil {
			return err
		}
		res.Body.Close()
		if res.StatusCode != http.StatusOK {
			return fmt.Errorf("health: %s", res.Status)
		}
		return nil
	}
	if cmd == "gen-vapid" {
		priv, pub, err := push.GenerateKeys()
		if err != nil {
			return err
		}
		fmt.Printf("UQ_VAPID_PUBLIC=%s\nUQ_VAPID_PRIVATE=%s\n", pub, priv)
		return nil
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	log := newLogger(cfg.LogLevel)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := store.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer db.Close()
	applied, err := store.Migrate(ctx, db)
	if err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	for _, m := range applied {
		log.Info("migration applied", "name", m)
	}
	if cmd == "migrate" {
		return nil
	}

	cat, err := catalog.Load()
	if err != nil {
		return err
	}
	hub := realtime.NewHub(db, log)
	pusher := push.New(db, log, cfg.VAPIDPublic, cfg.VAPIDPrivate, cfg.VAPIDSubject)
	mailer := mail.SMTP{Host: cfg.SMTPHost, Port: cfg.SMTPPort, User: cfg.SMTPUser, Password: cfg.SMTPPassword, From: cfg.SMTPFrom}
	svc := app.New(db, cfg, cat, hub, pusher, mailer, log)

	switch cmd {
	case "seed":
		fs := flag.NewFlagSet("seed", flag.ExitOnError)
		date := fs.String("date", time.Now().Format("2006-01-02"), "event day (YYYY-MM-DD)")
		activate := fs.Bool("activate", true, "make it the active event")
		admin := fs.String("admin", "", "create or promote an admin account with this e-mail")
		_ = fs.Parse(args)
		day, err := time.Parse("2006-01-02", *date)
		if err != nil {
			return fmt.Errorf("-date: %w", err)
		}
		id, err := svc.SeedEvent(ctx, cat.DemoEvent, app.SeedOptions{Date: day, Activate: *activate})
		if err != nil {
			return err
		}
		log.Info("event seeded", "id", id, "slug", cat.DemoEvent.Slug, "date", *date, "active", *activate)
		if *admin != "" {
			if err := svc.EnsureAdmin(ctx, *admin); err != nil {
				return err
			}
			log.Info("admin ensured", "email_domain", (*admin)[max(0, len(*admin)-12):])
		}
		return nil
	case "cleanup":
		n, err := svc.PurgeInactiveGuests(ctx, 365*24*time.Hour)
		log.Info("cleanup done", "guests_deleted", n)
		return err
	case "serve":
		return serve(ctx, cfg, log, svc, hub)
	default:
		return fmt.Errorf("unknown command %q", cmd)
	}
}

func serve(ctx context.Context, cfg config.Config, log *slog.Logger, svc *app.Service, hub *realtime.Hub) error {
	static := webui.FS()
	if static == nil {
		log.Warn("web app not embedded – serving the API only")
	}
	api := httpapi.New(svc, hub, cfg, log, static)
	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           api.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		IdleTimeout:       120 * time.Second,
		// No WriteTimeout: the SSE stream is long-lived; API handlers have their own 15 s deadline.
	}

	go hub.Run(ctx)
	go every(ctx, time.Minute, func() {
		if n, err := svc.SendDueReminders(ctx); err != nil {
			log.Warn("reminders", "err", err)
		} else if n > 0 {
			log.Info("reminders sent", "count", n)
		}
	})
	go every(ctx, 10*time.Minute, api.SweepLimiters)
	go every(ctx, 24*time.Hour, func() {
		if n, err := svc.PurgeInactiveGuests(ctx, 365*24*time.Hour); err == nil && n > 0 {
			log.Info("inactive guests purged", "count", n)
		}
	})

	errc := make(chan error, 1)
	go func() {
		log.Info("listening", "addr", cfg.Addr, "base_url", cfg.BaseURL, "version", cfg.Version, "push", svc.Push.Enabled())
		errc <- srv.ListenAndServe()
	}()
	select {
	case err := <-errc:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	case <-ctx.Done():
	}
	shutdown, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	log.Info("shutting down")
	api.Close()
	return srv.Shutdown(shutdown)
}

func every(ctx context.Context, d time.Duration, fn func()) {
	t := time.NewTicker(d)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			fn()
		}
	}
}
