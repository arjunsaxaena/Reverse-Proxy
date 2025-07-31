package proxy

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"reverse-proxy/pkg/logger"
	"sync"
	"sync/atomic"
	"time"
)

type ReverseProxy struct {
	backends []*url.URL
	counter  uint64

	healthMu sync.RWMutex
	healthy  map[string]bool // backend URL string -> health status
}

func NewReverseProxy(targets []string) *ReverseProxy {
	var urls []*url.URL
	for _, target := range targets {
		parsed, err := url.Parse(target)
		if err != nil {
			logger.Fatal("Invalid backend URL:", target)
		}
		urls = append(urls, parsed)
	}

	rp := &ReverseProxy{
		backends: urls,
		healthy:   make(map[string]bool),
	}

	for _, u := range urls {
		rp.healthy[u.String()] = false
	}

	go rp.healthCheckLoop()

	return rp
}

func (rp *ReverseProxy) healthCheckLoop() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		rp.checkAllBackends()
		<-ticker.C
	}
}

func (rp *ReverseProxy) checkAllBackends() {
	for _, backend := range rp.backends {
		healthURL := *backend
		healthURL.Path = "/health"
		go rp.checkBackend(backend.String(), healthURL.String())
	}
}

func (rp *ReverseProxy) checkBackend(backendKey, healthURL string) {
	resp, err := http.Get(healthURL)
	healthy := err == nil && resp.StatusCode == http.StatusOK
	if resp != nil {
		resp.Body.Close()
	}

	rp.healthMu.Lock()
	prev := rp.healthy[backendKey]
	rp.healthy[backendKey] = healthy
	rp.healthMu.Unlock()

	if healthy && !prev {
		logger.Info("Backend became healthy:", backendKey)
	} else if !healthy && prev {
		logger.Info("Backend became unhealthy:", backendKey)
	}
}

func (rp *ReverseProxy) getHealthyBackends() []*url.URL {
	rp.healthMu.RLock()
	defer rp.healthMu.RUnlock()
	var healthy []*url.URL
	for _, u := range rp.backends {
		if rp.healthy[u.String()] {
			healthy = append(healthy, u)
		}
	}
	return healthy
}

func (rp *ReverseProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	healthyBackends := rp.getHealthyBackends()
	if len(healthyBackends) == 0 {
		http.Error(w, "No healthy backends available", http.StatusServiceUnavailable)
		return
	}
	index := atomic.AddUint64(&rp.counter, 1)
	target := healthyBackends[int(index)%len(healthyBackends)]

	logger.Info("Forwarding request to:", target)

	proxy := httputil.NewSingleHostReverseProxy(target)

	proxy.ModifyResponse = func(resp *http.Response) error {
		return nil
	}

	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		http.Error(w, "Proxy error: "+err.Error(), http.StatusBadGateway)
	}

	proxy.ServeHTTP(w, r)
}
