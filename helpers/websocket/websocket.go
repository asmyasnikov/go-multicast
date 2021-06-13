package websocket

import (
	"github.com/asmyasnikov/go-multicast"
	"github.com/gorilla/websocket"
	"math"
	"strconv"
	"time"
)

var (
	// Upgrader is a helper for upgrade http connection to websocket connection
	Upgrader = websocket.Upgrader{
		ReadBufferSize:  128,
		WriteBufferSize: 128,
	}
)

const (
	_SET_DELAY = "_DELAY="
	_MIN_DELAY = time.Duration(0)
	_MAX_DELAY = time.Second
)

// Add helps to add websocket connection into multicast communicator
func Add(
	m *multicast.Multicast,
	conn *websocket.Conn,
	unmarshall func(b []byte) (msg interface{}, err error),
) (received <-chan interface{}, done chan<- struct{}) {
	return m.Add(
		func() (msg interface{}, err error) {
			_, data, err := conn.ReadMessage()
			if err != nil {
				return nil, err
			}
			if len(data) == 0 {
				return nil, nil
			}
			if len(data) > len(_SET_DELAY) && string(data[:len(_SET_DELAY)]) == _SET_DELAY {
				v, err := strconv.ParseFloat(string(data[len(_SET_DELAY):]), 64)
				if err != nil {
					return multicast.ChangeIntervalMessage{Error: err}, nil
				}
				d := time.Duration(v) * time.Millisecond
				if d > _MAX_DELAY || d < _MIN_DELAY {
					d = time.Duration(
						math.Min(
							float64(_MAX_DELAY.Milliseconds()),
							math.Max(
								float64(_MIN_DELAY.Milliseconds()),
								float64(d.Milliseconds()),
							),
						),
					) * time.Millisecond
				}
				return multicast.ChangeIntervalMessage{Interval: d}, nil
			}
			return unmarshall(data)
		},
		func(msg interface{}) error {
			b, err := multicast.Json(msg)
			if err != nil {
				return err
			}
			return conn.WriteMessage(websocket.TextMessage, b)
		},
		nil,
	)
}
