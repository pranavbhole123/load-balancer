package server

import (
	"fmt"
	"net/http"
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
	return http.ListenAndServe(
		fmt.Sprintf(":%d", s.port),
		s.handler,
	)
}
