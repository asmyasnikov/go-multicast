package multicast

import "time"

type ChangeIntervalMessage struct {
	Interval time.Duration
	Error    error
}
