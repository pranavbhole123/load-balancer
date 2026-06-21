package balancer

import (
	"fmt"
	"net/http/httputil"
	"net/url"
	"sync/atomic"
)

type RoundRobin struct {
	// what all things we need inside this struct
	targets []*httputil.ReverseProxy
	current uint64
}

// now Round robin need a constructor and a next method
func NewRoundRobin(backends []string) (*RoundRobin, error) {
	rr := &RoundRobin{}
	// keep current 0 at start 

	rr.current = 0

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
	idx := atomic.AddUint64(&r.current , 1) % uint64(len(r.targets))

	return r.targets[idx] , func(){}

}
