package proxy

import (
	"net/http"
	"net/http/httputil"
	"time"

	"github.com/pranavbhole123/load-balancer/internal/metrics"
)

type Proxy struct {
	balancer Balancer
	metrics  *metrics.Recorder
}

type Balancer interface {
	Next(r *http.Request) (*httputil.ReverseProxy, string, func())
}

// now we need to define the new responseWrite

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

// override the method

func (r *responseWriter) WriteHeader(code int) {
	r.statusCode = code
	r.ResponseWriter.WriteHeader(code)
}

func New(balanc Balancer, recorder *metrics.Recorder) *Proxy {

	return &Proxy{
		balancer: balanc,
		metrics: recorder,
	}
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	target, backend, done := p.balancer.Next(r)
	if target == nil {
		http.Error(w, "no backends available", http.StatusServiceUnavailable)
		return
	}
	defer done()
	defer p.metrics.TrackActive(backend)() // increment now, decrement when request finishes

	rw := &responseWriter{ResponseWriter: w, statusCode: 200}
	timer := time.Now()
	target.ServeHTTP(rw, r)

	p.metrics.Record(backend, rw.statusCode, time.Since(timer))

}
