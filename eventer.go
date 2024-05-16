package gobot

import (
	"fmt"
	"sync"
)

type eventChannel chan *Event

type eventer struct {
	// map of valid Event names
	eventnames map[string]string

	// new events get put in to the event channel
	in eventChannel

	// map of out channels used by subscribers
	outs map[eventChannel]eventChannel

	// map of event and channel
	mapping map[eventChannel]string

	// mutex to protect the eventChannel map
	eventsMutex sync.RWMutex
}

const eventChanBufferSize = 10

// Eventer is the interface which describes how a Driver or Adaptor
// handles events.
type Eventer interface {
	// Events returns the map of valid Event names.
	Events() (eventnames map[string]string)

	// Event returns an Event string from map of valid Event names.
	// Mostly used to validate that an Event name is valid.
	Event(name string) string

	// AddEvent registers a new Event name.
	AddEvent(name string)

	// DeleteEvent removes a previously registered Event name.
	DeleteEvent(name string)

	// Publish new events to any subscriber
	Publish(name string, data interface{})

	// Subscribe to events
	Subscribe() (events eventChannel)

	// Unsubscribe from an event channel
	Unsubscribe(events eventChannel)

	// Event handler
	On(name string, f func(s interface{})) error

	// Event handler, only executes one time
	Once(name string, f func(s interface{})) error

	// Metrics collects all channel length
	Metrics() map[string]int
}

// NewEventer returns a new Eventer.
func NewEventer() Eventer {
	evtr := &eventer{
		eventnames: make(map[string]string),
		in:         make(eventChannel, eventChanBufferSize),
		outs:       make(map[eventChannel]eventChannel),
		mapping:    make(map[eventChannel]string),
	}

	// goroutine to cascade "in" events to all "out" event channels
	go func() {
		for {
			evt := <-evtr.in
			evtr.eventsMutex.RLock()
			for _, out := range evtr.outs {
				out <- evt
			}
			evtr.eventsMutex.RUnlock()
		}
	}()

	return evtr
}

// Events returns the map of valid Event names.
func (e *eventer) Events() map[string]string {
	return e.eventnames
}

// Event returns an Event string from map of valid Event names.
// Mostly used to validate that an Event name is valid.
func (e *eventer) Event(name string) string {
	return e.eventnames[name]
}

// AddEvent registers a new Event name.
func (e *eventer) AddEvent(name string) {
	e.eventnames[name] = name
}

// DeleteEvent removes a previously registered Event name.
func (e *eventer) DeleteEvent(name string) {
	delete(e.eventnames, name)
}

// Publish new events to anyone that is subscribed
func (e *eventer) Publish(name string, data interface{}) {
	evt := NewEvent(name, data)
	e.in <- evt
}

// Subscribe to any events from this eventer
func (e *eventer) Subscribe() eventChannel {
	e.eventsMutex.Lock()
	defer e.eventsMutex.Unlock()
	out := make(eventChannel, eventChanBufferSize)
	e.outs[out] = out
	return out
}

// Unsubscribe from the event channel
func (e *eventer) Unsubscribe(events eventChannel) {
	e.eventsMutex.Lock()
	defer e.eventsMutex.Unlock()
	delete(e.outs, events)
	delete(e.mapping, events)
}

// On executes the event handler f when e is Published to.
func (e *eventer) On(n string, f func(s interface{})) error {
	out := e.Subscribe()
	e.eventsMutex.Lock()
	e.mapping[out] = n
	e.eventsMutex.Unlock()
	go func() {
		for {
			evt := <-out
			if evt.Name == n {
				f(evt.Data)
			}
		}
	}()

	return nil
}

// Once is similar to On except that it only executes f one time.
func (e *eventer) Once(n string, f func(s interface{})) error {
	out := e.Subscribe()
	e.eventsMutex.Lock()
	e.mapping[out] = n
	e.eventsMutex.Unlock()
	go func() {
	ProcessEvents:
		for evt := range out {
			if evt.Name == n {
				f(evt.Data)
				e.Unsubscribe(out)
				break ProcessEvents
			}
		}
	}()

	return nil
}

// Metrics collects all channel length
func (e *eventer) Metrics() map[string]int {
	e.eventsMutex.RLock()
	defer e.eventsMutex.RUnlock()
	metrics := make(map[string]int)
	metrics["in"] = len(e.in)
	for channel, event := range e.mapping {
		metrics[fmt.Sprintf("out_%s", event)] = len(channel)
	}
	return metrics
}
