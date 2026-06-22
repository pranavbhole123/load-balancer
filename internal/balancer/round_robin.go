package balancer

import (
	"fmt"
	"log"
	"net/http/httputil"
	"net/url"
	"sync/atomic"

	"github.com/pranavbhole123/load-balancer/internal/health"
)

type RoundRobin struct {
	// what all things we need inside this struct
	targets []*httputil.ReverseProxy
	current uint64
	checker *health.Checker
}

// now Round robin need a constructor and a next method
func NewRoundRobin(backends []string, checker *health.Checker) (*RoundRobin, error) {
	rr := &RoundRobin{}
	// keep current 0 at start

	rr.current = 0
	rr.checker = checker

	for _, b := range backends {
		remote, err := url.Parse(b)
		if err != nil {
			return nil, fmt.Errorf(
				"invalid backend url %q: %w",
				b,
				err,
			)
		}

		rr.targets = append(
			rr.targets,
			httputil.NewSingleHostReverseProxy(remote),
		)
	}

	return rr, nil
}

// also we need a next method

func (r *RoundRobin) Next() (*httputil.ReverseProxy, func()) {

	// now we choose the next element

	for i := 0; i < len(r.targets); i++ {

		idx := atomic.AddUint64(&r.current, 1) % uint64(len(r.targets))
		// now we need to check if the picked element is active
		
		if r.checker.IsActive(int(idx)) {

			log.Printf("picked backend %d", idx)

			return r.targets[idx], func() {}
		}
	}
	return nil, func() {}

}
