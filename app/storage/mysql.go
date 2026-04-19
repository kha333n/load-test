package storage

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"github.com/kha333n/load-test/app/metrics"
)

type MySQL struct {
	DB   *sql.DB
	pod  string
	node string
}

func NewMySQL(dsn, pod, node string) (*MySQL, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	for {
		if err := db.PingContext(ctx); err == nil {
			break
		}
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("mysql ping timeout: %w", ctx.Err())
		case <-time.After(time.Second):
		}
	}
	return &MySQL{DB: db, pod: pod, node: node}, nil
}

func (m *MySQL) Close() error { return m.DB.Close() }

func (m *MySQL) InUse() int { return m.DB.Stats().InUse }

func (m *MySQL) Migrate(ctx context.Context) error {
	_, err := m.DB.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS items (
			id INT PRIMARY KEY,
			payload VARCHAR(512) NOT NULL,
			hit_count INT NOT NULL DEFAULT 0,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`)
	return err
}

func (m *MySQL) Seed(ctx context.Context, n int) error {
	var count int
	if err := m.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM items").Scan(&count); err != nil {
		return err
	}
	if count >= n {
		return nil
	}

	tx, err := m.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, "INSERT IGNORE INTO items (id, payload, hit_count) VALUES (?, ?, 0)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for i := 1; i <= n; i++ {
		buf := make([]byte, 64)
		_, _ = rand.Read(buf)
		payload := hex.EncodeToString(buf)
		if _, err := stmt.ExecContext(ctx, i, payload); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// SelectByID runs a SELECT and records latency.
func (m *MySQL) SelectByID(ctx context.Context, endpoint string, id int) (string, int, error) {
	start := time.Now()
	var payload string
	var hits int
	err := m.DB.QueryRowContext(ctx, "SELECT payload, hit_count FROM items WHERE id = ?", id).Scan(&payload, &hits)
	metrics.ObserveMySQL(endpoint, "select", time.Since(start))
	return payload, hits, err
}

// IncrHit runs an UPDATE incrementing hit_count and records latency.
func (m *MySQL) IncrHit(ctx context.Context, endpoint string, id int) error {
	start := time.Now()
	_, err := m.DB.ExecContext(ctx, "UPDATE items SET hit_count = hit_count + 1, updated_at = NOW() WHERE id = ?", id)
	metrics.ObserveMySQL(endpoint, "update", time.Since(start))
	return err
}
