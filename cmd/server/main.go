package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/knorVV/github_sync_go/internal/api"
	"github.com/knorVV/github_sync_go/internal/config"
	"github.com/knorVV/github_sync_go/internal/github"
	"github.com/knorVV/github_sync_go/internal/scheduler"
	"github.com/knorVV/github_sync_go/internal/storage"
)

func main() {
	cfg := config.Load()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool, err := storage.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer pool.Close()

	ghClient := github.NewClient(cfg.GithubToken)
	store := storage.NewRepoStore(pool)

	sched := scheduler.New(cfg.SyncInterval, ghClient, store)
	sched.Start()
	defer sched.Stop()

	handler := api.NewHandler(store, sched)

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      handler.Router(),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	go func() {
		log.Printf("Server listening on :%s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Println("Shutting down...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	_ = srv.Shutdown(shutdownCtx)
}
