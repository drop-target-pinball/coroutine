package coroutine

import (
	"time"
)

type Sequencer struct {
	ops    []interface{}
	event  Event
	defers []func()
	closed bool
}

type opCancel struct {
	fn func()
}

type opDo struct {
	fn func()
}

type opDoRun struct {
	fn func() bool
}

type opLoop struct {
	n int
}

type opSleep struct {
	d time.Duration
}

type opWaitFor struct {
	events []Event
}

type opWaitForUntil struct {
	d      time.Duration
	events []Event
}

func NewSequencer() *Sequencer {
	return &Sequencer{
		ops:    make([]interface{}, 0),
		defers: make([]func(), 0),
	}
}

func (s *Sequencer) checkClosed() {
	if s.closed {
		panic("sequence is closed")
	}
}

func (s *Sequencer) Cancel(fn func()) {
	s.ops = append(s.ops, opCancel{fn})
}

func (s *Sequencer) Defer(fn func()) {
	s.defers = append(s.defers, fn)
}

func (s *Sequencer) Do(fn func()) {
	s.checkClosed()
	s.ops = append(s.ops, opDo{fn})
}

func (s *Sequencer) DoRun(fn func() bool) {
	s.checkClosed()
	s.ops = append(s.ops, opDoRun{fn})
}

func (s *Sequencer) Loop() {
	s.checkClosed()
	s.ops = append(s.ops, opLoop{-1})
	s.closed = true
}

func (s *Sequencer) LoopN(n int) {
	s.ops = append(s.ops, opLoop{n})
}

func (s *Sequencer) Event() Event {
	return s.event
}

func (s *Sequencer) Sleep(d time.Duration) {
	s.checkClosed()
	s.ops = append(s.ops, opSleep{d})
}

func (s *Sequencer) WaitFor(events ...Event) {
	s.checkClosed()
	s.ops = append(s.ops, opWaitFor{events})
}

func (s *Sequencer) WaitForUntil(d time.Duration, events ...Event) {
	s.checkClosed()
	s.ops = append(s.ops, opWaitForUntil{d, events})
}

func (s *Sequencer) Run(co *C) bool {
	defer func() {
		for _, fn := range s.defers {
			fn()
		}
	}()

	cancelFuncs := make([]func(), 0)
	cancel := func() {
		for _, fn := range cancelFuncs {
			fn()
		}
	}

	pc := 0 // program counter index into ops slice
	loopN := false
	n := 0
	for {
		if pc >= len(s.ops) {
			break
		}
		operation := s.ops[pc]
		switch op := operation.(type) {
		case opCancel:
			cancelFuncs = append(cancelFuncs, op.fn)
		case opDo:
			op.fn()
		case opDoRun:
			done := op.fn()
			if done {
				cancel()
				return true
			}
			cancelFuncs = nil
			s.event = nil
		case opLoop:
			if op.n < 0 {
				pc = 0
				continue
			}
			if !loopN {
				loopN = true
				n = op.n
			} else {
				n -= 1
			}
			if n > 0 {
				pc = 0
				continue
			}
		case opSleep:
			s.event = nil
			if done := co.Sleep(op.d); done {
				cancel()
				return true
			}
			cancelFuncs = nil
		case opWaitFor:
			event, done := co.WaitFor(op.events...)
			if done {
				cancel()
				return true
			}
			cancelFuncs = nil
			s.event = event
		case opWaitForUntil:
			event, done := co.WaitForUntil(op.d, op.events...)
			if done {
				cancel()
				return true
			}
			cancelFuncs = nil
			s.event = event
		}
		pc += 1
	}
	return false
}
