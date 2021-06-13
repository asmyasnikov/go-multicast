package multicast

import (
	"sync"
	"time"
)

type client struct {
	closeOnce sync.Once
	done      chan struct{}
	onDone    func()

	send     chan interface{}
	received chan interface{}

	intervalChan chan time.Duration
}

func newClient(
	read func() (msg interface{}, err error),
	write func(msg interface{}) (err error),
	snapshot interface{},
	onDone func(c *client),
	onError func(err error),
	merge func(source interface{}, diff interface{}) (merged interface{}),
	interval time.Duration,
) *client {
	c := &client{
		done:         make(chan struct{}),
		send:         make(chan interface{}, 20),
		received:     make(chan interface{}, 20),
		intervalChan: make(chan time.Duration),
	}
	if onDone != nil {
		c.onDone = func() {
			onDone(c)
		}
	}
	if merge == nil {
		merge = func(_ interface{}, diff interface{}) (merged interface{}) {
			return diff
		}
	}
	w := func(msg interface{}) {
		if msg == nil {
			return
		}
		if err := write(msg); err != nil {
			onError(err)
		}
	}
	if snapshot != nil {
		w(snapshot)
	}
	if read != nil {
		go c.readLoop(read, onError)
	}
	go c.writeLoop(w, merge, interval)
	return c
}

func (c *client) readLoop(
	read func() (msg interface{}, err error),
	onError func(err error),
) {
	defer c.close()
	for {
		select {
		case <-c.done:
			return
		default:
			msg, err := read()
			if err != nil {
				if onError != nil {
					onError(err)
				}
				return
			}
			if msg == nil {
				continue
			}
			select {
			case <-c.done:
				return
			default:
				switch t := msg.(type) {
				case ChangeIntervalMessage:
					if t.Error != nil {
						if onError != nil {
							onError(t.Error)
						}
					} else {
						c.intervalChan <- t.Interval
					}
				default:
					c.received <- msg
				}
			}
		}
	}
}

func (c *client) writeLoop(
	write func(msg interface{}),
	merge func(source interface{}, diff interface{}) (merged interface{}),
	delay time.Duration,
) {
	var accumulated interface{}
	var next time.Time
	update := func(d time.Duration) {
		delay = d
		if d != 0 {
			next = time.Now().Add(d)
		} else {
			next = time.Now().Add(time.Hour * 24)
		}
	}
	for {
		select {
		case <-c.done:
			return
		case d, ok := <-c.intervalChan:
			if !ok {
				return
			}
			update(d)
		case msg, ok := <-c.send:
			if !ok {
				return
			}
			if msg == nil {
				continue
			}
			if delay == 0 {
				write(msg)
				accumulated = nil
			} else {
				accumulated = merge(accumulated, msg)
			}
		case <-time.After(time.Until(next)):
			select {
			case <-c.done:
				return
			default:
				if accumulated != nil {
					write(accumulated)
					accumulated = nil
				}
				update(delay)
			}
		}
	}
}

func (c *client) close() {
	c.closeOnce.Do(func() {
		close(c.done)
		close(c.received)
		close(c.intervalChan)
		if c.onDone != nil {
			c.onDone()
		}
	})
}
