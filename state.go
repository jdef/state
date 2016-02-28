package state

type (
	Context interface {
		Done() <-chan struct{}
	}

	Event interface {
		IsEvent() bool
	}

	// HijackEvent signals to a downstream state that a state transition is desired
	// and that the downstream state should return the Fn given in this event.
	HijackEvent interface {
		Hijacked() Fn
	}

	EventSource interface {
		Source() <-chan Event
	}

	EventSink interface {
		Sink() chan<- Event
	}

	Events interface {
		EventSource
		EventSink
	}

	State interface {
		State() interface{}
	}

	Machine interface {
		Events
		State
		InitialState() Fn
	}

	Fn func(_ Context, _ Machine) Fn
)

func Run(ctx Context, m Machine) {
	state := m.InitialState()
	for state != nil {
		state = state(ctx, m)
	}
}
