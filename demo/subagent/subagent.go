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
}

// Subagent is a sub-state machine implementation that extends the machine implemented
// in the agent package. In particular the Connected state is broken into two sub-states,
// connectedStage1 and connectedStage2. This is done purely to illustrate how to use the
// SuperMachine interface to instigate sub-state transitions.
type Subagent struct {
	agent.SubMachineInterface
}

func New(a agent.SuperMachineInterface) Interface {
	return &Subagent{a.SubMachineInterface(0, happilyDisconnected)}
}

// Subagent implements agent.Interface
var _ agent.Interface = &Subagent{}

func (ha *Subagent) Disconnected() state.Fn { return happilyDisconnected }
func (ha *Subagent) Connected() state.Fn    { return connectedStage1 }
func (ha *Subagent) Terminating() state.Fn  { return happilyTerminating }

//
// states of the sub-state machine
//

func happilyTerminating(ctx state.Context, m state.Machine) state.Fn {
	println("happily terminating")
	defer println("<leaving happily terminating>")

	subagent := agent.Sub(m)

	// we'd normally clean up any resources here.
	// there's no good reason for overriding the terminating state in this
	// case, we just do it for demo purposes.

	return agent.Super(subagent).Terminating()(ctx, agent.Masquerade(subagent))
}

func happilyDisconnected(ctx state.Context, m state.Machine) state.Fn {
	println("happily disconnected")
	defer println("<leaving happily disconnected>")

	var (
		subagent = agent.Sub(m)
		t        = state.Upon(agent.Super(subagent).Disconnected(), ctx, agent.Masquerade(subagent))
	)

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
			subagent.Dispatch(ctx, event)

		case f := <-t.NextState():
			return f
		}
	}
}

func connectedStage1(ctx state.Context, m state.Machine) state.Fn {
	println("happily connected1")
	defer println("<leaving happily connected1>")

	var (
		subagent = agent.Sub(m)
		t        = state.Upon(agent.Super(subagent).Connected(), ctx, agent.Masquerade(subagent))
	)

	for {
		select {
		case event := <-m.Source():

			switch event.(type) {
			case *agent.DisconnectRequest:
				println(".. happily disconnecting")

			case *agent.Heartbeat:
				println(".. happily entering connectedStage2")
				f, ok := state.TryHijack(subagent.Super(), ctx, connectedStage2, t)
				if ok {
					return f
				}

			default:
			}

			// forward the event upstream
			subagent.Dispatch(ctx, event)

		case f := <-t.NextState():
			return f
		}
	}
}

func connectedStage2(ctx state.Context, m state.Machine) state.Fn {
	println("happily connected2")
	defer println("<leaving happily connected2>")

	var (
		subagent = agent.Sub(m)
		t        = state.Upon(agent.Super(subagent).Connected(), ctx, agent.Masquerade(subagent))
	)

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
			subagent.Dispatch(ctx, event)

		case f := <-t.NextState():
			return f
		}
	}
}
