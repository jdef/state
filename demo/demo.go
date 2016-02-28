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

package demo

import (
	"time"

	"github.com/jdef/state"
	"github.com/jdef/state/demo/agent"
)

type context chan struct{}

func (ctx context) Done() <-chan struct{} {
	return ctx
}

func logPulse(ctx state.Context, pulse <-chan struct{}) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-pulse:
			println("pulse")
		}
	}
}

func sendHeartbeat(ctx state.Context, sink state.EventSink) {
	ticker := time.NewTicker(1 * time.Second)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			select {
			case sink.Sink() <- &agent.Heartbeat{}:
			case <-ctx.Done():
				return
			}
		}
	}
}

func RunWith(a agent.Interface, pulse chan struct{}) {
	ctx := make(context)

	ch := make(chan struct{})
	go func() {
		defer close(ch)
		state.Run(ctx, a)
	}()

	// ping pong
	go logPulse(ctx, pulse)
	go sendHeartbeat(ctx, a)

	time.Sleep(2 * time.Second)

	a.Sink() <- &agent.ConnectRequest{}

	time.Sleep(2 * time.Second)

	a.Sink() <- &agent.DisconnectRequest{}

	time.Sleep(5 * time.Second)

	close(ctx) // tell the state machine to terminate
	<-ch
}
