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

//go:generate gosm -o helpers.go

package agent

import (
	"github.com/jdef/state"
)

type (
	Interface interface {
		state.Machine

		Disconnected() state.Fn
		Connected() state.Fn
		Terminating() state.Fn

		get() *Agent
	}

	Agent struct {
		state.Machine
		pulse chan<- struct{}
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
		Machine: state.NewSimpleMachine(backlog, disconnected),
		pulse:   pulse,
	}
}

func (a *Agent) Disconnected() state.Fn { return disconnected }
func (a *Agent) Connected() state.Fn    { return connected }
func (a *Agent) Terminating() state.Fn  { return terminating }

func (a *Agent) get() *Agent { return a }

func disconnected(ctx state.Context, m state.Machine) state.Fn {
	println("disconnected")
	defer println("<leaving disconnected>")
	agent := m.(Interface)
	next := state.Next(m) // support hijackers
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
		case fn := <-next:
			return fn
		case <-ctx.Done():
			return agent.Terminating()
		}
	}
}

func connected(ctx state.Context, m state.Machine) state.Fn {
	println("connected")
	defer println("<leaving connected>")
	agent := m.(Interface)
	next := state.Next(m) // support hijackers
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
		case fn := <-next:
			return fn
		case <-ctx.Done():
			return agent.Terminating()
		}
	}
}

func terminating(ctx state.Context, m state.Machine) state.Fn {
	println("terminating")
	defer println("<leaving terminating>")
	return nil
}

func (a *Agent) doConnect(_ *ConnectRequest) {
	println("do-connect")
}

func (a *Agent) doDisconnect(_ *DisconnectRequest) {
	println("do-disconnected")
}

func (a *Agent) doHeartbeat(ctx state.Context, _ *Heartbeat) {
	println("do-heartbeat")
	select {
	case a.pulse <- struct{}{}:
	case <-ctx.Done():
	}
}
