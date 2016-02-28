package main

import (
	"time"

	"github.com/jdef/state"
	"github.com/jdef/state/demo/agent"
	"github.com/jdef/state/demo/subagent"
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

func main() {
	ctx := make(context)
	pulse := make(chan struct{})

	var a agent.Interface

	a = subagent.New(agent.New(pulse, 10))

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
