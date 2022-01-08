package coroutine

import (
	"testing"
	"time"
)

func TestLoop(t *testing.T) {
	wd := NewWatchdog(1 * time.Second)
	g := NewGroup()
	a := 0

	cancel := g.NewCoroutine(func(co *C) {
		s := NewSequencer()

		s.WaitFor(testEvent("event"))
		s.Do(func() { a += 1 })
		s.Loop()
		s.Run(co)
	})

	g.Post(testEvent("event"))
	g.Post(testEvent("event"))
	g.Post(testEvent("event"))
	g.Tick()
	cancel()
	g.Post(testEvent("event"))
	g.Tick()

	if a != 3 {
		t.Errorf("\n have: %v \n want: %v", a, 3)
	}
	running := g.running()
	if running != 0 {
		t.Errorf("\n have: %v \n want: %v", running, 0)
	}

	cancel()
	wd.Stop()
}

func TestLoopN(t *testing.T) {
	wd := NewWatchdog(1 * time.Second)
	g := NewGroup()
	a := 0

	cancel := g.NewCoroutine(func(co *C) {
		s := NewSequencer()

		s.WaitFor(testEvent("event"))
		s.Do(func() { a += 1 })
		s.LoopN(2)
		s.Run(co)
	})

	g.Post(testEvent("event"))
	g.Post(testEvent("event"))
	g.Post(testEvent("event"))
	g.Post(testEvent("event"))
	g.Tick()

	if a != 3 {
		t.Errorf("\n have: %v \n want: %v", a, 3)
	}
	running := g.running()
	if running != 0 {
		t.Errorf("\n have: %v \n want: %v", running, 0)
	}

	cancel()
	wd.Stop()
}

func TestDefer(t *testing.T) {
	wd := NewWatchdog(1 * time.Second)
	g := NewGroup()
	a := 0
	b := 0

	cancel := g.NewCoroutine(func(co *C) {
		s := NewSequencer()

		s.Defer(func() { b = 1 })
		s.WaitFor(testEvent("event"))
		s.Do(func() { a += 1 })
		s.Loop()
		s.Run(co)
	})

	g.Post(testEvent("event"))
	g.Post(testEvent("event"))
	g.Post(testEvent("event"))
	g.Tick()
	cancel()
	g.Post(testEvent("event"))
	g.Tick()

	if a != 3 {
		t.Errorf("\n have: %v \n want: %v", a, 3)
	}
	running := g.running()
	if running != 0 {
		t.Errorf("\n have: %v \n want: %v", running, 0)
	}
	if b != 1 {
		t.Errorf("\n have: %v \n want: %v", b, 1)
	}

	cancel()
	wd.Stop()
}

func TestClosed(t *testing.T) {
	wd := NewWatchdog(1 * time.Second)
	g := NewGroup()
	a := 0

	cancel := g.NewCoroutine(func(co *C) {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("expecting panic")
			}
		}()

		s := NewSequencer()

		s.WaitFor(testEvent("event"))
		s.Do(func() { a += 1 })
		s.Loop()
		s.Do(func() { a = 99 })
		s.Run(co)
	})

	cancel()
	wd.Stop()
}

func TestDoCancel(t *testing.T) {
	wd := NewWatchdog(1 * time.Second)
	g := NewGroup()
	a := 0
	b := 0

	cancel := g.NewCoroutine(func(co *C) {
		s := NewSequencer()

		s.WaitFor(testEvent("event"))
		s.Do(func() { a = 1 })
		s.Cancel(func() { b = 1 })
		s.WaitFor(testEvent("event"))
		s.Do(func() { a = 2 })
		s.Run(co)
	})

	g.Post(testEvent("event"))
	g.Tick()
	cancel()
	g.Post(testEvent("event"))
	g.Tick()

	if a != 1 {
		t.Errorf("\n have: %v \n want: %v", a, 1)
	}
	if b != 1 {
		t.Errorf("\n have: %v \n want: %v", b, 1)
	}
	running := g.running()
	if running != 0 {
		t.Errorf("\n have: %v \n want: %v", running, 0)
	}

	cancel()
	wd.Stop()
}
