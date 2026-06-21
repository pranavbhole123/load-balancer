package main

import (
	"log"

	"github.com/pranavbhole123/load-balancer/internal/parser"
	"github.com/pranavbhole123/load-balancer/internal/proxy"
	"github.com/pranavbhole123/load-balancer/internal/server"
)

func main() {
	cfg, err := parser.Load("config.yaml")
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	urls := make([]string, len(cfg.Backends))
	for i, b := range cfg.Backends {
		urls[i] = b.URL
	}

	/*prox := proxy.New(urls)
	srv := server.New(cfg.Port, prox)
	log.Fatal(srv.Start())/*/
}
