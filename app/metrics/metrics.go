package metrics

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var Buckets = []float64{
	0.0005, 0.001, 0.002, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5,
}

var (
	HTTPDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "End-to-end HTTP request duration.",
			Buckets: Buckets,
		},
		[]string{"endpoint", "pod", "node", "status"},
	)
	HTTPRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total HTTP requests.",
		},
		[]string{"endpoint", "pod", "node", "status"},
	)
	RedisDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "redis_operation_duration_seconds",
			Help:    "Redis operation duration.",
			Buckets: Buckets,
		},
		[]string{"endpoint", "op", "pod", "node", "outcome"},
	)
	MySQLDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "mysql_query_duration_seconds",
			Help:    "MySQL query duration.",
			Buckets: Buckets,
		},
		[]string{"endpoint", "op", "pod", "node"},
	)
	MySQLConns = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "mysql_connections_in_use",
			Help: "MySQL connections currently in use.",
		},
		[]string{"pod", "node"},
	)
	RedisConns = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "redis_connections_in_use",
			Help: "Redis connections currently in use.",
		},
		[]string{"pod", "node"},
	)
	Inflight = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "app_inflight_requests",
			Help: "In-flight HTTP requests.",
		},
		[]string{"pod", "node"},
	)
)

var (
	pod  string
	node string
)

func Init(podName, nodeName string) {
	pod = podName
	node = nodeName
	prometheus.MustRegister(
		HTTPDuration,
		HTTPRequests,
		RedisDuration,
		MySQLDuration,
		MySQLConns,
		RedisConns,
		Inflight,
	)
}

func Pod() string  { return pod }
func Node() string { return node }

func Handler() http.HandlerFunc {
	return promhttp.Handler().ServeHTTP
}

func ObserveHTTP(endpoint string, status int, dur time.Duration) {
	s := strconv.Itoa(status)
	HTTPDuration.WithLabelValues(endpoint, pod, node, s).Observe(dur.Seconds())
	HTTPRequests.WithLabelValues(endpoint, pod, node, s).Inc()
}

func ObserveRedis(endpoint, op, outcome string, dur time.Duration) {
	RedisDuration.WithLabelValues(endpoint, op, pod, node, outcome).Observe(dur.Seconds())
}

func ObserveMySQL(endpoint, op string, dur time.Duration) {
	MySQLDuration.WithLabelValues(endpoint, op, pod, node).Observe(dur.Seconds())
}

type poolStater interface {
	InUse() int
}

func PollPoolStats(ctx context.Context, mysql poolStater, redis poolStater, every time.Duration) {
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			MySQLConns.WithLabelValues(pod, node).Set(float64(mysql.InUse()))
			RedisConns.WithLabelValues(pod, node).Set(float64(redis.InUse()))
		}
	}
}
