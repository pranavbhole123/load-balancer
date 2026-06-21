package proxy

import (
	"net/http"
	"net/http/httputil"
)

type Proxy struct {
	balancer Balancer
}

type Balancer interface {
	Next() (*httputil.ReverseProxy, func())
}

func New(balanc Balancer) *Proxy {

	return &Proxy{
		balancer: balanc,
	}
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	target , done := p.balancer.Next()

    defer done()
    target.ServeHTTP(w,r)
}
