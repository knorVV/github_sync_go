package scheduler

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/knorVV/github_sync_go/internal/github"

	"github.com/knorVV/github_sync_go/internal/storage"
)

type Scheduler struct {
	interval time.Duration
	client   *github.Client
	store    *storage.RepoStore
	mu       sync.Mutex
	timer    *time.Timer
	ctx      context.Context
	cancel   context.CancelFunc
}

func New(interval time.Duration, client *github.Client, store *storage.RepoStore) *Scheduler {
	ctx, cancel := context.WithCancel(context.Background())
	return &Scheduler{
		interval: interval,
		client:   client,
		store:    store,
		ctx:      ctx,
		cancel:   cancel,
	}
}

// Start запускает планировщик: первая синхронизация сразу,
// затем каждые interval.
func (s *Scheduler) Start() {
	s.syncNow()
	s.scheduleNext()
}

// ForceSync сбрасывает таймер и запускает синхронизацию немедленно.
func (s *Scheduler) ForceSync() {
	s.mu.Lock()
	if s.timer != nil {
		s.timer.Stop()
	}
	s.mu.Unlock()

	s.syncNow()
	s.scheduleNext()
}

func (s *Scheduler) scheduleNext() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.timer = time.AfterFunc(s.interval, func() {
		s.syncNow()
		s.scheduleNext()
	})
}

func (s *Scheduler) syncNow() {
	log.Println("Starting GitHub sync...")

	ctx, cancel := context.WithTimeout(s.ctx, 30*time.Second)
	defer cancel()

	repos, err := s.client.FetchTrending(ctx, 30)
	if err != nil {
		log.Printf("Sync failed: %v", err)
		return
	}

	if err := s.store.UpsertAll(ctx, repos); err != nil {
		log.Printf("DB upsert failed: %v", err)
		return
	}

	log.Printf("Synced %d repositories", len(repos))
}

func (s *Scheduler) Stop() {
	s.cancel()
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.timer != nil {
		s.timer.Stop()
	}
}
