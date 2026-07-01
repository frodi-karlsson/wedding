package main

import (
	"context"
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
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	d, err := db.Open(cfg.DBPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer d.Close()

	if err := db.Migrate(d); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	store := db.NewSQLiteStore(d)
	sender := email.NewResend(cfg.ResendAPIKey, cfg.ResendFrom, cfg.ResendTo)
	svc := invite.NewService(store, sender)
	a := auth.New(cfg.AdminPassword, cfg.SessionSecret)
	handler := server.New(svc, a, cfg.CORSAllowedOrigins)

	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: handler,
	}

	go func() {
		log.Printf("listening on :%s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
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
}
