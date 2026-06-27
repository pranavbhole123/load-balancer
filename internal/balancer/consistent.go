package balancer

import (
	"fmt"
	"hash/fnv"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sort"
	"sync"

	"github.com/pranavbhole123/load-balancer/internal/health"
)

type point struct {
	hash uint32
	idx  int
}

type ConsistentHash struct {
	backends     []string
	ring         []point
	targets      []*httputil.ReverseProxy
	virtualNodes int
	mu           sync.RWMutex
	checker      *health.Checker
}

func hash(key string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(key))
	return h.Sum32()
}

func (c *ConsistentHash) buildRing() {
	// the build ring hashes url#i  for each virtual node i of backend creating N different hash per backend

	for i, b := range c.backends {
		// for each backend n nodes
		for j := 0; j < c.virtualNodes; j++ {
			// we make a point
			pt := &point{}
			pt.idx = i
			// now compute the hash or url#j
			str := fmt.Sprintf("%s#%d", b, j)
			pt.hash = hash(str)
			c.ring = append(c.ring, *pt)
		}
	}
	sort.Slice(c.ring, func(i, j int) bool {
		return c.ring[i].hash < c.ring[j].hash
	})

}

func (c *ConsistentHash) Next(r *http.Request) (*httputil.ReverseProxy, string,func()) {
	// now what we do in this function is
	//first find the hash of incoming req ip

	c.mu.RLock()
	defer c.mu.RUnlock()
	h := hash(r.RemoteAddr)
	// now we need to find where it lies inside the array ring  // we need to binary seaarch

	idx := sort.Search(len(c.ring), func(i int) bool {
		return c.ring[i].hash >= h
	})

	if idx == len(c.ring) {
		idx = 0
	}

	//once we find the idx we need to find the real idx of the reverse proxy

	n := len(c.ring)
	for i := 0; i < n; i++ {

		candidate := c.ring[(idx+i)%n]

		if c.checker.IsActive(candidate.idx) {
			log.Printf("picked backend %d", candidate.idx)
			return c.targets[candidate.idx],c.backends[candidate.idx], func() {}
		}
	}
	return nil, "",func() {}
	// but if it is not active we need to find the next one in clockwise direction

}

func NewConsistentHash(backends []string, checker *health.Checker, nodes int, transport *http.Transport) (*ConsistentHash, error) {
	// now we make
	ans := &ConsistentHash{}

	ans.backends = backends
	// now we need to populate the targets
	ans.checker = checker
	ans.virtualNodes = nodes

	// we also need to build a ring

	ans.buildRing()

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
		ans.targets = append(ans.targets, temp)
	}

	// now we also need to build the ring

	return ans, nil

}
