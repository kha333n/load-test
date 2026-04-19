package handlers

import (
	mrand "math/rand/v2"
	"net/http"
	"strconv"

	"github.com/kha333n/load-test/app/storage"
)

const seedRows = 10000

func DBRead(m *storage.MySQL) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := mrand.IntN(seedRows) + 1
		_, hits, err := m.SelectByID(r.Context(), "/test/db-read", id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		w.Write([]byte(strconv.Itoa(hits)))
	}
}

func DBWrite(m *storage.MySQL) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := mrand.IntN(seedRows) + 1
		if _, _, err := m.SelectByID(r.Context(), "/test/db-write", id); err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		if err := m.IncrHit(r.Context(), "/test/db-write", id); err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		w.Write([]byte("ok"))
	}
}
