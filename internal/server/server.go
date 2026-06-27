package server

import (
	"fmt"
	"net/http"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Server struct {
	port    int
	handler http.Handler
	// add rest necessary things we need
}

// functions
func New(por int, h http.Handler) *Server {
	// constructor for the new server object i guess we dont need anythins else
	return &Server{
		port:    por,
		handler: h,
	}
}

// rest we need the start method
func (s *Server) Start() error {
	mux := http.NewServeMux()

	mux.Handle("/metrics",promhttp.Handler())
	mux.Handle("/", s.handler)
	return http.ListenAndServe(fmt.Sprintf(":%d", s.port), mux)
}
