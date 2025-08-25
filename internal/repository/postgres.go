package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"

	"desafiocotacaob3/internal/config"
)

type PostgresRepository struct {
	db *sql.DB
}

func NewPostgres(cfg *config.Config) (*PostgresRepository, error) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName)
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	repo := &PostgresRepository{db: db}
	if err := repo.migrate(context.Background()); err != nil {
		return nil, err
	}
	return repo, nil
}

func (r *PostgresRepository) migrate(ctx context.Context) error {
	const stmt = `CREATE TABLE IF NOT EXISTS quotes (
        id UUID PRIMARY KEY,
        date DATE NOT NULL,
        ticker TEXT NOT NULL,
        price NUMERIC NOT NULL,
        quantity NUMERIC NOT NULL,
        time TIME NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_quotes_ticker ON quotes (ticker);
CREATE INDEX IF NOT EXISTS idx_quotes_date ON quotes (date);`
	_, err := r.db.ExecContext(ctx, stmt)
	return err
}

func parseLine(line string) (ticker string, price float64, qty float64, t time.Time, ok bool, err error) {
	parts := strings.Split(line, ";")
	if len(parts) < 5 {
		return "", 0, 0, time.Time{}, false, nil
	}
	ticker = parts[1]
	var priceStr, qtyStr, timeStr string
	if len(parts) >= 6 {
		priceStr = parts[3]
		qtyStr = parts[4]
		timeStr = parts[5]
	} else {
		priceStr = parts[2]
		qtyStr = parts[3]
		timeStr = parts[4]
	}

	priceStr = strings.ReplaceAll(priceStr, ",", ".")
	if price, err = strconv.ParseFloat(priceStr, 64); err != nil {
		return "", 0, 0, time.Time{}, false, err
	}
	qtyStr = strings.ReplaceAll(qtyStr, ",", ".")
	if qty, err = strconv.ParseFloat(qtyStr, 64); err != nil {
		return "", 0, 0, time.Time{}, false, err
	}
	timeStr = strings.ReplaceAll(timeStr, ":", "")
	if len(timeStr) > 6 {
		timeStr = timeStr[:6]
	}
	if t, err = time.Parse("150405", timeStr); err != nil {
		return "", 0, 0, time.Time{}, false, err
	}
	return ticker, price, qty, t, true, nil
}
func (r *PostgresRepository) DayExists(ctx context.Context, day string) (bool, error) {
	const query = "SELECT EXISTS (SELECT 1 FROM quotes WHERE date = $1)"
	var exists bool
	if err := r.db.QueryRowContext(ctx, query, day).Scan(&exists); err != nil {
		return false, err
	}
	return exists, nil
}
func (r *PostgresRepository) InsertBatch(ctx context.Context, day string, lines []string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	stmt, err := tx.PrepareContext(ctx, `INSERT INTO quotes (id, date, ticker, price, quantity, time) VALUES ($1, $2, $3, $4, $5, $6) ON CONFLICT (id) DO NOTHING`)
	if err != nil {
		tx.Rollback()
		return err
	}
	defer stmt.Close()

	for _, line := range lines {
		ticker, price, qty, t, ok, err := parseLine(line)
		if err != nil {
			tx.Rollback()
			return err
		}
		if !ok {
			continue
		}

		id := uuid.New()
		if _, err := stmt.ExecContext(ctx, id, day, ticker, price, qty, t); err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}
func (r *PostgresRepository) QuoteSummary(ctx context.Context, ticker string, startDate time.Time) (float64, int64, bool, error) {
	condition := "WHERE ticker = $1"
	args := []any{ticker}
	if !startDate.IsZero() {
		condition += fmt.Sprintf(" AND date >= $%d", len(args)+1)
		args = append(args, startDate)
	}
	queryPrice := fmt.Sprintf("SELECT COALESCE(MAX(price), 0), COUNT(*) FROM quotes %s", condition)
	var maxPrice float64
	var count int64
	if err := r.db.QueryRowContext(ctx, queryPrice, args...).Scan(&maxPrice, &count); err != nil {
		return 0, 0, false, err
	}
	if count == 0 {
		return 0, 0, false, nil
	}
	queryVolume := fmt.Sprintf(
		"SELECT COALESCE(MAX(sum_qty), 0) FROM (SELECT date, SUM(quantity) AS sum_qty FROM quotes %s GROUP BY date) t",
		condition,
	)
	var maxDailyVolume int64
	if err := r.db.QueryRowContext(ctx, queryVolume, args...).Scan(&maxDailyVolume); err != nil {
		return 0, 0, false, err
	}
	return maxPrice, maxDailyVolume, true, nil
}
