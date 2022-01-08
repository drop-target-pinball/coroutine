package main

import (
	"time"

	"github.com/drop-target-pinball/coroutine"
)

func displayA(co *coroutine.C) {
	for {
		if done := co.Sleep(1 * time.Second); done {
			return
		}
		println("A")
	}
}

func displayB(co *coroutine.C) {
	for {
		if done := co.Sleep(2 * time.Second); done {
			return
		}
		println("B")
	}
}

func display(co *coroutine.C) {
	co.New(displayA)
	co.New(displayB)

	co.Sleep(10 * time.Second)
	println("done")
}

func main() {
	ticker := time.NewTicker(16670 * time.Microsecond) // 60 fps

	println("start")
	coroutine.New(display)
	for {
		<-ticker.C
		coroutine.Tick()
	}
}
