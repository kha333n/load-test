package handlers

import (
	"net/http"
	"time"

	"github.com/kha333n/load-test/app/storage"
)

func Health(m *storage.MySQL, r *storage.Redis) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		ctx, cancel := contextWithTimeout(req, 2*time.Second)
		defer cancel()

		if err := m.DB.PingContext(ctx); err != nil {
			http.Error(w, "mysql: "+err.Error(), http.StatusServiceUnavailable)
			return
		}
		if err := r.Ping(ctx); err != nil {
			http.Error(w, "redis: "+err.Error(), http.StatusServiceUnavailable)
			return
		}
		w.Write([]byte("ok"))
	}
}
