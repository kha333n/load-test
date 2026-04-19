package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	mrand "math/rand/v2"
	"net/http"
	"time"

	"github.com/kha333n/load-test/app/storage"
)

func CacheHit(rc *storage.Redis) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key := fmt.Sprintf("cache:hit:%d", mrand.IntN(1000)+1)
		val, outcome, err := rc.Get(r.Context(), "/test/cache-hit", key)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		if outcome != "hit" {
			http.Error(w, "expected hit, got "+outcome, http.StatusInternalServerError)
			return
		}
		w.Write([]byte(val[:32]))
	}
}

func CacheMiss(rc *storage.Redis) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		nonce := make([]byte, 16)
		_, _ = rand.Read(nonce)
		key := "cache:miss:" + hex.EncodeToString(nonce)

		_, _, err := rc.Get(r.Context(), "/test/cache-miss", key)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}

		payload := make([]byte, 512)
		_, _ = rand.Read(payload)
		val := hex.EncodeToString(payload)

		if err := rc.Set(r.Context(), "/test/cache-miss", key, val, 60*time.Second); err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		w.Write([]byte("ok"))
	}
}
