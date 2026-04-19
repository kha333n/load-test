package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/kha333n/load-test/app/metrics"
)

type endpointKey struct{}

func WithEndpoint(ctx context.Context, name string) context.Context {
	return context.WithValue(ctx, endpointKey{}, name)
}

func GetEndpoint(ctx context.Context) string {
	if v, ok := ctx.Value(endpointKey{}).(string); ok {
		return v
	}
	return ""
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

func Timing(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		endpoint := chi.RouteContext(r.Context()).RoutePattern()
		ctx := WithEndpoint(r.Context(), endpoint)

		metrics.Inflight.WithLabelValues(metrics.Pod(), metrics.Node()).Inc()
		defer metrics.Inflight.WithLabelValues(metrics.Pod(), metrics.Node()).Dec()

		rec := &statusRecorder{ResponseWriter: w, status: 200}
		start := time.Now()
		next.ServeHTTP(rec, r.WithContext(ctx))
		metrics.ObserveHTTP(endpoint, rec.status, time.Since(start))
	})
}
