package domain

import (
	"context"
	"testing"
)

type mockRepo struct {
	dayExists    bool
	insertCalled bool
	insertLines  []string
}

func (m *mockRepo) DayExists(ctx context.Context, day string) (bool, error) {
	return m.dayExists, nil
}

func (m *mockRepo) InsertBatch(ctx context.Context, day string, lines []string) error {
	m.insertCalled = true
	m.insertLines = append([]string{}, lines...)
	return nil
}

func TestServiceImport(t *testing.T) {
	repo := &mockRepo{}
	svc := NewService(repo)
	lines := []string{"a", "b"}
	err := svc.Import(context.Background(), "20240101", lines)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !repo.insertCalled {
		t.Fatalf("expected insert call")
	}
	if len(repo.insertLines) != len(lines) {
		t.Fatalf("lines mismatch")
	}
}

func TestServiceImportSkip(t *testing.T) {
	repo := &mockRepo{dayExists: true}
	svc := NewService(repo)
	err := svc.Import(context.Background(), "20240101", []string{"x"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.insertCalled {
		t.Fatalf("unexpected insert call")
	}
}
