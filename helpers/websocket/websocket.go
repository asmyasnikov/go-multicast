package websocket

import (
	"encoding/json"
	"github.com/asmyasnikov/go-multicast"
	"github.com/gorilla/websocket"
	"math"
	"net/http"
	"strconv"
	"time"
)

var (
	// Upgrader is a helper for upgrade http connection to websocket connection
	Upgrader = websocket.Upgrader{
		ReadBufferSize:  128,
		WriteBufferSize: 128,
		CheckOrigin: func(_ *http.Request) bool {
			return true
		},
	}
)

const (
	setDelay = "_DELAY="
	minDelay = time.Duration(0)
	maxDelay = time.Second
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
			if len(data) > len(setDelay) && string(data[:len(setDelay)]) == setDelay {
				v, err := strconv.ParseFloat(string(data[len(setDelay):]), 64)
				if err != nil {
					return multicast.ChangeIntervalMessage{Error: err}, nil
				}
				d := time.Duration(v) * time.Millisecond
				if d > maxDelay || d < minDelay {
					d = time.Duration(
						math.Min(
							float64(maxDelay.Milliseconds()),
							math.Max(
								float64(minDelay.Milliseconds()),
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
			b, err := json.Marshal(msg)
			if err != nil {
				return err
			}
			return conn.WriteMessage(websocket.TextMessage, b)
		},
		nil,
	)
}
