package coroutine

import (
	"testing"
	"time"
)

func TestSleep(t *testing.T) {
	wd := NewWatchdog(1 * time.Second)
	g, clk := newMockGroup()
	a := 0
	cancel := g.NewCoroutine(func(co *C) {
		s := NewSequencer()

		s.Sleep(1000 * time.Millisecond)
		s.Do(func() { a = 1 })

		s.Sleep(1000 * time.Millisecond)
		s.Do(func() { a = 2 })

		s.Run(co)
	})

	clk.Add(1500 * time.Millisecond)
	g.Tick()
	if a != 1 {
		t.Errorf("\n have: %v \n want: %v", a, 1)
	}
	running := g.running()
	if running != 1 {
		t.Errorf("\n have: %v \n want: %v", running, 1)
	}

	clk.Add(1500 * time.Millisecond)
	g.Tick()
	if a != 2 {
		t.Errorf("\n have: %v \n want: %v", a, 2)
	}
	running = g.running()
	if g.running() != 0 {
		t.Errorf("\n have: %v \n want: %v", running, 0)
	}
	cancel()
	wd.Stop()
}

func TestWaitFor(t *testing.T) {
	wd := NewWatchdog(1 * time.Second)
	g := NewGroup()
	a := 0
	cancel := g.NewCoroutine(func(co *C) {
		s := NewSequencer()

		s.WaitFor("event")
		s.Do(func() { a = s.Event().(int) })

		s.WaitFor("event")
		s.Do(func() { a = s.Event().(int) })

		s.Run(co)
	})

	g.PostWith("event", 1)
	g.Tick()
	if a != 1 {
		t.Errorf("\n have: %v \n want: %v", a, 1)
	}
	running := g.running()
	if running != 1 {
		t.Errorf("\n have: %v \n want: %v", running, 1)
	}

	g.PostWith("event", 2)
	g.Tick()
	if a != 2 {
		t.Errorf("\n have: %v \n want: %v", a, 2)
	}
	running = g.running()
	if g.running() != 0 {
		t.Errorf("\n have: %v \n want: %v", running, 0)
	}
	cancel()
	wd.Stop()
}

func TestWaitForUntil(t *testing.T) {
	wd := NewWatchdog(1 * time.Second)
	g, clk := newMockGroup()
	a := 0
	cancel := g.NewCoroutine(func(co *C) {
		s := NewSequencer()

		s.WaitForUntil(1000*time.Millisecond, "event")
		s.Do(func() { a = 1 })

		s.WaitForUntil(1000*time.Millisecond, "event")
		s.Do(func() { a = 2 })

		s.Run(co)
	})

	g.Post("event")
	g.Tick()
	if a != 1 {
		t.Errorf("\n have: %v \n want: %v", a, 1)
	}
	running := g.running()
	if running != 1 {
		t.Errorf("\n have: %v \n want: %v", running, 1)
	}

	clk.Add(1500 * time.Millisecond)
	g.Tick()
	if a != 2 {
		t.Errorf("\n have: %v \n want: %v", a, 2)
	}
	running = g.running()
	if g.running() != 0 {
		t.Errorf("\n have: %v \n want: %v", running, 0)
	}
	cancel()
	wd.Stop()
}

func TestCancel(t *testing.T) {
	wd := NewWatchdog(1 * time.Second)
	g := NewGroup()
	a := 0
	cancel := g.NewCoroutine(func(co *C) {
		s := NewSequencer()

		s.WaitFor("event")
		s.Do(func() { a = s.Event().(int) })

		s.WaitFor("event")
		s.Do(func() { a = s.Event().(int) })

		s.Run(co)
	})

	g.PostWith("event", 1)
	g.Tick()
	if a != 1 {
		t.Errorf("\n have: %v \n want: %v", a, 1)
	}
	running := g.running()
	if running != 1 {
		t.Errorf("\n have: %v \n want: %v", running, 1)
	}
	cancel()

	g.PostWith("event", 2)
	g.Tick()
	if a != 1 {
		t.Errorf("\n have: %v \n want: %v", a, 1)
	}
	running = g.running()
	if g.running() != 0 {
		t.Errorf("\n have: %v \n want: %v", running, 0)
	}
	cancel()
	wd.Stop()
}

