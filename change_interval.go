package multicast

import "time"

// ChangeIntervalMessage is a specified message for change interval of sending high-frequency messages
type ChangeIntervalMessage struct {
	Interval time.Duration
	Error    error
}
