package middleware

import (
	"net"
	"net/http"
	"sync"

	"golang.org/x/time/rate"
)

// first define the ratelimiter struct

type RateLimiter struct {
	// what all things we need
	mu      sync.Mutex
	clients map[string]*rate.Limiter
	// now we need the
	rps   rate.Limit
	burst int // initial capacity of the bucket

}

// now what all methods we need think

// we need a constructor

func NewRateLimiter(burst int, rps int) *RateLimiter {

	r := &RateLimiter{}

	r.burst = burst
	r.rps = rate.Limit(rps)
	clients := make(map[string]*rate.Limiter)

	r.clients = clients

	return r
}

// think about what more we need
func (r *RateLimiter) Allow(ip string) bool {

	return r.getLimiter(ip).Allow()
}

func (r *RateLimiter) getLimiter(ip string) *rate.Limiter {
	r.mu.Lock()
	limiter, exists := r.clients[ip]

	if !exists {
		limiter = rate.NewLimiter(r.rps, r.burst)
		r.clients[ip] = limiter
	}
	r.mu.Unlock()
	return limiter
}

// we also need the warp method so that we include the ratelimiter as a middleware

func (rl *RateLimiter) Wrap(next http.Handler) http.Handler {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//ip := r.RemoteAddr // this returns along with the port need to process
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err!=nil{
			http.Error(w,"error parsing the ip", http.StatusBadRequest)
			return 
		}
		if !rl.Allow(ip) {
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}
