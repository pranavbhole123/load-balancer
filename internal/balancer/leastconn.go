package balancer

import (
	"fmt"
	"log"
	"net/http/httputil"
	"net/url"
	"sync"
)

type Leastconn struct {
	// what all things should it have
	targets []*httputil.ReverseProxy
	counts  []int64
	mu      sync.Mutex
}

// now we need a constructo

func NewLeastConn(backends []string) (*Leastconn, error) {
	l := &Leastconn{}

	for _, b := range backends {
		remote, err := url.Parse(b)
		if err != nil {
			return nil, fmt.Errorf(
				"invalid backend url %q: %w",
				b,
				err,
			)
		}

		l.targets = append(
			l.targets,
			httputil.NewSingleHostReverseProxy(remote),
		)
		l.counts = append(l.counts, 0)
	}

	return l, nil
}

func (l *Leastconn) Next() (*httputil.ReverseProxy, func()) {
	l.mu.Lock()

	mini := l.counts[0]
	idx := 0
	for i, val := range l.counts {
		if val < mini {
			idx = i
			mini = val
		}
	}
	l.counts[idx]++
	log.Printf("picked backend %d", idx)

	l.mu.Unlock() // unlock before returning, not deferred

	return l.targets[idx], func() {
		l.mu.Lock()
		l.counts[idx]--
		l.mu.Unlock()
	}
}
