package agent

import (
	"github.com/jdef/state"
)

type AgentInterface interface {
	state.Machine

	Disconnected() state.Fn
	Connected() state.Fn
	Terminating() state.Fn

	get() *Agent
}

type Agent struct {
	events chan state.Event
	pulse  chan<- struct{}
}

// Agent implements AgentInterface
var _ AgentInterface = &Agent{}

func New(pulse chan<- struct{}) AgentInterface {
	return &Agent{
		events: make(chan state.Event),
		pulse:  pulse,
	}
}

func (a *Agent) Source() <-chan state.Event { return a.events }
func (a *Agent) Sink() chan<- state.Event   { return a.events }
func (a *Agent) Disconnected() state.Fn     { return disconnected }
func (a *Agent) Connected() state.Fn        { return connected }
func (a *Agent) Terminating() state.Fn      { return terminating }
func (a *Agent) State() interface{}         { return a }
func (a *Agent) InitialState() state.Fn     { return disconnected }
func (a *Agent) get() *Agent                { return a }

func checkForHijack(e state.Event, otherwise state.Fn) state.Fn {
	if h, ok := e.(state.HijackEvent); ok {
		if f := h.Hijacked(); f != nil {
			return f
		}
	}
	return otherwise
}

func disconnected(ctx state.Context, m state.Machine) state.Fn {
	defer println("<leaving disconnected>")
	for {
		select {
		case event := <-m.Source():
			agent := m.State().(AgentInterface)
			switch event := event.(type) {
			case *ConnectRequest:
				agent.get().doConnect(event)
				return checkForHijack(event, agent.Connected())
			case *Heartbeat:
				agent.get().doHeartbeat(ctx, event)
			}
		case <-ctx.Done():
			agent := m.State().(AgentInterface)
			return agent.Terminating()
		}
	}
}

func connected(ctx state.Context, m state.Machine) state.Fn {
	defer println("<leaving connected>")
	for {
		select {
		case event := <-m.Source():
			agent := m.State().(AgentInterface)
			switch event := event.(type) {
			case *DisconnectRequest:
				agent.get().doDisconnect(event)
				return checkForHijack(event, agent.Disconnected())
			case *Heartbeat:
				agent.get().doHeartbeat(ctx, event)
			}
		case <-ctx.Done():
			agent := m.State().(AgentInterface)
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

//
// events
//

type (
	hijackSupport     struct{ state.Fn }
	DisconnectRequest struct{ hijackSupport }
	ConnectRequest    struct{ hijackSupport }
	Heartbeat         struct{}
)

func (h *hijackSupport) Hijacked() state.Fn { return h.Fn }

func (_ *DisconnectRequest) IsEvent() bool { return true }
func (_ *ConnectRequest) IsEvent() bool    { return true }
func (_ *Heartbeat) IsEvent() bool         { return true }
