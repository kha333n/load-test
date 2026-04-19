package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/kha333n/load-test/app/handlers"
	"github.com/kha333n/load-test/app/metrics"
	appmw "github.com/kha333n/load-test/app/middleware"
	"github.com/kha333n/load-test/app/storage"
)

func main() {
	cfg := loadConfig()
	log.Printf("starting load-test app pod=%s node=%s listen=%s", cfg.Pod, cfg.Node, cfg.Listen)

	metrics.Init(cfg.Pod, cfg.Node)

	mysqlCli, err := storage.NewMySQL(cfg.MySQLDSN, cfg.Pod, cfg.Node)
	if err != nil {
		log.Fatalf("mysql: %v", err)
	}
	defer mysqlCli.Close()

	if err := mysqlCli.Migrate(context.Background()); err != nil {
		log.Fatalf("mysql migrate: %v", err)
	}
	if err := mysqlCli.Seed(context.Background(), 10000); err != nil {
		log.Fatalf("mysql seed: %v", err)
	}

	redisCli := storage.NewRedis(cfg.RedisAddr, cfg.Pod, cfg.Node)
	defer redisCli.Close()

	if err := redisCli.WarmCache(context.Background(), 1000); err != nil {
		log.Fatalf("redis warm: %v", err)
	}

	go metrics.PollPoolStats(context.Background(), mysqlCli, redisCli, 5*time.Second)

	r := chi.NewRouter()
	r.Use(appmw.RequestID)

	r.Get("/healthz", handlers.Health(mysqlCli, redisCli))
	r.Get("/metrics", metrics.Handler())

	r.Group(func(r chi.Router) {
		r.Use(appmw.Timing)
		r.Get("/test/compute", handlers.Compute())
		r.Get("/test/cache-hit", handlers.CacheHit(redisCli))
		r.Get("/test/cache-miss", handlers.CacheMiss(redisCli))
		r.Get("/test/db-read", handlers.DBRead(mysqlCli))
		r.Get("/test/db-write", handlers.DBWrite(mysqlCli))
		r.Get("/test/combined", handlers.Combined(redisCli, mysqlCli))
	})

	srv := &http.Server{
		Addr:              cfg.Listen,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("listening on %s", cfg.Listen)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("http: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	log.Println("shutting down")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("shutdown: %v", err)
	}
}

type config struct {
	Listen    string
	MySQLDSN  string
	RedisAddr string
	Pod       string
	Node      string
}

func loadConfig() config {
	return config{
		Listen:    envOr("LISTEN_ADDR", ":8080"),
		MySQLDSN:  envOr("MYSQL_DSN", "root:root@tcp(127.0.0.1:3306)/loadtest?parseTime=true&charset=utf8mb4"),
		RedisAddr: envOr("REDIS_ADDR", "127.0.0.1:6379"),
		Pod:       envOr("POD_NAME", "local"),
		Node:      envOr("NODE_NAME", "local"),
	}
}

func envOr(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
