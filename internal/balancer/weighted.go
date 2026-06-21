package balancer

import (
	"fmt"
	"net/http/httputil"
	"net/url"
)

// think we dont even need a new struct as it same a round robin

func NewWeighted(backends []string, weights []int) (*RoundRobin, error) {

	// now what we need to do is to expand the backends

	rr := &RoundRobin{}
	// keep current 0 at start

	rr.current = 0

	for j, b := range backends {
		remote, err := url.Parse(b)
		if err != nil {
			return nil, fmt.Errorf(
				"invalid backend url %q: %w",
				b,
				err,
			)
		}
		proxy := httputil.NewSingleHostReverseProxy(remote)
		for i := 0; i < weights[j]; i++ {
    		rr.targets = append(rr.targets, proxy)  // same pointer, repeated
		}
	}

	return rr, nil
}
