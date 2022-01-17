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

		s.WaitFor(testEvent("event"))
		s.Do(func() { a = 1 })

		s.WaitFor(testEvent("event"))
		s.Do(func() { a = 2 })

		s.Run(co)
	})

	g.Post(testEvent("event"))
	g.Tick()
	if a != 1 {
		t.Errorf("\n have: %v \n want: %v", a, 1)
	}
	running := g.running()
	if running != 1 {
		t.Errorf("\n have: %v \n want: %v", running, 1)
	}

	g.Post(testEvent("event"))
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

		s.WaitForUntil(1000*time.Millisecond, testEvent("event"))
		s.Do(func() { a = 1 })

		s.WaitForUntil(1000*time.Millisecond, testEvent("event"))
		s.Do(func() { a = 2 })

		s.Run(co)
	})

	g.Post(testEvent("event"))
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

		s.WaitFor(testEvent("event"))
		s.Do(func() { a = 1 })

		s.WaitFor(testEvent("event"))
		s.Do(func() { a = 2 })

		s.Run(co)
	})

	g.Post(testEvent("event"))
	g.Tick()
	if a != 1 {
		t.Errorf("\n have: %v \n want: %v", a, 1)
	}
	running := g.running()
	if running != 1 {
		t.Errorf("\n have: %v \n want: %v", running, 1)
	}
	cancel()

	g.Post(testEvent("event"))
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

		s.WaitFor(testEvent("event1"))
		s.Do(func() { a = 1 })

		s.WaitFor(testEvent("event1"))
		s.Do(func() { a = 2 })

		s.Run(co)
	})
	bCancel := g.NewCoroutine(func(co *C) {
		s := NewSequencer()

		s.WaitFor(testEvent("event2"))
		s.Do(func() { b = 10 })

		s.WaitFor(testEvent("event2"))
		s.Do(func() { b = 20 })

		s.Run(co)
	})

	g.Post(testEvent("event1"))
	g.Post(testEvent("event2"))
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

	g.Post(testEvent("event1"))
	g.Post(testEvent("event2"))
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

			s.WaitFor(testEvent("event2"))
			s.Do(func() { b = 10 })

			s.WaitFor(testEvent("event2"))
			s.Do(func() { b = 20 })
			s.Run(co)
		})

		s := NewSequencer()
		s.WaitFor(testEvent("event1"))
		s.Do(func() { a = 1 })
		s.Run(co)
	})

	g.Post(testEvent("event2"))
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

	g.Post(testEvent("event2"))
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

	g.Post(testEvent("event1"))
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
	c := 0
	aDef := 0
	bDef := 0
	cDef := 0

	cancel := g.NewCoroutine(func(co *C) {
		co.New(func(co *C) {
			co.New(func(co *C) {
				s := NewSequencer()
				s.Defer(func() { cDef = 1 })
				s.Do(func() { c = 1 })
				s.WaitFor(testEvent("event3"))
				s.Do(func() { c = 2 })
				s.Run(co)
			})

			s := NewSequencer()
			s.Defer(func() { bDef = 1 })
			s.Do(func() { b = 1 })
			s.WaitFor(testEvent("event2"))
			s.Do(func() { b = 2 })
			s.Run(co)
		})

		s := NewSequencer()
		s.Defer(func() { aDef = 1 })
		s.Do(func() { a = 1 })
		s.WaitFor(testEvent("event1"))
		s.Do(func() { a = 2 })
		s.Run(co)
	})

	g.Tick()
	cancel()
	g.Post(testEvent("event1"))
	g.Post(testEvent("event2"))
	g.Post(testEvent("event3"))
	g.Tick()

	if a != 1 {
		t.Errorf("\n have: %v \n want: %v", a, 1)
	}
	if b != 1 {
		t.Errorf("\n have: %v \n want: %v", b, 1)
	}
	if c != 1 {
		t.Errorf("\n have: %v \n want: %v", c, 1)
	}
	if aDef != 1 {
		t.Errorf("\n have: %v \n want: %v", aDef, 1)
	}
	if bDef != 1 {
		t.Errorf("\n have: %v \n want: %v", bDef, 1)
	}
	if cDef != 1 {
		t.Errorf("\n have: %v \n want: %v", cDef, 1)
	}
	running := g.running()
	if running != 0 {
		t.Errorf("\n have: %v \n want: %v", running, 0)
	}

	cancel()
	wd.Stop()
}

func TestStop(t *testing.T) {
	wd := NewWatchdog(1 * time.Second)
	g := NewGroup()
	a := 0
	cancel := g.NewCoroutine(func(co *C) {
		s := NewSequencer()

		s.WaitFor(testEvent("event"))
		s.Do(func() { a = 1 })

		s.WaitFor(testEvent("event"))
		s.Do(func() { a = 2 })

		s.Run(co)
	})

	g.Post(testEvent("event"))
	g.Tick()
	if a != 1 {
		t.Errorf("\n have: %v \n want: %v", a, 1)
	}
	running := g.running()
	if running != 1 {
		t.Errorf("\n have: %v \n want: %v", running, 1)
	}

	g.Stop()
	g.Post(testEvent("event"))
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
