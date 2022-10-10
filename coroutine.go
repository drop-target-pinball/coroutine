package coroutine

import (
	"time"

	"github.com/benbjohnson/clock"
)

type Event interface {
	Key() interface{}
}

type Group struct {
	clock  clock.Clock
	active []*C
	queue  []Event
}

type C struct {
	group      *Group
	children   []*C
	yield      chan request
	resume     chan response
	requesting request
}

type request struct {
	valid   bool
	cancel  bool
	expires time.Time
	events  []Event
}

type response struct {
	timeout bool
	cancel  bool
	event   Event
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
		children: make([]*C, 0),
		yield:    make(chan request),
		resume:   make(chan response),
	}
	g.add(co)

	cancelFunc := func() {
		// Cancel the outstanding request. This will get cleaned up on the
		// next call to Tick
		co.cancel()

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
		children: make([]*C, 0),
		yield:    make(chan request),
		resume:   make(chan response),
	}
	c.group.add(co)
	c.children = append(c.children, co)
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

func (c *C) WaitFor(events ...Event) (Event, bool) {
	c.yield <- request{valid: true, events: events}
	return c.waitForResume()
}

func (c *C) WaitForUntil(d time.Duration, events ...Event) (Event, bool) {
	expires := c.group.clock.Now().Add(d)
	c.yield <- request{valid: true, expires: expires, events: events}
	return c.waitForResume()
}

func (c *C) waitForResume() (Event, bool) {
	response := <-c.resume
	if response.cancel {
		return nil, true
	}
	if response.timeout {
		return nil, false
	}
	return response.event, false
}

func (c *C) cancel() {
	c.requesting.cancel = true
	for _, co := range c.children {
		co.cancel()
	}
}

func (g *Group) Post(evt Event) {
	g.queue = append(g.queue, evt)
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
		var evt Event
		evt, g.queue = g.queue[0], g.queue[1:]

		for _, co := range g.active {
			if co == nil {
				continue
			}
			// Resume if the requested event key matches
			for _, evtReq := range co.requesting.events {
				if evt.Key() == evtReq.Key() {
					co.resume <- response{event: evt}
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

	// Slide active coroutines down
	i := 0
	for _, co := range g.active {
		if co != nil {
			g.active[i] = co
			i++
		}
	}
	for ; i < len(g.active); i++ {
		g.active[i] = nil
	}
}

func (g *Group) Stop() {
	for _, co := range g.active {
		co.cancel()
	}
	g.Tick()
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

func Post(evt Event) {
	group.Post(evt)
}

func Tick() {
	group.Tick()
}

type testEvent string

func (e testEvent) Key() interface{} {
	return e
}
