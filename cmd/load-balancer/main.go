package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/pranavbhole123/load-balancer/internal/balancer"
	"github.com/pranavbhole123/load-balancer/internal/health"
	"github.com/pranavbhole123/load-balancer/internal/middleware"
	"github.com/pranavbhole123/load-balancer/internal/parser"
	"github.com/pranavbhole123/load-balancer/internal/proxy"
	"github.com/pranavbhole123/load-balancer/internal/server"
)

func helperChoose(algo string, urls []string, weights []int, checker *health.Checker,  transport *http.Transport) (proxy.Balancer, error) {
	switch algo {
	case "round-robin":
		a, b := balancer.NewRoundRobin(urls,checker,transport)
		return a, b
	case "least-connection":
		a, b := balancer.NewLeastConn(urls, checker,transport)
		return a, b
	case "weighted":
		a, b := balancer.NewWeighted(urls, weights, checker,transport)
		return a, b

	default:
		return nil, fmt.Errorf("please enter valid algorith name %q", algo)
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

	// now we need to customixe main
	// firs twe need to see which type of algo and make a balancer accoringly
	// we will make a fucntion fot this

	
	check := health.NewChecker(urls, 
    time.Duration(cfg.HealthInterval) * time.Second,
    time.Duration(cfg.HealthTimeout) * time.Second,
	)


	transport := &http.Transport{
	MaxIdleConns:        300,
    MaxIdleConnsPerHost: 50,
    MaxConnsPerHost:     100,
    IdleConnTimeout:     90 * time.Second,
	}

	balance, err := helperChoose(cfg.Algorithm, urls, weights , check,transport)

	if err != nil {
		log.Fatal(err)
	}



	ratelimiter := middleware.NewRateLimiter(cfg.RateBurst  ,cfg.RateLimit)

	proxy := proxy.New(balance)

	// now we need to launch a go routine which will do background checks
	//first we make the context 

	handler := ratelimiter.Wrap(proxy)
	ctx , cancel := context.WithCancel(context.Background())

	
	
	
	serv := server.New(cfg.Port, handler)


	go check.Start(ctx)

	defer cancel()
	log.Fatal(serv.Start())


}
