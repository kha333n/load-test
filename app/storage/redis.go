package storage

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/kha333n/load-test/app/metrics"
)

type Redis struct {
	C    *redis.Client
	pod  string
	node string
}

func NewRedis(addr, pod, node string) *Redis {
	c := redis.NewClient(&redis.Options{
		Addr:         addr,
		PoolSize:     20,
		MinIdleConns: 5,
		DialTimeout:  3 * time.Second,
		ReadTimeout:  2 * time.Second,
		WriteTimeout: 2 * time.Second,
	})
	return &Redis{C: c, pod: pod, node: node}
}

func (r *Redis) Close() error { return r.C.Close() }

func (r *Redis) InUse() int {
	s := r.C.PoolStats()
	return int(s.TotalConns - s.IdleConns)
}

// WarmCache populates cache:hit:{1..n} with ~1KB random payloads.
func (r *Redis) WarmCache(ctx context.Context, n int) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	for {
		if err := r.C.Ping(ctx).Err(); err == nil {
			break
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("redis ping timeout: %w", ctx.Err())
		case <-time.After(time.Second):
		}
	}

	payload := make([]byte, 512)
	_, _ = rand.Read(payload)
	val := hex.EncodeToString(payload) // ~1KB

	pipe := r.C.Pipeline()
	for i := 1; i <= n; i++ {
		pipe.Set(ctx, fmt.Sprintf("cache:hit:%d", i), val, 0)
	}
	_, err := pipe.Exec(ctx)
	return err
}

// Get records latency. outcome is "hit" if found, "miss" if redis.Nil, "err" otherwise.
func (r *Redis) Get(ctx context.Context, endpoint, key string) (string, string, error) {
	start := time.Now()
	v, err := r.C.Get(ctx, key).Result()
	dur := time.Since(start)

	outcome := "hit"
	if errors.Is(err, redis.Nil) {
		outcome = "miss"
	} else if err != nil {
		outcome = "err"
	}
	metrics.ObserveRedis(endpoint, "get", outcome, dur)

	if errors.Is(err, redis.Nil) {
		return "", outcome, nil
	}
	return v, outcome, err
}

// Set records latency.
func (r *Redis) Set(ctx context.Context, endpoint, key, val string, ttl time.Duration) error {
	start := time.Now()
	err := r.C.Set(ctx, key, val, ttl).Err()
	outcome := "ok"
	if err != nil {
		outcome = "err"
	}
	metrics.ObserveRedis(endpoint, "set", outcome, time.Since(start))
	return err
}

// Ping for health checks (not metric-recorded).
func (r *Redis) Ping(ctx context.Context) error {
	return r.C.Ping(ctx).Err()
}
