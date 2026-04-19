package handlers

import (
	"crypto/sha256"
	"net/http"
)

// Compute does a fixed CPU-bound hash loop: SHA-256 over 10KB, 100 iterations.
func Compute() http.HandlerFunc {
	payload := make([]byte, 10*1024)
	for i := range payload {
		payload[i] = byte(i % 251)
	}
	return func(w http.ResponseWriter, r *http.Request) {
		var sum [32]byte
		buf := payload
		for i := 0; i < 100; i++ {
			sum = sha256.Sum256(buf)
			buf = sum[:]
		}
		w.Write(sum[:])
	}
}
