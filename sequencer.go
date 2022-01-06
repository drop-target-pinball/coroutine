package coroutine

import "time"

type Sequencer struct {
	ops    []interface{}
	event  interface{}
	defers []func()
	closed bool
}

type opFn struct {
	fn func()
}

type opLoop struct {
	n int
}

type opSleep struct {
	d time.Duration
}

type opWaitFor struct {
	keys []interface{}
}

type opWaitForUntil struct {
	d    time.Duration
	keys []interface{}
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

func (s *Sequencer) Defer(fn func()) {
	s.defers = append(s.defers, fn)
}

func (s *Sequencer) Do(fn func()) {
	s.checkClosed()
	s.ops = append(s.ops, opFn{fn})
}

func (s *Sequencer) Loop() {
	s.checkClosed()
	s.ops = append(s.ops, opLoop{-1})
	s.closed = true
}

func (s *Sequencer) LoopN(n int) {
	s.checkClosed()
	s.ops = append(s.ops, opLoop{n})
	s.closed = true
}

func (s *Sequencer) Event() interface{} {
	return s.event
}

func (s *Sequencer) Sleep(d time.Duration) {
	s.checkClosed()
	s.ops = append(s.ops, opSleep{d})
}

func (s *Sequencer) WaitFor(keys ...interface{}) {
	s.checkClosed()
	s.ops = append(s.ops, opWaitFor{keys})
}

func (s *Sequencer) WaitForUntil(d time.Duration, keys ...interface{}) {
	s.checkClosed()
	s.ops = append(s.ops, opWaitForUntil{d, keys})
}

func (s *Sequencer) Run(co *C) bool {
	defer func() {
		for _, fn := range s.defers {
			fn()
		}
	}()

	pc := 0 // program counter index into ops slice
	loopN := false
	n := 0
	for {
		if pc >= len(s.ops) {
			break
		}
		operation := s.ops[pc]
		switch op := operation.(type) {
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
				return true
			}
		case opFn:
			op.fn()
		case opWaitFor:
			event, done := co.WaitFor(op.keys...)
			if done {
				return true
			}
			s.event = event
		case opWaitForUntil:
			event, done := co.WaitForUntil(op.d, op.keys...)
			if done {
				return true
			}
			s.event = event
		}
		pc += 1
	}
	return false
}
