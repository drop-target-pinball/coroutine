# coroutine

[![Go Reference](https://pkg.go.dev/badge/github.com/drop-target-pinball/coroutine.svg)](https://pkg.go.dev/github.com/drop-target-pinball/coroutine)

A coroutine implementation based on timers and events.

*Work in progress*

## Overview

This library provides a coroutine implementation that is useful for developing
event-driven applications.

Since this code is being developed for the [Super Pinball
System](https://github.com/drop-target-pinball/spin), we will use an example
based on a pinball game. Let's say we need to write some code to award one
million points if the player shoots the left ramp, shoots the right ramp, and
then shoots the left ramp again. One way to implement this is to create event
listeners for the ramps:


```go
var pos = 0

func LeftRamp() {
    if pos == 0 {
        pos == 1
    } else if pos == 2 {
        pos = 0
        AwardScore(1_000_000)
    }
}

func RightRamp() {
    if pos == 1 {
        pos == 2
    }
}

AddListener(leftRamp)
AddListener(rightRamp)
```

Since go has goroutines and channels, we can flip this around. Instead of
having a function called when an event is received, we can call a function and
wait until an event is received:

```go
WaitFor(event.LeftRamp)
WaitFor(event.RightRamp)
WaitFor(event.LeftRamp)
AwardScore(1_000_000)
```

The latter is easier to read, easier to write, and no longer needs the
variable, `pos`, to keep track of state.

To make this shot sequence into a game mode we should add a timer so the player
only has 30 seconds to complete the sequence and we should add some audio and
video too. Multiple coroutines can be used to separate out this logic so we
could have a coroutine each for:

- sequencing voice callouts and sound effects
- rendering of the dot-matrix display
- counting down a 30 second timer
- watching for the shot sequence

A top-level coroutine can then start these four coroutines and wait for the
player to complete the shot or wait for the timer to expire.

Only one coroutine is allowed to run at any time. If one coroutine is
executing, all other coroutines are blocked waiting to be resumed. This allows
coroutines to share state without the use of locks.

## Usage

Use `New` to create a new coroutine. It takes a function with a context,
`*coroutine.C`, as its first argument:

```go
func foo(co *coroutine.C) {
    // ...
}

cancel := coroutine.New(foo)
```

The coroutine is then executed in a goroutine until the function exits or
yields. `New` returns a cancellation function that can be called, if needed, to
terminate the coroutine before the function exits.

A coroutine can yield by calling one of the following functions found in the
context:

- `Sleep`: resume after a time duration has elapsed
- `WaitFor`: resume when a specific event has been received
- `WaitForUntil`: resume when a specific event has been received or after a time duration has elapsed

The return values of the yield functions should be checked to see if the
coroutine has been canceled. If so, it should perform any necessary cleanup
tasks and directly exit the function without yielding again.

```go
if done := co.Sleep(5 * time.Second); done {
    return
}
```

In the main application loop, call the `Post` function to queue an event.
The `Tick` function should then be called once for each loop iteration to
resume coroutines as needed. For example:

```go
ticker := time.NewTicker(16670 * time.Microsecond) // 60 fps

for {
    <-ticker.C
    for _, event := range GetEvents() {
        coroutine.Post(event)
    }
    coroutine.Tick()
}
```

## Events

Any type that implements the `Event` interface can be used as an event. It must
implement one function, `Key`, and it can return any type that is comparable.

When a coroutine waits for an event it specifies the key it is interested in.
The coroutine is resumed on the first event that matches the key.

For example, let's say that we have an event struct to represent when a switch
is pressed down or released:

```go
type SwitchEvent struct {
    ID        string
    Released  bool
    Timestamp time.Time
}
```

The `ID` identifies the switch, `Released` is `false` if the switch is being
pressed down and `true` when released, and `Timestamp` records when this event
was received. When waiting for this event, we are interested in matching the
`ID` and the state of `Released` but not the `Timestamp` since that varies. The
`Key` function is then:

```go
func (e SwitchEvent) Key() interface{} {
    return SwitchEvent{ID: s.ID, Released: s.Released}
}
```

To wait for when the `"StartButton"` is pressed:

```go
WaitFor(SwitchEvent{ID: "StartButton"})
```

To wait for when the `"StartButton`" is released:

```go
WaitFor(SwitchEvent{ID: "StartButton", Released: true})
```

The common case here is to wait for switches being pressed and using `Released`
as the boolean takes advantage of it being `false` when it is omitted. If
waiting for switches to be released is the common case, using `Pressed` as the
boolean would be preferred.

If all fields are used for the key, the `Key` function can return itself:

```go
func (e TimeoutEvent) Key() interface{} {
    return e
}
```

## Cancellation

When a coroutine is created with `coroutine.New`, a cancellation function is
also created. This function is called automatically when the coroutine function
exits or it can be called manually to terminate the coroutine early.

A coroutine has the option of creating another coroutine that shares its
cancellation function by using `New` in the coroutine context instead of
`coroutine.New`. In this case, the child coroutine is canceled when the parent
coroutine is canceled.

For example, let's create two coroutines--one will display "A" every second,
one will display "B" every other second:

```go
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
```

Now create another coroutine that starts these two coroutines by sharing the
same cancellation function. The coroutine then waits 10 seconds and then exits:

```go
func display(co *coroutine.C) {
    co.New(displayA)
    co.New(displayB)

    co.Sleep(10 * time.Second)
    println("done")
}
```

Now create a main loop:

```go
func main() {
    ticker := time.NewTicker(16670 * time.Microsecond) // 60 fps

    println("start")
    coroutine.New(display)
    for {
        <-ticker.C
        coroutine.Tick()
    }
}
```

When `display` exits after printing `"done"`, its cancellation function is
automatically called. This then causes `displayA` and `displayB` to be
cancelled. The program then no longer prints any output. Run the example with:

```bash
go run examples/cancellation/cancellation.go
```

## Sequencer

There are many times where a coroutine has a simple structure that is repeated:

- do something
- wait for something

The Judge Dredd pinball machine says "Law master computer online. Welcome
aboard" when starting a game and then a few seconds later says "Use fire button
to launch ball". Code for that would look something like this:

```go
PlaySpeech("law_master_computer_online_welcome_aboard.wav")
if _, done := co.WaitFor(SpeechFinishedEvent{}); done {
    StopSpeech("law_master_computer_online_welcome_aboard.wav")
    return
}
if done := co.Sleep(5 * time.Second); done {
    return
}
PlaySpeech("use_fire_button_to_launch_ball.wav")
if _, done := co.WaitFor(SpeechFinishedEvent{}); done {
    StopSpeech("use_fire_button_to_launch_ball.wav")
    return
}
```

A sequencer can be used instead:

```go
func launchAudio(co *coroutine.C) {
    s := coroutine.NewSequencer(co)

    s.Do(func() { PlaySpeech("law_master_computer_online_welcome_aboard.wav")})
    s.Cancel(func() { PlaySpeech("law_master_computer_online_welcome_aboard.wav")}
    s.WaitFor(SpeechFinishedEvent{})

    s.Sleep(5_000 * time.Millisecond)

    s.Do(func() { PlaySpeech("use_fire_button_to_launch_ball.wav")})
    s.Cancel(func() { PlaySpeech("use_fire_button_to_launch_ball.wav")}
    s.WaitFor(SpeechFinishedEvent{})

    if done := s.Run(); done {
        return
    }
    // ...
}
```

`Run` returns true if any of the yields were cancelled. While this is still a
bit verbose, a specialized sequencer can be made based on this sequencer to
make it more concise. For example, the equivalent code using the sequencer in
the Super Pinball System is:

```go
func launchAudio(co *coroutine.C) {
    s := spin.NewSequencer(co)

    s.Do(spin.PlaySpeech{ID: SpeechLawMasterComputerOnlineWelcomeAboard})
    s.WaitFor(spin.SpeechFinishedEvent{})

    s.Sleep(5_000)

    s.Do(spin.PlaySpeech{ID: UseFireButtonToLaunchBall})
    s.WaitFor(spin.SpeechFinishedEvent{})

    if done := s.Run(); done {
        return
    }
    // ...
}
```

In this specialized version:

- `Do` now takes an action instead of a function.
- `Cancel` is now automatically called if the `Do` action is `PlaySpeech`
- `Sleep` takes an integer for milliseconds instead of a `time.Duration`

And now the initial example in this README can be written as follows:

```go
func watchRamps(co *coroutine.C) {
    s := spin.NewSequencer(co)

    s.WaitFor(spin.ShotEvent{ID: jd.ShotLeftRamp})
    s.WaitFor(spin.ShotEvent{ID: jd.ShotRightRamp})
    s.WaitFor(spin.ShotEvent{ID: jd.ShotLeftRamp})
    s.Do(spin.AwardScore{Val: 1_000_000})
    s.Run()
}
```

## Watchdog

Since only one coroutine can run at a time it should complete its work in a
timely manner when resumed and yield as soon as possible. A watchdog is
provided to help debug issues that can cause deadlock or unexpected coroutine
processing delays. To use, create the watchdog before starting any coroutines
and reset it on each iteration in the main loop:


```go
ticker := time.NewTicker(16670 * time.Microsecond) // 60 fps

watchdog := coroutine.NewWatchdog(1 * time.Second)
coroutine.New(display)
for {
    <-ticker.C
    watchdog.Reset()
    coroutine.Tick()
}
```

In this case, if the watchdog hasn't been reset within one second, something
has gone wrong. The watchdog will then print out the stack trace for all
running goroutines and then panic.

An example is provided where `time.Sleep` is accidentally used instead of
`co.Sleep` which causes the watchdog to panic:

```
go run examples/watchdog/watchdog.go
```

## Documentation

There is no API documentation at the moment but it can be written upon request.

## Demo

A very simple pinball "simulator" is provided as a demonstration:

```
go run examples/pinball/pinball.go
```

In this simulation, a random switch event is generated every so often. Every
three seconds, there is a check to see if the ball should drain. It starts off
at a 1% chance and then increases by 1% each time. Scoring is as follows:

- sling: 10 points
- standup target
  - unlit: 100 points + bonus 10 points
  - lit: 25 points + bonus 10 points
- outlanes: 125 points

When all three standup targets are hit, the kickback in the left outlane is lit
and the standup targets reset. When the ball drains, the end of ball sequence
runs to show the final score.

## Questions?

Send an email to mike dot mcgann at droptargetpinball dot com.

## License

MIT
