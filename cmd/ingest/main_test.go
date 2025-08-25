package main

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"
)

func zipBytes(files map[string]string) []byte {
	buf := new(bytes.Buffer)
	zw := zip.NewWriter(buf)
	for name, content := range files {
		w, _ := zw.Create(name)
		_, _ = w.Write([]byte(content))
	}
	_ = zw.Close()
	return buf.Bytes()
}

func TestPrevBusinessDays(t *testing.T) {
	ref := time.Date(2024, 5, 13, 0, 0, 0, 0, time.UTC)
	got := prevBusinessDays(7, ref)
	want := []time.Time{
		time.Date(2024, 5, 2, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 5, 3, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 5, 6, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 5, 7, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 5, 8, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 5, 9, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 5, 10, 0, 0, 0, 0, time.UTC),
	}
	if len(got) != len(want) {
		t.Fatalf("expected %d days, got %d", len(want), len(got))
	}
	for i := range want {
		if !got[i].Equal(want[i]) {
			t.Fatalf("day %d = %s, want %s", i, got[i].Format("2006-01-02"), want[i].Format("2006-01-02"))
		}
	}
}

func TestFetchDayZip(t *testing.T) {
	data := zipBytes(map[string]string{"a.txt": "hello"})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)
	}))
	defer srv.Close()
	orig := b3BaseURL
	b3BaseURL = srv.URL
	defer func() { b3BaseURL = orig }()

	got, err := fetchDayZip(context.Background(), time.Date(2024, 5, 5, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("fetchDayZip error: %v", err)
	}
	if !bytes.Equal(got, data) {
		t.Fatalf("unexpected zip bytes")
	}
}

func TestProcessDay(t *testing.T) {
	body := "DT_NEG;TICKER;PRECO;QUANTIDADE;HORA\n" +
		"2024-05-05;PETR4;10,5;100;12:00:00\n" +
		"2024-05-05;VALE3;20,7;50;13:00:00\n"
	data := zipBytes(map[string]string{"mock.csv": body})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)
	}))
	defer srv.Close()
	orig := b3BaseURL
	b3BaseURL = srv.URL
	defer func() { b3BaseURL = orig }()

	linesCh, errCh, err := processDay(context.Background(), time.Date(2024, 5, 5, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("processDay error: %v", err)
	}
	var lines []string
	for line := range linesCh {
		lines = append(lines, line)
	}
	if err := <-errCh; err != nil {
		t.Fatalf("processDay error: %v", err)
	}
	want := []string{
		"2024-05-05;PETR4;10.5;100;12:00:00",
		"2024-05-05;VALE3;20.7;50;13:00:00",
	}
	if !reflect.DeepEqual(lines, want) {
		t.Fatalf("lines mismatch: %#v", lines)
	}
}

type mockRepo struct {
	dayExists bool
	batches   [][]string
}

func (m *mockRepo) DayExists(ctx context.Context, day string) (bool, error) {
	return m.dayExists, nil
}

func (m *mockRepo) InsertBatch(ctx context.Context, day string, lines []string) error {
	cp := append([]string(nil), lines...)
	m.batches = append(m.batches, cp)
	return nil
}

func TestIngestDayBatches(t *testing.T) {
	var sb bytes.Buffer
	sb.WriteString("DT_NEG;TICKER;PRECO;QUANTIDADE;HORA\n")
	for i := 0; i < 1001; i++ {
		fmt.Fprintf(&sb, "2024-05-05;ABC%d;1,0;1;12:00:00\n", i)
	}
	data := zipBytes(map[string]string{"mock.csv": sb.String()})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)
	}))
	defer srv.Close()
	orig := b3BaseURL
	b3BaseURL = srv.URL
	defer func() { b3BaseURL = orig }()

	repo := &mockRepo{}
	day := time.Date(2024, 5, 5, 0, 0, 0, 0, time.UTC)
	if err := ingestDay(context.Background(), repo, day); err != nil {
		t.Fatalf("ingestDay error: %v", err)
	}
	if len(repo.batches) != 2 {
		t.Fatalf("expected 2 batches, got %d", len(repo.batches))
	}
	if len(repo.batches[0]) != 1000 || len(repo.batches[1]) != 1 {
		t.Fatalf("unexpected batch sizes: %d, %d", len(repo.batches[0]), len(repo.batches[1]))
	}
	if repo.batches[0][0] != "2024-05-05;ABC0;1.0;1;12:00:00" {
		t.Fatalf("first line mismatch: %s", repo.batches[0][0])
	}
	if repo.batches[1][0] != "2024-05-05;ABC1000;1.0;1;12:00:00" {
		t.Fatalf("last line mismatch: %s", repo.batches[1][0])
	}
}
