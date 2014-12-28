package stats

import (
	"sync"
	"time"

	"github.com/arjantop/saola/stats"
	"github.com/cactus/go-statsd-client/statsd"
)

type StatsdStatsReceiver struct {
	client       statsd.Statter
	rate         float32
	scope        string
	counters     map[string]stats.Counter
	countersLock *sync.Mutex
	timers       map[string]stats.Timer
	timersLock   *sync.Mutex
	scopes       map[string]*StatsdStatsReceiver
	scopesLock   *sync.Mutex
}

func NewStatsdStatsReceiver(client statsd.Statter, rate float32) *StatsdStatsReceiver {
	return newScopedStatsdStatsReceiver(client, rate, "")
}

func newScopedStatsdStatsReceiver(client statsd.Statter, rate float32, scope string) *StatsdStatsReceiver {
	return &StatsdStatsReceiver{
		client:       client,
		rate:         rate,
		scope:        scope,
		counters:     make(map[string]stats.Counter),
		countersLock: new(sync.Mutex),
		timers:       make(map[string]stats.Timer),
		timersLock:   new(sync.Mutex),
		scopes:       make(map[string]*StatsdStatsReceiver),
		scopesLock:   new(sync.Mutex),
	}
}

func (r *StatsdStatsReceiver) Counter(name string) stats.Counter {
	r.countersLock.Lock()
	defer r.countersLock.Unlock()
	if c, ok := r.counters[name]; ok {
		return c
	}
	c := &statsdCounter{stats.ScopedName(r.scope, name), r.client, r.rate}
	r.counters[name] = c
	return c
}

func (r *StatsdStatsReceiver) Timer(name string) stats.Timer {
	r.timersLock.Lock()
	defer r.timersLock.Unlock()
	if t, ok := r.timers[name]; ok {
		return t
	}
	t := &statsdTimer{stats.ScopedName(r.scope, name), r.client, r.rate}
	r.timers[name] = t
	return t
}

func (r *StatsdStatsReceiver) Scope(scope string) stats.StatsReceiver {
	r.scopesLock.Lock()
	defer r.scopesLock.Unlock()
	if s, ok := r.scopes[scope]; ok {
		return s
	}
	s := newScopedStatsdStatsReceiver(r.client, r.rate, stats.ScopedName(r.scope, scope))
	r.scopes[scope] = s
	return s
}

type statsdCounter struct {
	name   string
	client statsd.Statter
	rate   float32
}

func (c *statsdCounter) Incr() {
	c.Add(1)
}

func (c *statsdCounter) Add(delta int64) {
	c.client.Inc(c.name, delta, c.rate)
}

type statsdTimer struct {
	name   string
	client statsd.Statter
	rate   float32
}

func (t *statsdTimer) Add(d time.Duration) {
	t.client.Timing(t.name, int64(d)/int64(time.Millisecond), t.rate)
}
