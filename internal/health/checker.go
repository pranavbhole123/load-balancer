package health

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

type Checker struct {
	backends []string
	active   []bool
	mu       sync.RWMutex
	interval time.Duration
	client   *http.Client
}

// / we also need a constructor
func NewChecker(backends []string, interval time.Duration, Timeout time.Duration) *Checker {
	// first create a client
	// at first assume all backends active then the first ealth check updates the status
	client := &http.Client{
		Timeout: Timeout, // give up after 3 seconds
	}
	active := make([]bool, len(backends))
	for i := range active {
		active[i] = true // assume healthy at start
	}
	return &Checker{
		backends: backends,
		active:   active,
		interval: interval,
		client:   client,
	}
}

func (c *Checker) checkBackend(i int, url string) {
	resp, err := c.client.Get(url)
	if err != nil {
		c.mu.Lock()
		c.active[i] = false
		log.Printf("backend at index %d dead", i)
		c.mu.Unlock()
		return
	}
	defer resp.Body.Close()

	c.mu.Lock()
	c.active[i] = resp.StatusCode >= 200 && resp.StatusCode < 300
	c.mu.Unlock()
}

func (c *Checker) helper() {
	// we have client and all now we need to ping the backedns

	for i, b := range c.backends {

		c.checkBackend(i, b)
	}
}

func (c *Checker) Start(ctx context.Context) {
	// loops with time .ticker and pings backends
	//updates the backend status
	// exits when the context is cancelled

	// first of all we need a ticker
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	//now we ahve the ticker

	for {
		select {
		case <-ticker.C:
			// we need to ping all the backends for that we will makw a helper function
			c.helper()

		case <-ctx.Done():
			// we need to quit the process thus we can return
			return
		}
	}

}

func (c *Checker) IsActive(idx int) bool {
	c.mu.RLock()
	// we just need to read
	ans := c.active[idx]

	c.mu.RUnlock()
	return ans
}

// package health

func (c *Checker) ExpandByWeights(backends []string, weights []int) error {
	if len(backends) != len(weights) {
		return fmt.Errorf("backends and weights length mismatch")
	}

	var expandedBackends []string
	var expandedActive []bool

	c.mu.Lock()
	defer c.mu.Unlock()

	for i, b := range backends {
		w := weights[i]
		if w <= 0 {
			return fmt.Errorf("invalid weight for backend %q: %d", b, w)
		}

		for j := 0; j < w; j++ {
			expandedBackends = append(expandedBackends, b)
			expandedActive = append(expandedActive, true)
		}
	}

	c.backends = expandedBackends
	c.active = expandedActive

	return nil
}
