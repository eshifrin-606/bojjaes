package weekcache

import (
	"context"
	"time"
)

// CurrentWeekTTL is how long the current NFL week is served from memory. The
// week flips once, overnight, so an hour bounds lateness to an hour nobody is
// setting a lineup in.
const CurrentWeekTTL = time.Hour

// Single caches one unkeyed fetch, such as the current NFL week. It is a Cache
// holding one fixed key, so single-flight, the detached fetch, and dropping
// failures are Cache's, not a second implementation of them.
type Single[V any] struct {
	cache *Cache[V]
}

func NewSingle[V any](label string, source func(context.Context) (V, error), ttl time.Duration, opts ...Option) *Single[V] {
	keyed := func(ctx context.Context, _, _ int) (V, error) { return source(ctx) }
	cache := New(label, keyed, ttl, opts...)
	cache.describe = func(int, int) string { return "current" }
	return &Single[V]{cache: cache}
}

// AsOf returns the value and the instant the fetch that produced it completed.
func (s *Single[V]) AsOf(ctx context.Context) (V, time.Time, error) {
	return s.cache.AsOf(ctx, 0, 0)
}
