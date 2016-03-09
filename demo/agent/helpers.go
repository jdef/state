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

/*
 * super-state machine helper code follows
 */

type (
	// TODO(jdef) I don't like this name, need to revisit
	SuperMachineInterface interface {
		Interface
		state.Transition
		state.SuperMachine

		// SubMachineInterface is a convenience func to create package-
		// specific sub-state machines.
		SubMachineInterface(int, state.Fn) SubMachineInterface
	}

	superMachineInterfaceImpl struct {
		Interface
		hijackChan chan state.Fn
	}
)

func AsSuperMachine(i Interface) SuperMachineInterface {
	return &superMachineInterfaceImpl{Interface: i, hijackChan: make(chan state.Fn)}
}

func (a *superMachineInterfaceImpl) NextState() <-chan state.Fn                    { return a.hijackChan }
func (a *superMachineInterfaceImpl) Hijack() chan<- state.Fn                       { return a.hijackChan }
func (a *superMachineInterfaceImpl) SubMachine(l int, f state.Fn) state.SubMachine { return newSubMachine(a, l, f) }

func (a *superMachineInterfaceImpl) SubMachineInterface(queueLen int, initialFn state.Fn) SubMachineInterface {
	return a.SubMachine(queueLen, initialFn).(SubMachineInterface)
}

/*
 * sub-state machine helper code follows
 */

type (
	// TODO(jdef) I don't like this name, need to revisit
	SubMachineInterface interface {
		Interface
		state.Transition
		state.SubMachine
	}

	// subMachineInterfaceImpl is a helper for quickly building sub-state machines that
	// extend Agent functionality. Sub-state machines typically need their
	// own event queue and may want to override the initial state func.
	subMachineInterfaceImpl struct {
		SuperMachineInterface
		events       chan state.Event
		initialState state.Fn
	}
)

// subMachineInterfaceImpl implements state.SubMachineInterface
var _ SubMachineInterface = &subMachineInterfaceImpl{}

func newSubMachine(super SuperMachineInterface, queueLength int, initialState state.Fn) state.SubMachine {
	return &subMachineInterfaceImpl{
		SuperMachineInterface: super,
		events:                make(chan state.Event, queueLength),
		initialState:          initialState,
	}
}

func (m *subMachineInterfaceImpl) InitialState() state.Fn {
	if m.initialState != nil {
		return m.initialState
	}
	return m.SuperMachineInterface.InitialState()
}

// Dispatch sends an event to the super-state machine. The super-state machine should
// probably have a buffered event queue if there's a party external to the state
// machine substrate that's also feeding events into the machine, otherwise this
// may block indefinitely.
func (m *subMachineInterfaceImpl) Dispatch(ctx state.Context, e state.Event) {
	select {
	case <-ctx.Done():
		return
	case m.Super().(SuperMachineInterface).Sink() <- e:
	}
}

func (m *subMachineInterfaceImpl) Source() <-chan state.Event                { return m.events }
func (m *subMachineInterfaceImpl) Sink() chan<- state.Event                  { return m.events }
func (m *subMachineInterfaceImpl) Super() state.SuperMachine                 { return m.SuperMachineInterface }
func (m *subMachineInterfaceImpl) Hijack() chan<- state.Fn                   { return nil } // is not hijackable
func (m *subMachineInterfaceImpl) NextState() <-chan state.Fn                { return nil } // is not hijackable
func (m *subMachineInterfaceImpl) SubMachine(int, state.Fn) state.SubMachine { return nil } // is not hijackable

// Masquerade returns a reference to an imposter of the super-machine that may
// be passed to the super-machine's state handlers for upstream event delegation.
// The Source of the returned instance is expected to reference the source event
// stream of the actual super-machine. All other interface funcs may be overridden
// by the sub-machine implementation.
func Masquerade(m SubMachineInterface) state.Machine { return &masqInterface{m} }

type masqInterface struct {
	SubMachineInterface
}

// Source returns the upstream source so that we may pass this masqInterface instance
// to a super-state handler and it will read events from its own source, instead
// of sub-machine's.
func (m *masqInterface) Source() <-chan state.Event { return m.Super().(SuperMachineInterface).Source() }

// NextState returns the upstream NextState so that we may pass this masqInterface instance
// to a super-state handler and it will read states from the super-machine helper
// associated with this sub-state machine.
func (m *masqInterface) NextState() <-chan state.Fn { return m.Super().(SuperMachineInterface).NextState() }

// Super is a convenience func that returns the super-state machine as Interface
func SuperOf(sub SubMachineInterface) Interface {
	return sub.Super().(Interface)
}

// Sub is a convenience func that returns the given Machine as a SubMachineInterface
func AsSub(m state.Machine) SubMachineInterface {
	return m.(SubMachineInterface)
}
