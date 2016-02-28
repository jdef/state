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

package state_test

import (
	"fmt"
	"time"

	"github.com/jdef/state"
)

type (
	pingEvent struct{ state.AbstractEvent }
	pongEvent struct{ state.AbstractEvent }
)

var (
	ping = &pingEvent{}
	pong = &pongEvent{}
)

func newPingpong() state.Machine {
	return state.NewSimpleMachine(1, awaitEvent)
}

func awaitEvent(ctx state.Context, m state.Machine) state.Fn {
	for {
		// check for a tie
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		event := <-m.Source()

		switch event.(type) {
		case *pingEvent:
			fmt.Println("ping")
			time.Sleep(1 * time.Second)
			select {
			case m.Sink() <- pong:
			case <-ctx.Done():
				return nil
			}
		case *pongEvent:
			fmt.Println("pong")
			time.Sleep(1 * time.Second)
			select {
			case m.Sink() <- ping:
			case <-ctx.Done():
				return nil
			}
		}
	}
}

func Example() {
	var (
		pp  = newPingpong()
		ctx = make(state.SimpleContext)
	)

	// kickstart the game
	pp.Sink() <- ping

	// run long enough for some events to pass
	time.AfterFunc(4500*time.Millisecond, ctx.Cancel)
	state.Run(ctx, pp)

	// Output:
	// ping
	// pong
	// ping
	// pong
	// ping
}
