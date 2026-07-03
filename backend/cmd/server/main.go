package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"wedding/backend/internal/auth"
	"wedding/backend/internal/config"
	"wedding/backend/internal/db"
	"wedding/backend/internal/email"
	"wedding/backend/internal/invite"
	"wedding/backend/internal/server"
)

func main() {
	// Healthcheck mode: `server healthcheck` makes an HTTP GET to the local
	// /healthz endpoint and exits 0 on 200, 1 otherwise. This lets a distroless
	// image's HEALTHCHECK run without a shell or extra binary. PORT is read from
	// the same env the server uses.
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		os.Exit(runHealthcheck())
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	d, err := db.Open(cfg.DBPath)
	if err != nil {
		if d != nil {
			d.Close()
		}
		log.Fatalf("open db: %v", err)
	}

	if err := db.Migrate(d); err != nil {
		d.Close()
		log.Fatalf("migrate: %v", err)
	}

	store := db.NewSQLiteStore(d)

	var sender email.Sender
	if cfg.ResendAPIKey == "fake" {
		log.Println("using fake email sender (staging)")
		sender = &email.Fake{}
	} else {
		sender = email.NewResend(cfg.ResendAPIKey, cfg.ResendFrom, cfg.ResendTo)
	}

	svc := invite.NewService(store, sender)
	a := auth.New(cfg.AdminPassword, cfg.SessionSecret, cfg.SecureCookie)
	handler := server.New(svc, a, d, cfg.CORSAllowedOrigins)

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	go func() {
		log.Printf("listening on :%s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			d.Close()
			log.Fatalf("server: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("shutdown: %v", err)
	}
	if err := d.Close(); err != nil {
		log.Printf("close db: %v", err)
	}
}

// runHealthcheck probes the local /healthz endpoint and returns the process exit
// code: 0 if it responds 200, 1 otherwise. PORT is resolved the same way the
// server resolves it (PORT env, default 8080) so the probe always targets the
// running listener.
func runHealthcheck() int {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://127.0.0.1:"+port+"/healthz", http.NoBody)
	if err != nil {
		log.Printf("healthcheck: %v", err)
		return 1
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("healthcheck: %v", err)
		return 1
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Printf("healthcheck: status %d", resp.StatusCode)
		return 1
	}
	return 0
}
