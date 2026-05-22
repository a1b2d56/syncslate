package middleware

import (
	"net/http"
	"strings"
	"sync"

	"golang.org/x/time/rate"
)

type IPRateLimiter struct {
	ips sync.Map
	r   rate.Limit
	b   int
}

func NewIPRateLimiter(r rate.Limit, b int) *IPRateLimiter {
	return &IPRateLimiter{
		r: r,
		b: b,
	}
}

func (i *IPRateLimiter) GetLimiter(ip string) *rate.Limiter {
	limiter, exists := i.ips.Load(ip)
	if !exists {
		l := rate.NewLimiter(i.r, i.b)
		i.ips.Store(ip, l)
		return l
	}
	return limiter.(*rate.Limiter)
}

func RateLimit(rpm int) func(http.Handler) http.Handler {
	limiter := NewIPRateLimiter(rate.Limit(float64(rpm)/60.0), rpm)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := getIP(r)
			l := limiter.GetLimiter(ip)
			if !l.Allow() {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Retry-After", "60")
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte(`{"error":{"code":"RATE_LIMITED","message":"too many requests"}}`))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func getIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	parts := strings.Split(r.RemoteAddr, ":")
	return parts[0]
}
