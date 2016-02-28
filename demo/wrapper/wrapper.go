package wrapper

import (
	"github.com/jdef/state"
	"github.com/jdef/state/demo/agent"
)

//
// demonstration of agent state machine composition
//

type Interface interface {
	agent.AgentInterface
	internal() *Wrapper
}

type Wrapper struct {
	agent.AgentInterface
	eventChan chan state.Event
}

func New(a agent.AgentInterface) Interface {
	return &Wrapper{
		AgentInterface: a,
		eventChan:      make(chan state.Event),
	}
}

// Wrapper implements agent.AgentInterface
var _ agent.AgentInterface = &Wrapper{}

// Connected overrides the default implementation
func (ha *Wrapper) Connected() state.Fn        { return connectedStage1 }
func (ha *Wrapper) Disconnected() state.Fn     { return happilyDisconnected }
func (ha *Wrapper) Terminating() state.Fn      { return happilyTerminating }
func (ha *Wrapper) InitialState() state.Fn     { return happilyDisconnected }
func (ha *Wrapper) Sink() chan<- state.Event   { return ha.eventChan }
func (ha *Wrapper) Source() <-chan state.Event { return ha.eventChan }
func (ha *Wrapper) internal() *Wrapper         { return ha }

type SuperMachine struct {
	Interface
}

// Source returns the upstream source so that we may pass this upstream instance
// to a upstream state handler and it will read events from its own source, not wrapper's.
func (d *SuperMachine) Source() <-chan state.Event {
	return d.Super().Source()
}

func (d *SuperMachine) Super() agent.AgentInterface {
	return d.Interface.internal().AgentInterface
}

func (d *SuperMachine) Send(ctx state.Context, e state.Event) {
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

func happilyTerminating(ctx state.Context, m state.Machine) state.Fn {
	println("happily terminating")
	defer println("<leaving happily terminating>")

	var (
		upstream = &SuperMachine{m.(Interface)}
	)

	// we'd normally clean up any resources here.
	// there's no good reason for overriding the terminating state in this
	// case, we just do it for demo purposes.

	return upstream.Super().(agent.AgentInterface).Terminating()(ctx, upstream)
}

func happilyDisconnected(ctx state.Context, m state.Machine) state.Fn {
	println("happily disconnected")
	defer println("<leaving happily disconnected>")

	var (
		upstream = &SuperMachine{m.(Interface)}
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
		upstream = &SuperMachine{wrapper}
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
		upstream = &SuperMachine{m.(Interface)}
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
