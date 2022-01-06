package coroutine

import (
	"time"

	"github.com/benbjohnson/clock"
)

type queued struct {
	key   interface{}
	event interface{}
}

type Group struct {
	clock  clock.Clock
	active []*C
	queue  []queued
}

type C struct {
	group      *Group
	children   []CancelFunc
	yield      chan request
	resume     chan response
	requesting request
}

type request struct {
	valid   bool
	cancel  bool
	expires time.Time
	keys    []interface{}
}

type response struct {
	timeout bool
	cancel  bool
	key     interface{}
	event   interface{}
}

func NewGroup() *Group {
	return &Group{
		clock:  clock.New(),
		active: make([]*C, 0),
	}
}

func newMockGroup() (*Group, *clock.Mock) {
	g := NewGroup()
	mock := clock.NewMock()
	g.clock = mock
	return g, mock
}

type CancelFunc func()

func (g *Group) NewCoroutine(fn func(*C)) CancelFunc {
	co := &C{
		group:    g,
		children: make([]CancelFunc, 0),
		yield:    make(chan request),
		resume:   make(chan response),
	}
	g.add(co)

	cancelFunc := func() {
		// Cancel the outstanding request. This will get cleaned up on the
		// next call to Tick
		co.requesting.cancel = true

		// All children of this coroutine are also canceled
		for _, cancelChild := range co.children {
			cancelChild()
		}

		// Let the coroutines have a chance to clean up
		g.Tick()
	}

	go func() {
		fn(co)
		close(co.yield)
		cancelFunc()
	}()

	// Let the newly created coroutine reach its first yield
	co.requesting = <-co.yield

	return cancelFunc
}

func (c *C) New(fn func(*C)) {
	co := &C{
		group:    c.group,
		children: make([]CancelFunc, 0),
		yield:    make(chan request),
		resume:   make(chan response),
	}
	c.group.add(co)
	c.children = append(c.children, func() {
		co.requesting.cancel = true
	})
	go func() {
		fn(co)
		close(co.yield)
	}()
	// Let the newly created coroutine reach its first yield
	co.requesting = <-co.yield
}

func (c *C) Sleep(d time.Duration) bool {
	expires := c.group.clock.Now().Add(d)
	c.yield <- request{valid: true, expires: expires}
	_, done := c.waitForResume()
	return done
}

func (c *C) WaitFor(keys ...interface{}) (interface{}, bool) {
	c.yield <- request{valid: true, keys: keys}
	return c.waitForResume()
}

func (c *C) WaitForUntil(d time.Duration, keys ...interface{}) (interface{}, bool) {
	expires := c.group.clock.Now().Add(d)
	c.yield <- request{valid: true, expires: expires, keys: keys}
	return c.waitForResume()
}

func (c *C) waitForResume() (interface{}, bool) {
	response := <-c.resume
	if response.cancel {
		return nil, true
	}
	if response.timeout {
		return nil, false
	}
	return response.event, false
}

func (g *Group) PostWith(key interface{}, event interface{}) {
	g.queue = append(g.queue, queued{key, event})
}

func (g *Group) Post(key interface{}) {
	g.queue = append(g.queue, queued{key, key})
}

func (g *Group) Tick() {
	now := g.clock.Now()
	for i, co := range g.active {
		if co == nil {
			continue
		}

		// If the coroutine has been canceled, let it know so that it can
		// cleanup.
		if co.requesting.valid && co.requesting.cancel {
			co.resume <- response{cancel: true}
			co.requesting = <-co.yield
			g.active[i] = nil
		}

		// Resume if requested timer has expired
		expires := co.requesting.expires
		if !expires.IsZero() && now.After(expires) {
			co.resume <- response{timeout: true}
			co.requesting = <-co.yield
		}
	}

	// Service the queue
	for len(g.queue) > 0 {
		var q queued
		q, g.queue = g.queue[0], g.queue[1:]

		for _, co := range g.active {
			if co == nil {
				continue
			}
			// Resume if the requested event key matches
			for _, keyReq := range co.requesting.keys {
				if q.key == keyReq {
					co.resume <- response{key: q.key, event: q.event}
					co.requesting = <-co.yield
					break
				}
			}
		}
	}

	// Coroutine is no longer active if the yield channel has been closed.
	// When closed, the channel returns a result where
	// requesting.valid has the default value of false
	for i, co := range g.active {
		if co == nil {
			continue
		}
		if !co.requesting.valid {
			g.active[i] = nil
			continue
		}
	}
}

func (g *Group) add(co *C) {
	// When removing from the list, the value in the slice is simply set
	// to nil. When adding, iterate to see if there are any open spaces,
	// and if not, append.
	for i, active := range g.active {
		if active == nil {
			g.active[i] = co
			return
		}
	}
	g.active = append(g.active, co)
}

func (g *Group) running() int {
	n := 0
	for _, active := range g.active {
		if active != nil && active.requesting.valid {
			n++
		}
	}
	return n
}

var group = NewGroup()

func New(fn func(*C)) CancelFunc {
	return group.NewCoroutine(fn)
}

func PostWith(key interface{}, event interface{}) {
	group.PostWith(key, event)
}

func Post(key interface{}) {
	group.Post(key)
}

func Tick() {
	group.Tick()
}
