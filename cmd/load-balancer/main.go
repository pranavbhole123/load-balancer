// cmd/load-balancer/main.go
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pranavbhole123/load-balancer/internal/balancer"
	"github.com/pranavbhole123/load-balancer/internal/health"
	"github.com/pranavbhole123/load-balancer/internal/metrics"
	"github.com/pranavbhole123/load-balancer/internal/middleware"
	"github.com/pranavbhole123/load-balancer/internal/parser"
	"github.com/pranavbhole123/load-balancer/internal/proxy"
	"github.com/pranavbhole123/load-balancer/internal/server"
)

const ConsistentHashNodes = 150

func helperChoose(algo string, urls []string, weights []int, checker *health.Checker, transport *http.Transport) (proxy.Balancer, error) {
	switch algo {
	case "round-robin":
		a, b := balancer.NewRoundRobin(urls, checker, transport)
		return a, b
	case "least-connection":
		a, b := balancer.NewLeastConn(urls, checker, transport)
		return a, b
	case "weighted":
		a, b := balancer.NewWeighted(urls, weights, checker, transport)
		return a, b
	case "consistent-hash":
		a, b := balancer.NewConsistentHash(urls, checker, ConsistentHashNodes, transport)
		return a, b
	default:
		return nil, fmt.Errorf("please enter valid algorithm name %q", algo)
	}
}

func main() {
	cfg, err := parser.Load("config.yaml")
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	urls := make([]string, len(cfg.Backends))
	for i, b := range cfg.Backends {
		urls[i] = b.URL
	}

	weights := make([]int, len(cfg.Backends))
	for i, b := range cfg.Backends {
		weights[i] = b.Weight
	}

	check := health.NewChecker(urls,
		time.Duration(cfg.HealthInterval)*time.Second,
		time.Duration(cfg.HealthTimeout)*time.Second,
	)

	transport := &http.Transport{
		MaxIdleConns:        300,
		MaxIdleConnsPerHost: 50,
		MaxConnsPerHost:     100,
		IdleConnTimeout:     90 * time.Second,
	}

	balance, err := helperChoose(cfg.Algorithm, urls, weights, check, transport)
	if err != nil {
		log.Fatal(err)
	}

	ratelimiter := middleware.NewRateLimiter(cfg.RateBurst, cfg.RateLimit)
	recorder := metrics.NewRecorder()
	prox := proxy.New(balance, recorder)
	handler := ratelimiter.Wrap(prox)

	// health checker context — cancelled on shutdown
	healthCtx, healthCancel := context.WithCancel(context.Background())
	go check.Start(healthCtx)

	serv := server.New(cfg.Port, handler)

	// run server in goroutine
	go func() {
		log.Printf("starting load balancer on port %d", cfg.Port)
		if err := serv.Start(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	// block until SIGTERM or SIGINT
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	<-quit

	log.Println("shutdown signal received — draining in-flight requests...")

	// give in-flight requests 30 seconds to complete
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	healthCancel() // stop health checker

	if err := serv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("forced shutdown: %v", err)
	}

	log.Println("shutdown complete")
}