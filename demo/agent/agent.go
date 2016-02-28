package agent

import (
	"github.com/jdef/state"
)

type (
	Interface interface {
		state.Machine
		state.Hijackable // support sub-state machines

		Disconnected() state.Fn
		Connected() state.Fn
		Terminating() state.Fn

		get() *Agent
	}

	Agent struct {
		events chan state.Event
		pulse  chan<- struct{}
		hijack chan state.Fn
	}

	//
	// events
	//

	DisconnectRequest struct{ state.AbstractEvent }
	ConnectRequest    struct{ state.AbstractEvent }
	Heartbeat         struct{ state.AbstractEvent }
)

// Agent implements Interface
var _ Interface = &Agent{}

func New(pulse chan<- struct{}, backlog int) Interface {
	return &Agent{
		events: make(chan state.Event, backlog),
		pulse:  pulse,
		hijack: make(chan state.Fn),
	}
}

func (a *Agent) Hijack() chan<- state.Fn    { return a.hijack }
func (a *Agent) Source() <-chan state.Event { return a.events }
func (a *Agent) Sink() chan<- state.Event   { return a.events }
func (a *Agent) Disconnected() state.Fn     { return disconnected }
func (a *Agent) Connected() state.Fn        { return connected }
func (a *Agent) Terminating() state.Fn      { return terminating }
func (a *Agent) InitialState() state.Fn     { return disconnected }
func (a *Agent) get() *Agent                { return a }

func disconnected(ctx state.Context, m state.Machine) state.Fn {
	defer println("<leaving disconnected>")
	agent := m.(Interface)
	for {
		select {
		case event := <-m.Source():
			switch event := event.(type) {
			case *ConnectRequest:
				agent.get().doConnect(event)
				return agent.Connected()
			case *Heartbeat:
				agent.get().doHeartbeat(ctx, event)
			}
		case fn := <-agent.get().hijack:
			return fn
		case <-ctx.Done():
			return agent.Terminating()
		}
	}
}

func connected(ctx state.Context, m state.Machine) state.Fn {
	defer println("<leaving connected>")
	agent := m.(Interface)
	for {
		select {
		case event := <-m.Source():
			switch event := event.(type) {
			case *DisconnectRequest:
				agent.get().doDisconnect(event)
				return agent.Disconnected()
			case *Heartbeat:
				agent.get().doHeartbeat(ctx, event)
			}
		case fn := <-agent.get().hijack:
			return fn
		case <-ctx.Done():
			return agent.Terminating()
		}
	}
}

func terminating(ctx state.Context, m state.Machine) state.Fn {
	defer println("<leaving terminating>")
	return nil
}

func (a *Agent) doConnect(_ *ConnectRequest) {
	println("connected()")
}

func (a *Agent) doDisconnect(_ *DisconnectRequest) {
	println("disconnected()")
}

func (a *Agent) doHeartbeat(ctx state.Context, _ *Heartbeat) {
	println("heartbeat()")
	select {
	case a.pulse <- struct{}{}:
	case <-ctx.Done():
	}
}
