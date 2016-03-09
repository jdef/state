## Status

This is experimental software.
It almost certainly has sharp edges.
Handle with care.

* go:generate templates require Golang 1.6+

## State

This package was inspired by the lexical scanner in Golang, presented [here](https://www.youtube.com/watch?v=HxaD_trXwRE).

The original "pattern" itself is quite simple and there's a single-state demo in the `example_test.go` file of this package.
My goal with this project, however, is to explore the nature of composable state machines in Go.
If you're not interested in composable state machines then this package is probably overkill for your use case.
(But you're welcome to stay!)

### Demo Agent

The demo agent is a state machine that transitions between three states: `Connected`, `Disconnected`, and `Terminating`.
It also generates heartbeat events to a pulse chan.
The example file in the package will run the agent (via `go test -v ./demo/agent`).

The state machine implementation of this demo agent is special: it may be extended.
Such is illustrated in the `demo/subagent` package, where the `Connected` state is broken into two stages.
I've attempted to keep most the ugly interface casting confined to the `helper` module of `package agent`.
I've also tried to prevent private field/func access bleeding from `agent.go` into `helper.go`.

### TODOs

- [x] build a Golang code generator that generates `helper.go`-like files for packages containing state machines.
