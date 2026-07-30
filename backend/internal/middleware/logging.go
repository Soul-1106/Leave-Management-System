package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"net/http"
	"time"
)

type loggingWriter struct {
	http.ResponseWriter
	status int
}

func (writer *loggingWriter) WriteHeader(status int) {
	writer.status = status
	writer.ResponseWriter.WriteHeader(status)
}

func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			value := make([]byte, 8)
			_, _ = rand.Read(value)
			requestID = hex.EncodeToString(value)
		}
		w.Header().Set("X-Request-ID", requestID)
		writer := &loggingWriter{ResponseWriter: w, status: http.StatusOK}
		started := time.Now()
		next.ServeHTTP(writer, r)
		log.Printf("request id=%s method=%s path=%q status=%d duration_ms=%d",
			requestID, r.Method, r.URL.Path, writer.status, time.Since(started).Milliseconds())
	})
}
