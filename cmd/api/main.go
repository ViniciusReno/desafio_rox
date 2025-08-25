package main

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	"desafiocotacaob3/internal/config"
	"desafiocotacaob3/internal/repository"
	"desafiocotacaob3/internal/util"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load configuration")
	}

	repo, err := repository.NewPostgres(cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to database")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/quotes/summary", quotesSummaryHandler(repo))

	addr := ":" + cfg.APIPort
	log.Info().Msgf("API running on port %s", cfg.APIPort)
	if err := http.ListenAndServe(addr, gzipMiddleware(mux)); err != nil {
		log.Fatal().Err(err).Msg("failed to start server")
	}
}

func gzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			w.Header().Set("Content-Encoding", "gzip")
			gz := gzip.NewWriter(w)
			defer gz.Close()
			gw := gzipResponseWriter{ResponseWriter: w, Writer: gz}
			next.ServeHTTP(gw, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}

type gzipResponseWriter struct {
	http.ResponseWriter
	io.Writer
}

func (w gzipResponseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

type apiError struct {
	ID      string `json:"id"`
	Message string `json:"message"`
}

var (
	errMissingTicker  = apiError{ID: "ERR_MISSING_TICKER", Message: "ticker query param is required"}
	errInvalidDate    = apiError{ID: "ERR_INVALID_DATE", Message: "invalid date_start format"}
	errTickerNotFound = apiError{ID: "ERR_TICKER_NOT_FOUND", Message: "ticker not found"}
)

type summaryResponse struct {
	Ticker         string  `json:"ticker"`
	MaxRangeValue  float64 `json:"max_range_value"`
	MaxDailyVolume int64   `json:"max_daily_volume"`
}

type quoteSummaryRepo interface {
	QuoteSummary(ctx context.Context, ticker string, startDate time.Time) (float64, int64, bool, error)
}

func writeError(w http.ResponseWriter, status int, e apiError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(e)
}

func quotesSummaryHandler(repo quoteSummaryRepo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ticker := strings.ToUpper(r.URL.Query().Get("ticker"))
		if ticker == "" {
			writeError(w, http.StatusBadRequest, errMissingTicker)
			return
		}

		var startDate time.Time
		if ds := r.URL.Query().Get("date_start"); ds != "" {
			var err error
			startDate, err = time.Parse("2006-01-02", ds)
			if err != nil {
				writeError(w, http.StatusBadRequest, errInvalidDate)
				return
			}
		} else {
			startDate = util.BusinessDaysAgo(time.Now().UTC(), 7)
		}

		maxPrice, maxVolume, ok, err := repo.QuoteSummary(r.Context(), ticker, startDate)
		if err != nil {
			writeError(w, http.StatusInternalServerError, apiError{ID: "ERR_INTERNAL", Message: err.Error()})
			return
		}
		if !ok {
			writeError(w, http.StatusNotFound, errTickerNotFound)
			return
		}

		summary := summaryResponse{Ticker: ticker, MaxRangeValue: maxPrice, MaxDailyVolume: maxVolume}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(summary)
	}
}
