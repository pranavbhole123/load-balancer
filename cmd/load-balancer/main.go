package main

import (
	"fmt"
	"log"

	"github.com/pranavbhole123/load-balancer/internal/balancer"
	"github.com/pranavbhole123/load-balancer/internal/parser"
	"github.com/pranavbhole123/load-balancer/internal/proxy"
	"github.com/pranavbhole123/load-balancer/internal/server"
)

func helperChoose(algo string , urls []string, weights []int)(proxy.Balancer , error){
	switch algo{
	case "round-robin":
		a , b := balancer.NewRoundRobin(urls)
		return a , b
	case "least-connection":
		a , b := balancer.NewLeastConn(urls)
		return a, b
	case "weighted":
		a , b := balancer.NewWeighted(urls,weights)
		return a,b

	default:
		return nil , fmt.Errorf("please enter valid algorith name %q",algo)
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

	weights := make([]int , len(cfg.Backends))

	for i ,b := range cfg.Backends{
		weights[i] = b.Weight
	}

	// now we need to customixe main 
	// firs twe need to see which type of algo and make a balancer accoringly
	// we will make a fucntion fot this 

	balance, err := helperChoose(cfg.Algorithm,urls,weights)

	
	if err != nil {
    	log.Fatal(err)
	}

	proxy := proxy.New(balance)
	serv := server.New(cfg.Port , proxy)

	log.Fatal(serv.Start())
}	
