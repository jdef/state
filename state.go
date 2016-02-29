/*
Copyright 2016 James DeFelice

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package state

type (
	// Context provides runtime context to state machines.
	Context interface {
		// Done returns a chan that closes to signal that the state machine
		// should terminate.
		Done() <-chan struct{}
	}

	// SimpleContext is created via `make(SimpleContext)`
	SimpleContext chan struct{}

	// Event is a marker interface for event objects. Event's aren't required to
	// exhibit any state or behavior by default.
	Event interface {
		// Event is an interface marker func. It is never invoked.
		Event() struct{}
	}

	// Event implementations may embed AbstractEvent to reduce boilerplate code.
	AbstractEvent struct{}

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

	// SimpleEvents provides a basic implementation of the Events interface
	SimpleEvents struct {
		events chan Event
	}

	Machine interface {
		Events
		// InitialState returns the initial state func of a state machine
		InitialState() Fn
	}

	// SimpleMachine provides a basic implementation of the Machine interface
	SimpleMachine struct {
		Events
		initialState Fn
	}

	// Fn is a state func that implements behavior for a particular state. Upon
	// state transition the next state is returned. Nil is returned to indicate
	// that there is no next state and that the state machine should die.
	Fn func(Context, Machine) Fn

	// SuperMachine is implemented by state machines that support being extended
	// by sub-state machines. A sub-state machine triggers a state transition
	// by sending desired next state to the chan returned by Hijack.
	SuperMachine interface {
		Hijack() chan<- Fn
		// SubMachine returns a simple sub-state Machine with the given queue length
		// and (optional) initial state. If nil is given for the initial state of the
		// sub-state Machine then the actual initial state is derived from the
		// InitialState() func of the super-state Machine.
		SubMachine(int, Fn) SubMachine
	}

	SubMachine interface {
		Super() SuperMachine
		Forward(Context, Event)
	}
)

var (
	// AbstractEvent implements Event
	_ Event = &AbstractEvent{}
	// SimpleContext implements Context
	_ Context = make(SimpleContext)
	// SimpleEvents implements Events
	_ Events = &SimpleEvents{}
	// SimpleMachine implements Machine
	_ Machine = &SimpleMachine{}
)

func (_ *AbstractEvent) Event() struct{} { return struct{}{} }

func (c SimpleContext) Done() <-chan struct{} { return c }

func NewSimpleEvents(queueLength int) Events  { return &SimpleEvents{make(chan Event, queueLength)} }
func (se *SimpleEvents) Source() <-chan Event { return se.events }
func (se *SimpleEvents) Sink() chan<- Event   { return se.events }

func NewSimpleMachine(queueLength int, initialState Fn) Machine {
	return &SimpleMachine{
		Events:       NewSimpleEvents(queueLength),
		initialState: initialState,
	}
}

func (m *SimpleMachine) InitialState() Fn { return m.initialState }

// Cancel closes the context; may be invoked multiple times without error but is
// not safe to execute concurrently.
func (c SimpleContext) Cancel() {
	select {
	case <-c:
		// already closed
	default:
		close(c)
	}
}

// Run runs a state Machine, beginning with the InitialState() and transitioning
// through states as returned by state funcs until reaching a nil state Fn.
func Run(ctx Context, m Machine) {
	state := m.InitialState()
	for state != nil {
		state = state(ctx, m)
	}
}
