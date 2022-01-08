package main

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/drop-target-pinball/coroutine"
)

type event string

func (e event) Key() interface{} {
	return e
}

var playfieldSwitches = []event{
	event("standup target #1"),
	event("standup target #2"),
	event("standup target #3"),
	event("left sling"),
	event("right sling"),
}

var drainSwitches = []event{
	event("left outlane"),
	event("right outlane"),
	event("down the middle"),
}

var (
	score    int
	bonus    int
	kickback bool
	gameOver bool
)

func awardScore(val int) {
	score += val
	fmt.Printf("*** awarded %v points, score %v\n", val, score)
}

func randomSwitchEvent(co *coroutine.C) {
	fmt.Println("(+) randomSwitchEvent: start")
	defer fmt.Println("(-) randomSwitchEvent: done")

	for {
		d := time.Duration(rand.Intn(3000)+500) * time.Millisecond
		if done := co.Sleep(d); done {
			return
		}
		i := rand.Intn(len(playfieldSwitches))
		sw := playfieldSwitches[i]
		fmt.Printf("switch: %v\n", sw)
		coroutine.Post(playfieldSwitches[i])
	}
}

func randomDrainEvent(co *coroutine.C) {
	fmt.Println("(+) randomDrainEvent: start")
	defer fmt.Println("(-) randomDrainEvent: done")

	drainChance := 1
	for {
		if done := co.Sleep(3 * time.Second); done {
			return
		}
		roll := rand.Intn(100)
		if roll < drainChance {
			i := rand.Intn(len(drainSwitches))
			sw := drainSwitches[i]
			fmt.Printf("switch: %v\n", sw)
			coroutine.Post(drainSwitches[i])
			drainChance = drainChance / 2
		} else {
			drainChance += 1
		}
	}
}

func watchSlings(co *coroutine.C) {
	fmt.Println("(+) watchSlings: start")
	defer fmt.Println("(-) watchSlings: done")

	s := coroutine.NewSequencer()
	s.WaitFor(event("left sling"), event("right sling"))
	s.Do(func() { awardScore(10) })
	s.Loop()
	s.Run(co)
}

func watchStandups(co *coroutine.C) {
	fmt.Println("(+) watchStandups: start")
	defer fmt.Println("(-) watchStandups: done")

	nLit := 0
	lit := map[event]bool{
		event("standup target #1"): false,
		event("standup target #2"): false,
		event("standup target #3"): false,
	}

	for {
		evt, done := co.WaitFor(
			event("standup target #1"),
			event("standup target #2"),
			event("standup target #3"),
		)
		if done {
			return
		}
		sw := evt.(event)
		if lit[sw] {
			awardScore(25)
			bonus += 10
		} else {
			awardScore(100)
			bonus += 10
			lit[sw] = true
			nLit += 1
			if nLit == 3 {
				if !kickback {
					fmt.Println("*** kickback lit")
					kickback = true
				}
				for sw := range lit {
					lit[sw] = false
				}
				nLit = 0
			}
		}
	}
}

func watchExits(co *coroutine.C) {
	fmt.Println("(+) watchExits: start")
	defer fmt.Println("(-) watchExits: done")

	drained := false
	for !drained {
		evt, done := co.WaitFor(
			event("left outlane"),
			event("right outlane"),
			event("down the middle"),
		)
		if done {
			return
		}
		switch evt {
		case event("left outlane"):
			awardScore(125)
			if kickback {
				fmt.Println("*** kickback")
				kickback = false
			} else {
				drained = true
			}
		case event("right outlane"):
			awardScore(125)
			drained = true
		case event("down the middle"):
			drained = true
		}
	}
	fmt.Println("ball drained")
	coroutine.Post(event("ball drained"))
}

func basicMode(co *coroutine.C) {
	fmt.Println("(+) basicMode: start")
	defer fmt.Println("(-) basicMode: done")

	co.New(randomSwitchEvent)
	co.New(randomDrainEvent)
	co.New(watchSlings)
	co.New(watchStandups)
	co.New(watchExits)

	co.WaitFor(event("ball drained"))
	fmt.Println("end of ball")
	coroutine.Post(event("end of ball"))
}

func gameMode(co *coroutine.C) {
	fmt.Println("(+) gameMode: start")
	defer fmt.Println("(-) gameMode: done")

	coroutine.New(basicMode)
	co.WaitFor(event("end of ball"))

	totalScore := score + bonus

	s := coroutine.NewSequencer()
	s.Sleep(1 * time.Second)
	s.Do(func() { fmt.Printf("score: %v\n", score) })
	s.Sleep(1 * time.Second)
	s.Do(func() { fmt.Printf("bonus: %v\n", bonus) })
	s.Sleep(1 * time.Second)
	s.Do(func() { fmt.Printf("total score: %v\n", totalScore) })
	s.Sleep(2 * time.Second)
	s.Do(func() {
		fmt.Println("game over")
		gameOver = true
	})
	s.Run(co)
}

func main() {
	wd := coroutine.NewWatchdog(1 * time.Second)
	coroutine.New(gameMode)
	for !gameOver {
		time.Sleep(50 * time.Millisecond)
		coroutine.Tick()
		wd.Reset()
	}
}
