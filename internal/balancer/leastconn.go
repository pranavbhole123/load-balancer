package balancer

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"

	"github.com/pranavbhole123/load-balancer/internal/health"
)

type Leastconn struct {
	// what all things should it have
	targets []*httputil.ReverseProxy
	counts  []int64
	mu      sync.Mutex
	checker *health.Checker
}

// now we need a constructo

func NewLeastConn(backends []string, checker *health.Checker, transport *http.Transport) (*Leastconn, error) {
	l := &Leastconn{}
	l.checker = checker

	for _, b := range backends {
		remote, err := url.Parse(b)
		if err != nil {
			return nil, fmt.Errorf(
				"invalid backend url %q: %w",
				b,
				err,
			)
		}

		temp := httputil.NewSingleHostReverseProxy(remote)
		temp.Transport = transport
		l.targets = append(
			l.targets,
			temp,
		)
		l.counts = append(l.counts, 0)
	}

	return l, nil
}

func (l *Leastconn) Next() (*httputil.ReverseProxy, func()) {
	l.mu.Lock()
	defer l.mu.Unlock()

	idx := -1
	mini := int64(-1)

	for i := range l.targets {
		if !l.checker.IsActive(i) {
			continue
		}
		if mini == -1 || l.counts[i] < mini {
			mini = l.counts[i]
			idx = i
		}
	}

	if idx == -1 {
		return nil, func() {}
	}

	l.counts[idx]++
	return l.targets[idx], func() {
		l.mu.Lock()
		l.counts[idx]--
		l.mu.Unlock()
	}
}
