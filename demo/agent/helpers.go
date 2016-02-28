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

package agent

import (
	"github.com/jdef/state"
)

type (
	// subMachine is a helper for quickly building sub-state machines that
	// extend Agent functionality. Sub-state machines typically need their
	// own event queue and may want to override the initial state func.
	subMachine struct {
		Interface
		events       chan state.Event
		initialState state.Fn
	}

	// TODO(jdef) I don't like this name, need to revisit
	SubMachineInterface interface {
		Interface
		state.SubMachine
	}
)

// subMachine implements state.SubMachineInterface
var _ SubMachineInterface = &subMachine{}

func newSubMachine(super Interface, queueLength int, initialState state.Fn) state.SubMachine {
	return &subMachine{
		Interface:    super,
		events:       make(chan state.Event, queueLength),
		initialState: initialState,
	}
}

func (m *subMachine) InitialState() state.Fn {
	if m.initialState != nil {
		return m.initialState
	}
	return m.Interface.InitialState()
}

// Forward sends an event to the super-state machine. The super-state machine should
// probably have a buffered event queue if there's a party external to the state
// machine substrate that's also feeding events into the machine, otherwise this
// may block indefinitely.
func (m *subMachine) Forward(ctx state.Context, e state.Event) {
	select {
	case <-ctx.Done():
		return
	case m.Super().Sink() <- e:
	}
}

func (m *subMachine) Source() <-chan state.Event { return m.events }
func (m *subMachine) Sink() chan<- state.Event   { return m.events }
func (m *subMachine) Super() state.SuperMachine  { return m.Interface }

// Masquerade returns a reference to an imposter of the super-machine that may
// be passed to the super-machine's state handlers for upstream event delegation.
// The Source of the returned instance is expected to reference the source event
// stream of the actual super-machine. All other interface funcs may be overridden
// by the sub-machine implementation.
func Masquerade(m SubMachineInterface) state.Machine { return &masq{m} }

type masq struct {
	SubMachineInterface
}

// Source returns the upstream source so that we may pass this masq instance
// to a super-state handler and it will read events from its own source, instead
// of sub-machine's.
func (m *masq) Source() <-chan state.Event {
	return m.Super().Source()
}
