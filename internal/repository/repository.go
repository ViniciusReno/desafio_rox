package repository

import (
	"context"
	"sync"
	"time"
)

type Repository struct {
	mu   sync.Mutex
	days map[string]bool
}

func New() *Repository {
	return &Repository{days: make(map[string]bool)}
}
func (r *Repository) DayExists(ctx context.Context, day string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, ok := r.days[day]
	return ok, nil
}
func (r *Repository) Begin(ctx context.Context, day string) *Tx {
	return &Tx{repo: r, day: day}
}
func (r *Repository) InsertBatch(ctx context.Context, day string, lines []string) error {
	return retry(ctx, 3, func() error {
		tx := r.Begin(ctx, day)
		for _, line := range lines {
			if err := tx.InsertLine(ctx, line); err != nil {
				return err
			}
		}
		return tx.Commit()
	})
}

type Tx struct {
	repo  *Repository
	day   string
	lines []string
}

func (tx *Tx) InsertLine(ctx context.Context, line string) error {
	tx.lines = append(tx.lines, line)
	return nil
}
func (tx *Tx) Commit() error {
	tx.repo.mu.Lock()
	defer tx.repo.mu.Unlock()
	tx.repo.days[tx.day] = true
	return nil
}
func retry(ctx context.Context, attempts int, fn func() error) error {
	var err error
	for i := 0; i < attempts; i++ {
		if err = fn(); err == nil {
			return nil
		}
		select {
		case <-time.After(time.Second):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return err
}
