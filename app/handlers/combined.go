package handlers

import (
	"fmt"
	mrand "math/rand/v2"
	"net/http"
	"time"

	"github.com/kha333n/load-test/app/storage"
)

func Combined(rc *storage.Redis, m *storage.MySQL) http.HandlerFunc {
	const ep = "/test/combined"
	return func(w http.ResponseWriter, r *http.Request) {
		id := mrand.IntN(seedRows) + 1
		key := fmt.Sprintf("combined:item:%d", id)

		val, outcome, err := rc.Get(r.Context(), ep, key)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		if outcome == "hit" {
			w.Write([]byte(val[:32]))
			return
		}

		payload, _, err := m.SelectByID(r.Context(), ep, id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		if err := rc.Set(r.Context(), ep, key, payload, 30*time.Second); err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		w.Write([]byte(payload[:32]))
	}
}
