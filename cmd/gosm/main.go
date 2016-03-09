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

package main

import (
	"flag"
	"io"
	"os"
	"text/template"
)

func env(name, defaultValue string) (value string) {
	if value = os.Getenv(name); value == "" {
		value = defaultValue
	}
	return
}

func dieUpon(err error) {
	if err != nil {
		panic(err.Error())
	}
}

func main() {
	var (
		statepkg = flag.String("statepkg", "github.com/jdef/state", "fully-qualified name of the state package")
		pkg      = flag.String("pkg", env("GOPACKAGE", "foo"), "name of the target package")
		of       = flag.String("o", "", "name of the file to write output to, default to STDOUT")
		iface    = flag.String("iface", "Interface", "name of the public state machine interface type")
	)
	flag.Parse()

	// TODO(jdef) should probably parse the file that declared the generator and ensure that
	// the "iface" type is a state.Machine

	ctx := struct {
		StatePackage string
		Package      string
		Interface    string
	}{*statepkg, *pkg, *iface}

	var output io.Writer
	if *of != "" {
		f, err := os.Create(*of)
		dieUpon(err)
		defer f.Close()
		output = f
	} else {
		output = os.Stdout
	}

	t, err := template.New("helpers").Parse(helpersTemplate)
	dieUpon(err)

	err = t.Execute(output, ctx)
	dieUpon(err)
}

const helpersTemplate = `/*
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

package {{.Package}}

import (
	"{{.StatePackage}}"
)

/*
 * super-state machine helper code follows
 */

type (
	// TODO(jdef) I don't like this name, need to revisit
	SuperMachine{{.Interface}} interface {
		{{.Interface}}
		state.Transition
		state.SuperMachine

		// SubMachine{{.Interface}} is a convenience func to create package-
		// specific sub-state machines.
		SubMachine{{.Interface}}(int, state.Fn) SubMachine{{.Interface}}
	}

	superMachine{{.Interface}}Impl struct {
		{{.Interface}}
		hijackChan chan state.Fn
	}
)

func AsSuperMachine(i {{.Interface}}) SuperMachine{{.Interface}} {
	return &superMachine{{.Interface}}Impl{ {{- .Interface}}: i, hijackChan: make(chan state.Fn)}
}

func (a *superMachine{{.Interface}}Impl) NextState() <-chan state.Fn                    { return a.hijackChan }
func (a *superMachine{{.Interface}}Impl) Hijack() chan<- state.Fn                       { return a.hijackChan }
func (a *superMachine{{.Interface}}Impl) SubMachine(l int, f state.Fn) state.SubMachine { return newSubMachine(a, l, f) }

func (a *superMachine{{.Interface}}Impl) SubMachine{{.Interface}}(queueLen int, initialFn state.Fn) SubMachine{{.Interface}} {
	return a.SubMachine(queueLen, initialFn).(SubMachine{{.Interface}})
}

/*
 * sub-state machine helper code follows
 */

type (
	// TODO(jdef) I don't like this name, need to revisit
	SubMachine{{.Interface}} interface {
		{{.Interface}}
		state.Transition
		state.SubMachine
	}

	// subMachine{{.Interface}}Impl is a helper for quickly building sub-state machines that
	// extend Agent functionality. Sub-state machines typically need their
	// own event queue and may want to override the initial state func.
	subMachine{{.Interface}}Impl struct {
		SuperMachine{{.Interface}}
		events       chan state.Event
		initialState state.Fn
	}
)

// subMachine{{.Interface}}Impl implements state.SubMachine{{.Interface}}
var _ SubMachine{{.Interface}} = &subMachine{{.Interface}}Impl{}

func newSubMachine(super SuperMachine{{.Interface}}, queueLength int, initialState state.Fn) state.SubMachine {
	return &subMachine{{.Interface}}Impl{
		SuperMachine{{.Interface}}: super,
		events:                make(chan state.Event, queueLength),
		initialState:          initialState,
	}
}

func (m *subMachine{{.Interface}}Impl) InitialState() state.Fn {
	if m.initialState != nil {
		return m.initialState
	}
	return m.SuperMachine{{.Interface}}.InitialState()
}

// Dispatch sends an event to the super-state machine. The super-state machine should
// probably have a buffered event queue if there's a party external to the state
// machine substrate that's also feeding events into the machine, otherwise this
// may block indefinitely.
func (m *subMachine{{.Interface}}Impl) Dispatch(ctx state.Context, e state.Event) {
	select {
	case <-ctx.Done():
		return
	case m.Super().(SuperMachine{{.Interface}}).Sink() <- e:
	}
}

func (m *subMachine{{.Interface}}Impl) Source() <-chan state.Event                { return m.events }
func (m *subMachine{{.Interface}}Impl) Sink() chan<- state.Event                  { return m.events }
func (m *subMachine{{.Interface}}Impl) Super() state.SuperMachine                 { return m.SuperMachine{{.Interface}} }
func (m *subMachine{{.Interface}}Impl) Hijack() chan<- state.Fn                   { return nil } // is not hijackable
func (m *subMachine{{.Interface}}Impl) NextState() <-chan state.Fn                { return nil } // is not hijackable
func (m *subMachine{{.Interface}}Impl) SubMachine(int, state.Fn) state.SubMachine { return nil } // is not hijackable

// Masquerade returns a reference to an imposter of the super-machine that may
// be passed to the super-machine's state handlers for upstream event delegation.
// The Source of the returned instance is expected to reference the source event
// stream of the actual super-machine. All other interface funcs may be overridden
// by the sub-machine implementation.
func Masquerade(m SubMachine{{.Interface}}) state.Machine { return &masq{{.Interface}}{m} }

type masq{{.Interface}} struct {
	SubMachine{{.Interface}}
}

// Source returns the upstream source so that we may pass this masq{{.Interface}} instance
// to a super-state handler and it will read events from its own source, instead
// of sub-machine's.
func (m *masq{{.Interface}}) Source() <-chan state.Event { return m.Super().(SuperMachine{{.Interface}}).Source() }

// NextState returns the upstream NextState so that we may pass this masq{{.Interface}} instance
// to a super-state handler and it will read states from the super-machine helper
// associated with this sub-state machine.
func (m *masq{{.Interface}}) NextState() <-chan state.Fn { return m.Super().(SuperMachine{{.Interface}}).NextState() }

// Super is a convenience func that returns the super-state machine as {{.Interface}}
func SuperOf(sub SubMachine{{.Interface}}) {{.Interface}} {
	return sub.Super().({{.Interface}})
}

// Sub is a convenience func that returns the given Machine as a SubMachine{{.Interface}}
func AsSub(m state.Machine) SubMachine{{.Interface}} {
	return m.(SubMachine{{.Interface}})
}
`
