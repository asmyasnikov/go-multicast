package multicast

import (
	"errors"
	"fmt"
	"github.com/stretchr/testify/require"
	"sync"
	"testing"
	"time"
)

func TestClientChangeIntervalMessage(t *testing.T) {
	type msg struct {
		I int
	}
	delay := time.Millisecond * 10
	wg := sync.WaitGroup{}
	wg.Add(10)
	prev := time.Now()
	i := 9
	readCount := 0
	ch := make(chan struct{})
	c := newClient(
		func() (interface{}, error) {
			if readCount == 0 {
				defer func() {
					readCount++
				}()
				return ChangeIntervalMessage{Interval: delay}, nil
			}
			wg.Wait()
			close(ch)
			return nil, fmt.Errorf("EOF")
		},
		func(msg interface{}) error {
			defer wg.Done()
			b, err := Json(msg)
			require.NoError(t, err)
			require.Equal(t, fmt.Sprintf("{\"I\":%d}", i), string(b))
			i += 10
			since := time.Since(prev)
			require.GreaterOrEqual(t, since.Milliseconds(), delay.Milliseconds())
			prev = time.Now()
			return nil
		},
		nil,
		nil,
		nil,
		nil,
		0,
	)
	defer c.close()
	for i := 0; i < 100; i++ {
		if i%10 == 0 {
			time.Sleep(time.Millisecond * 100)
		}
		c.send <- msg{
			I: i,
		}
	}
	select {
	case <-ch:
	case <-time.After(time.Second):
		require.Error(t, errors.New("timeout elapsed"))
	}
}
