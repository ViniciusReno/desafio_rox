package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"desafiocotacaob3/internal/util"
)

type stubSummaryRepo struct {
	lastTicker string
	lastStart  time.Time
	maxPrice   float64
	maxVolume  int64
	ok         bool
	err        error
}

func (s *stubSummaryRepo) QuoteSummary(ctx context.Context, ticker string, startDate time.Time) (float64, int64, bool, error) {
	s.lastTicker = ticker
	s.lastStart = startDate
	return s.maxPrice, s.maxVolume, s.ok, s.err
}

func TestQuotesSummaryTickerNotFound(t *testing.T) {
	repo := &stubSummaryRepo{ok: false}
	srv := httptest.NewServer(quotesSummaryHandler(repo))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/quotes/summary?ticker=XXXX")
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", resp.StatusCode)
	}

	var e apiError
	if err := json.NewDecoder(resp.Body).Decode(&e); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if e.ID != errTickerNotFound.ID {
		t.Fatalf("expected error %s, got %s", errTickerNotFound.ID, e.ID)
	}
}

func TestQuotesSummaryInvalidDate(t *testing.T) {
	srv := httptest.NewServer(quotesSummaryHandler(nil))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/quotes/summary?ticker=ABEV&date_start=2024-13-01")
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", resp.StatusCode)
	}

	var e apiError
	if err := json.NewDecoder(resp.Body).Decode(&e); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if e.ID != errInvalidDate.ID {
		t.Fatalf("expected error %s, got %s", errInvalidDate.ID, e.ID)
	}
}

func TestQuotesSummaryDefaultDate(t *testing.T) {
	repo := &stubSummaryRepo{maxPrice: 10.5, maxVolume: 1000, ok: true}
	srv := httptest.NewServer(quotesSummaryHandler(repo))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/quotes/summary?ticker=PETR4")
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	var sResp summaryResponse
	if err := json.NewDecoder(resp.Body).Decode(&sResp); err != nil {
		t.Fatalf("decode: %v", err)
	}

	expectedStart := util.BusinessDaysAgo(time.Now().UTC(), 7)
	if !repo.lastStart.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, repo.lastStart)
	}
	if sResp.Ticker != "PETR4" {
		t.Fatalf("unexpected ticker %s", sResp.Ticker)
	}
	if sResp.MaxRangeValue != 10.5 || sResp.MaxDailyVolume != 1000 {
		t.Fatalf("unexpected summary values %+v", sResp)
	}
}
