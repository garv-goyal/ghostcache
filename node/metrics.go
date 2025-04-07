package node

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	CacheHits = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "ghostcache_cache_hits_total",
			Help: "Total number of cache hits",
		},
	)
	CacheMisses = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "ghostcache_cache_misses_total",
			Help: "Total number of cache misses",
		},
	)
	TTLUpdates = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "ghostcache_ttl_updates_total",
			Help: "Total number of TTL updates",
		},
	)
	TotalKeys = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "ghostcache_total_keys",
			Help: "Current number of keys in the cache",
		},
	)
)

func init() {
	prometheus.MustRegister(CacheHits, CacheMisses, TTLUpdates, TotalKeys)
}
