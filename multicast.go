// Package multicast provides multicast
// helpers over websocket.
package multicast

import (
	"context"
	"reflect"
	"sync"
	"time"
)

type Multicast struct {
	mtx          sync.RWMutex
	snapshot     interface{}
	clients      map[*client]struct{}
	merge        func(source interface{}, diff interface{}) (merged interface{})
	onError      func(error)
	defaultDelay time.Duration
}

// Creates new Multicast with empty connections and channels
func New(
	ctx context.Context,
	onError func(error),
	merge func(source interface{}, diff interface{}) (merged interface{}),
	defaultDelay time.Duration,
) *Multicast {
	m := &Multicast{
		clients: make(map[*client]struct{}),
		merge: func(source interface{}, diff interface{}) (merged interface{}) {
			if source == nil || merge == nil {
				return diff
			}
			return merge(source, diff)
		},
		onError: func(err error) {
			if onError != nil {
				onError(err)
			}
		},
		defaultDelay: defaultDelay,
	}
	go func() {
		<-ctx.Done()
		m.mtx.RLock()
		defer m.mtx.RUnlock()
		for c := range m.clients {
			c.close()
		}
	}()
	return m
}

func (m *Multicast) Add(
	read func() (msg interface{}, err error),
	write func(msg interface{}) (err error),
	onDone func(),
) (received <-chan interface{}, done chan <- struct{}) {
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
) (received <-chan interface{}, done chan <- struct{}) {
	c := newClient(
		read,
		write,
		func() interface{} {
			m.mtx.RLock()
			defer m.mtx.RUnlock()
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
		m.defaultDelay,
	)
	m.mtx.Lock()
	m.clients[c] = struct{}{}
	m.mtx.Unlock()
	return c.received, c.done
}

func (m *Multicast) SendAll(data interface{}) error {
	if reflect.ValueOf(data).Type().Kind() != reflect.Ptr {
		panic("expected only pointer to data")
	}
	m.mtx.RLock()
	defer m.mtx.RUnlock()
	for c := range m.clients {
		select {
		case _ = <-c.done:
			delete(m.clients, c)
		default:
			c.send <- data
		}
	}
	m.snapshot = m.merge(m.snapshot, data)
	return nil
}
