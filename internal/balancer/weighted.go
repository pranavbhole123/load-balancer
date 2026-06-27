package balancer

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/pranavbhole123/load-balancer/internal/health"
)

// think we dont even need a new struct as it same a round robin





func NewWeighted(backends []string, weights []int, checker *health.Checker, transport *http.Transport ) (*RoundRobin, error) {

	// now what we need to do is to expand the backends

	rr := &RoundRobin{}
	// keep current 0 at start

	// but we need to edit the checker as per weights to procceed

	rr.current = 0
	checker.ExpandByWeights(backends ,weights)
	rr.checker = checker

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
		proxy.Transport = transport
		for i := 0; i < weights[j]; i++ {
    		rr.targets = append(rr.targets, proxy)  // same pointer, repeated
			rr.backend = append(rr.backend, b)
		}
	}

	return rr, nil
}
