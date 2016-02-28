package wrapper

import (
	"github.com/jdef/state"
	"github.com/jdef/state/demo/agent"
)

//
// demonstration of agent state machine composition
//

type Interface interface {
	agent.Interface
	internal() *Wrapper
}

type Wrapper struct {
	agent.Interface
	eventChan chan state.Event
}

func New(a agent.Interface) Interface {
	return &Wrapper{
		Interface: a,
		eventChan: make(chan state.Event),
	}
}

// Wrapper implements agent.Interface
var _ agent.Interface = &Wrapper{}

// Connected overrides the default implementation
func (ha *Wrapper) InitialState() state.Fn     { return happilyDisconnected }
func (ha *Wrapper) Disconnected() state.Fn     { return happilyDisconnected }
func (ha *Wrapper) Connected() state.Fn        { return connectedStage1 }
func (ha *Wrapper) Terminating() state.Fn      { return happilyTerminating }
func (ha *Wrapper) Sink() chan<- state.Event   { return ha.eventChan }
func (ha *Wrapper) Source() <-chan state.Event { return ha.eventChan }
func (ha *Wrapper) internal() *Wrapper         { return ha }

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
// to a upstream state handler and it will read events from its own source, not wrapper's.
func (d *upstream) Source() <-chan state.Event {
	return d.Super().Source()
}

func (d *upstream) Super() agent.Interface {
	return d.Interface.(Interface).internal().Interface
}

func (d *upstream) Send(ctx state.Context, e state.Event) {
	// TODO(jdef) this is ugly, we probably need/want something better if
	// we're at all concerned about preserving event order
	go func() {
		select {
		case <-ctx.Done():
			return
		case d.Super().Sink() <- e:
		}
	}()
}

//
// states of the sub-state machine
//

func happilyTerminating(ctx state.Context, m state.Machine) state.Fn {
	println("happily terminating")
	defer println("<leaving happily terminating>")

	var (
		upstream = superMachine(m)
	)

	// we'd normally clean up any resources here.
	// there's no good reason for overriding the terminating state in this
	// case, we just do it for demo purposes.

	return upstream.Super().(agent.Interface).Terminating()(ctx, upstream)
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
		fn <- upstream.Super().Disconnected()(ctx, upstream)
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
			upstream.Send(ctx, event)

		case f := <-fn:
			return f
		}
	}
}

func connectedStage1(ctx state.Context, m state.Machine) state.Fn {
	println("happily connected1")
	defer println("<leaving happily connected1>")

	var (
		wrapper  = m.(Interface)
		upstream = superMachine(wrapper)
		fn       = make(chan state.Fn)
	)

	// we're happy to let upstream's Connected state handler
	// drive the state transition when it's ready
	go func() {
		fn <- upstream.Super().Connected()(ctx, upstream)
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
				wrapper.Hijack() <- connectedStage2
			default:
			}

			// forward the event upstream
			upstream.Send(ctx, event)

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
		fn <- upstream.Super().Connected()(ctx, upstream)
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
			upstream.Send(ctx, event)

		case f := <-fn:
			return f
		}
	}
}
