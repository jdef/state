package subagent

import (
	"github.com/jdef/state"
	"github.com/jdef/state/demo/agent"
)

//
// demonstration of agent state machine composition
//

type Interface interface {
	agent.Interface
	internal() *Subagent
}

// Subagent is a sub-state machine implementation that extends the machine implemented
// in the agent package. In particular the Connected state is broken into two sub-states,
// connectedStage1 and connectedStage2. This is done purely to illustrate how to use the
// Hijackable interface to instigate sub-state transitions.
type Subagent struct {
	agent.Interface
	eventChan chan state.Event
}

func New(a agent.Interface) Interface {
	return &Subagent{
		Interface: a,
		eventChan: make(chan state.Event),
	}
}

// Subagent implements agent.Interface
var _ agent.Interface = &Subagent{}

func (ha *Subagent) Hijack() chan<- state.Fn    { return nil } // subagent is not hijackable
func (ha *Subagent) InitialState() state.Fn     { return happilyDisconnected }
func (ha *Subagent) Disconnected() state.Fn     { return happilyDisconnected }
func (ha *Subagent) Connected() state.Fn        { return connectedStage1 }
func (ha *Subagent) Terminating() state.Fn      { return happilyTerminating }
func (ha *Subagent) Sink() chan<- state.Event   { return ha.eventChan }
func (ha *Subagent) Source() <-chan state.Event { return ha.eventChan }
func (ha *Subagent) internal() *Subagent        { return ha }

//
// some glue that simplifies interaction with the super-state machine
//

type upstream struct {
	agent.Interface
}

func superMachine(m state.Machine) *upstream {
	return &upstream{m.(agent.Interface)}
}

// Source returns the upstream source so that we may pass this upstream instance
// to a upstream state handler and it will read events from its own source, not subagent's.
func (d *upstream) Source() <-chan state.Event {
	return d.super().Source()
}

// super returns the super-state machine
func (d *upstream) super() agent.Interface {
	return d.Interface.(Interface).internal().Interface
}

// Send forwards an event to the super-state machine. The super-state machine should
// probably have a buffered event queue if there's a party external to the state
// machine substrate that's also feeding events into the machine, otherwise this
// may block indefinitely.
func (d *upstream) send(ctx state.Context, e state.Event) {
	select {
	case <-ctx.Done():
		return
	case d.super().Sink() <- e:
	}
}

//
// states of the sub-state machine
//

func happilyTerminating(ctx state.Context, m state.Machine) state.Fn {
	println("happily terminating")
	defer println("<leaving happily terminating>")

	upstream := superMachine(m)

	// we'd normally clean up any resources here.
	// there's no good reason for overriding the terminating state in this
	// case, we just do it for demo purposes.

	return upstream.super().Terminating()(ctx, upstream)
}

func happilyDisconnected(ctx state.Context, m state.Machine) state.Fn {
	println("happily disconnected")
	defer println("<leaving happily disconnected>")

	var (
		upstream = superMachine(m)
		fn       = make(chan state.Fn)
	)

	// we're happy to let upstream's Disconnected state handler
	// drive the state transition when it's ready
	go func() {
		fn <- upstream.super().Disconnected()(ctx, upstream)
	}()

	for {
		select {
		case event := <-m.Source():

			switch event.(type) {
			case *agent.ConnectRequest:
				println(".. happily connecting")

			default:
				// noop
			}

			// forward the event upstream
			upstream.send(ctx, event)

		case f := <-fn:
			return f
		}
	}
}

func connectedStage1(ctx state.Context, m state.Machine) state.Fn {
	println("happily connected1")
	defer println("<leaving happily connected1>")

	var (
		upstream = superMachine(m)
		fn       = make(chan state.Fn)
	)

	// we're happy to let upstream's Connected state handler
	// drive the state transition when it's ready
	go func() {
		fn <- upstream.super().Connected()(ctx, upstream)
	}()

	for {
		select {
		case event := <-m.Source():

			switch event.(type) {
			case *agent.DisconnectRequest:
				println(".. happily disconnecting")

			case *agent.Heartbeat:
				println(".. happily entering connectedStage2")

				// we'll still forward the heartbeat but it may be
				// processed after we transition to stage2. for this
				// demo it doesn't matter if that happens.
				select {
				case upstream.super().Hijack() <- connectedStage2:
				case <-ctx.Done():
				}
			default:
			}

			// forward the event upstream
			upstream.send(ctx, event)

		case f := <-fn:
			return f
		}
	}
}

func connectedStage2(ctx state.Context, m state.Machine) state.Fn {
	println("happily connected2")
	defer println("<leaving happily connected2>")

	var (
		upstream = superMachine(m)
		fn       = make(chan state.Fn)
	)

	// we're happy to let upstream's Connected state handler
	// drive the state transition when it's ready
	go func() {
		fn <- upstream.super().Connected()(ctx, upstream)
	}()

	for {
		select {
		case event := <-m.Source():

			switch event.(type) {
			case *agent.DisconnectRequest:
				println(".. happily disconnecting")

			default:
				// noop
			}

			// forward the event upstream
			upstream.send(ctx, event)

		case f := <-fn:
			return f
		}
	}
}
