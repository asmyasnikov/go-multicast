// Package multicast provides multicast
// helpers over websocket.
package multicast

import (
	"context"
	"reflect"
	"sync"
	"time"
)

// Multicast is a main communicator for linking sender and receivers
type Multicast struct {
	mtx             sync.RWMutex
	snapshot        interface{}
	clients         map[*client]struct{}
	merge           func(source interface{}, diff interface{}) (merged interface{})
	onError         func(error)
	defaultInterval time.Duration
}

// New creates new Multicast with empty clients
func New(
	ctx context.Context,
	onError func(error),
	merge func(source interface{}, diff interface{}) (merged interface{}),
	initInterval time.Duration,
) *Multicast {
	m := &Multicast{
		clients: make(map[*client]struct{}),
		merge:   merge,
		onError: func(err error) {
			if onError != nil {
				onError(err)
			}
		},
		defaultInterval: initInterval,
	}
	go func() {
		<-ctx.Done()
		m.mtx.RLock()
		wg := sync.WaitGroup{}
		for c := range m.clients {
			wg.Add(1)
			go func(c *client) {
				defer wg.Done()
				c.close()
			}(c)
		}
		m.mtx.RUnlock()
		wg.Wait()
	}()
	return m
}

// Add appends some client to multicast
func (m *Multicast) Add(
	read func() (msg interface{}, err error),
	write func(msg interface{}) (err error),
	onDone func(),
) (received <-chan interface{}, done chan<- struct{}) {
	return m.add(
		read,
		write,
		onDone,
	)
}

func (m *Multicast) add(
	read func() (msg interface{}, err error),
	write func(msg interface{}) (err error),
	onDone func(),
) (received <-chan interface{}, done chan<- struct{}) {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	c := newClient(
		read,
		write,
		func() interface{} {
			return m.snapshot
		}(),
		func(c *client) {
			m.mtx.Lock()
			delete(m.clients, c)
			m.mtx.Unlock()
			if onDone != nil {
				onDone()
			}
		},
		m.onError,
		m.merge,
		m.defaultInterval,
	)
	m.clients[c] = struct{}{}
	return c.received, c.done
}

// SendAll provide sending message to multiple clients
func (m *Multicast) SendAll(msg interface{}) error {
	if reflect.ValueOf(msg).Type().Kind() != reflect.Ptr {
		panic("expected only pointer to msg")
	}
	m.mtx.RLock()
	defer m.mtx.RUnlock()
	for c := range m.clients {
		select {
		case _ = <-c.done:
			delete(m.clients, c)
		default:
			c.send <- msg
		}
	}
	m.snapshot = m.merge(m.snapshot, msg)
	return nil
}
