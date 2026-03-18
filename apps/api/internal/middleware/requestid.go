package middleware

import (
	"math/rand"
	"net/http"
	"time"

	"github.com/dylangeraci/flowforge/internal/logging"
	"github.com/oklog/ulid/v2"
)

func RequestID(next http.Handler) http.Handler {
	entropy := ulid.Monotonic(rand.New(rand.NewSource(time.Now().UnixNano())), 0)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := ulid.MustNew(ulid.Timestamp(time.Now()), entropy).String()
		ctx := logging.WithRequestID(r.Context(), id)
		w.Header().Set("X-Request-ID", id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