func TestMulti(t *testing.T) {
	wd := NewWatchdog(1 * time.Second)
	g := NewGroup()
	a := 0
	b := 0
	aCancel := g.NewCoroutine(func(co *C) {
		s := NewSequencer()

		s.WaitFor("event1")
		s.Do(func() { a = s.Event().(int) })

		s.WaitFor("event1")
		s.Do(func() { a = s.Event().(int) })

		s.Run(co)
	})
	bCancel := g.NewCoroutine(func(co *C) {
		s := NewSequencer()

		s.WaitFor("event2")
		s.Do(func() { b = s.Event().(int) })

		s.WaitFor("event2")
		s.Do(func() { b = s.Event().(int) })

		s.Run(co)
	})

	g.PostWith("event1", 1)
	g.PostWith("event2", 10)
	g.Tick()
	if a != 1 {
		t.Errorf("\n have: %v \n want: %v", a, 1)
	}
	if b != 10 {
		t.Errorf("\n have: %v \n want: %v", b, 10)
	}
	running := g.running()
	if running != 2 {
		t.Errorf("\n have: %v \n want: %v", running, 2)
	}

	g.PostWith("event1", 2)
	g.PostWith("event2", 20)
	g.Tick()
	if a != 2 {
		t.Errorf("\n have: %v \n want: %v", a, 2)
	}
	if b != 20 {
		t.Errorf("\n have: %v \n want: %v", b, 20)
	}
	running = g.running()
	if running != 0 {
		t.Errorf("\n have: %v \n want: %v", running, 0)
	}

	aCancel()
	bCancel()
	wd.Stop()
}

func TestSub(t *testing.T) {
	wd := NewWatchdog(1 * time.Second)
	g := NewGroup()
	a := 0
	b := 0
	cancel := g.NewCoroutine(func(co *C) {
		co.New(func(co *C) {
			s := NewSequencer()

			s.WaitFor("event2")
			s.Do(func() { b = s.Event().(int) })

			s.WaitFor("event2")
			s.Do(func() { b = s.Event().(int) })
			s.Run(co)
		})

		s := NewSequencer()
		s.WaitFor("event1")
		s.Do(func() { a = s.Event().(int) })
		s.Run(co)
	})

	g.PostWith("event2", 10)
	g.Tick()
	if a != 0 {
		t.Errorf("\n have: %v \n want: %v", a, 0)
	}
	if b != 10 {
		t.Errorf("\n have: %v \n want: %v", b, 10)
	}
	running := g.running()
	if running != 2 {
		t.Errorf("\n have: %v \n want: %v", running, 2)
	}

	g.PostWith("event2", 20)
	g.Tick()
	if a != 0 {
		t.Errorf("\n have: %v \n want: %v", a, 0)
	}
	if b != 20 {
		t.Errorf("\n have: %v \n want: %v", b, 20)
	}
	running = g.running()
	if running != 1 {
		t.Errorf("\n have: %v \n want: %v", running, 1)
	}

	g.PostWith("event1", 1)
	g.Tick()
	if a != 1 {
		t.Errorf("\n have: %v \n want: %v", a, 1)
	}
	if b != 20 {
		t.Errorf("\n have: %v \n want: %v", b, 20)
	}
	running = g.running()
	if running != 0 {
		t.Errorf("\n have: %v \n want: %v", running, 0)
	}

	cancel()
	wd.Stop()
}

func TestSubCancel(t *testing.T) {
	wd := NewWatchdog(1 * time.Second)
	g := NewGroup()
	a := 0
	b := 0
	aDef := 0
	bDef := 0

	cancel := g.NewCoroutine(func(co *C) {
		co.New(func(co *C) {
			s := NewSequencer()

			s.Defer(func() { bDef = 1 })
			s.Do(func() { b = 1 })
			s.WaitFor("event2")
			s.Do(func() { b = 2 })
			s.Run(co)
		})

		s := NewSequencer()
		s.Defer(func() { aDef = 1 })
		s.Do(func() { a = 1 })
		s.WaitFor("event1")
		s.Do(func() { a = 2 })
		s.Run(co)
	})

	g.Tick()
	cancel()
	g.Post("event1")
	g.Post("event2")
	g.Tick()

	if a != 1 {
		t.Errorf("\n have: %v \n want: %v", a, 1)
	}
	if b != 1 {
		t.Errorf("\n have: %v \n want: %v", b, 1)
	}
	if aDef != 1 {
		t.Errorf("\n have: %v \n want: %v", aDef, 1)
	}
	if bDef != 1 {
		t.Errorf("\n have: %v \n want: %v", bDef, 1)
	}
	running := g.running()
	if running != 0 {
		t.Errorf("\n have: %v \n want: %v", running, 0)
	}

	cancel()
	wd.Stop()
}
