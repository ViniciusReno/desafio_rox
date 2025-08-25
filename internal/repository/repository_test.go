package repository

import (
	"context"
	"os"
	"strings"
	"testing"
)

func TestRepositoryInsertBatch(t *testing.T) {
	repo := New()
	ctx := context.Background()
	day := "20240101"
	b, err := os.ReadFile("testdata/sample.txt")
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(b)), "\n")
	exists, err := repo.DayExists(ctx, day)
	if err != nil {
		t.Fatalf("day exists: %v", err)
	}
	if exists {
		t.Fatalf("unexpected existing day")
	}
	if err := repo.InsertBatch(ctx, day, lines); err != nil {
		t.Fatalf("insert: %v", err)
	}
	exists, err = repo.DayExists(ctx, day)
	if err != nil {
		t.Fatalf("day exists: %v", err)
	}
	if !exists {
		t.Fatalf("expected day recorded")
	}
}
