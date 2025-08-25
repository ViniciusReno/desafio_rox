package main

import (
	"archive/zip"
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	"desafiocotacaob3/internal/config"
	"desafiocotacaob3/internal/repository"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load configuration")
	}
	log.Info().Msgf("Ingest running with DB %s", cfg.DBName)

	repo, err := repository.NewPostgres(cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to database")
	}

	ctx := context.Background()
	processed := make(map[string]struct{})
	run := func() {
		days := prevBusinessDays(7, time.Now())
		for _, day := range days {
			dayStr := day.Format("2006-01-02")
			if _, ok := processed[dayStr]; ok {
				continue
			}
			if err := ingestDay(ctx, repo, day); err != nil {
				log.Error().Err(err).Msgf("failed to ingest %s", dayStr)
				continue
			}
			processed[dayStr] = struct{}{}
		}
	}

	run()
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()
	for range ticker.C {
		run()
	}
}

func prevBusinessDay(t time.Time) time.Time {
	d := t.AddDate(0, 0, -1)
	for d.Weekday() == time.Saturday || d.Weekday() == time.Sunday {
		d = d.AddDate(0, 0, -1)
	}
	return d
}

func prevBusinessDays(n int, ref time.Time) []time.Time {
	days := make([]time.Time, 0, n)
	d := ref
	for len(days) < n {
		d = prevBusinessDay(d)
		days = append(days, d)
	}
	for i, j := 0, len(days)-1; i < j; i, j = i+1, j-1 {
		days[i], days[j] = days[j], days[i]
	}
	return days
}

type inserter interface {
	DayExists(ctx context.Context, day string) (bool, error)
	InsertBatch(ctx context.Context, day string, lines []string) error
}

func ingestDay(ctx context.Context, repo inserter, day time.Time) error {
	dayStr := day.Format("2006-01-02")
	exists, err := repo.DayExists(ctx, dayStr)
	if err != nil {
		return err
	}
	if exists {
		log.Info().Msgf("data for %s already ingested", dayStr)
		return nil
	}

	return retry(3, func() error {
		linesCh, errCh, err := processDay(ctx, day)
		if err != nil {
			return err
		}
		const batchSize = 1000
		batch := make([]string, 0, batchSize)
		for line := range linesCh {
			batch = append(batch, line)
			if len(batch) >= batchSize {
				if err := repo.InsertBatch(ctx, dayStr, batch); err != nil {
					return err
				}
				batch = batch[:0]
			}
		}
		if len(batch) > 0 {
			if err := repo.InsertBatch(ctx, dayStr, batch); err != nil {
				return err
			}
		}
		if err := <-errCh; err != nil {
			return err
		}
		return nil
	})
}

var b3BaseURL = "https://arquivos.b3.com.br/rapinegocios/tickercsv"

func fetchDayZip(ctx context.Context, day time.Time) ([]byte, error) {
	_ = ctx
	url := fmt.Sprintf("%s/%s", b3BaseURL, day.Format("2006-01-02"))
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %s", resp.Status)
	}
	return io.ReadAll(resp.Body)
}

func processDay(ctx context.Context, day time.Time) (<-chan string, <-chan error, error) {
	zipBytes, err := fetchDayZip(ctx, day)
	if err != nil {
		return nil, nil, err
	}
	r, err := zip.NewReader(bytes.NewReader(zipBytes), int64(len(zipBytes)))
	if err != nil {
		return nil, nil, err
	}

	lines := make(chan string)
	errCh := make(chan error, 1)
	go func() {
		defer close(lines)
		defer close(errCh)

		for _, f := range r.File {
			rc, err := f.Open()
			if err != nil {
				errCh <- err
				return
			}
			scanner := bufio.NewScanner(rc)
			scanner.Buffer(make([]byte, 1024), 10*1024*1024)
			first := true
			for scanner.Scan() {
				if first {
					first = false
					continue
				}
				parts := strings.Split(scanner.Text(), ";")
				if len(parts) < 5 {
					continue
				}
				parts[2] = strings.ReplaceAll(parts[2], ",", ".")
				select {
				case lines <- strings.Join(parts, ";"):
				case <-ctx.Done():
					rc.Close()
					errCh <- ctx.Err()
					return
				}
			}
			rc.Close()
			if err := scanner.Err(); err != nil {
				if errors.Is(err, bufio.ErrTooLong) {
					errCh <- fmt.Errorf("scanner buffer overflow: %w", err)
				} else {
					errCh <- err
				}
				return
			}
		}
		errCh <- nil
	}()

	return lines, errCh, nil
}

func retry(times int, fn func() error) error {
	var err error
	for i := 0; i < times; i++ {
		if err = fn(); err == nil {
			return nil
		}
		time.Sleep(time.Second)
	}
	return err
}
