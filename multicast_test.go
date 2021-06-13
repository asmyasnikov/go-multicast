package multicast

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/require"
	"sync"
	"testing"
	"time"
)

func TestMulticast_SendAll(t *testing.T) {
	type msg struct {
		idx  uint64
		idxs []uint64
	}
	interval := time.Millisecond * 100
	limit := time.Second * 5
	ctx, cancel := context.WithTimeout(context.Background(), limit)
	defer cancel()
	m := New(
		ctx,
		nil,
		func(source interface{}, diff interface{}) (merged interface{}) {
			if source == nil {
				source = &msg{idxs: []uint64{}}
			}
			_source := source.(*msg)
			_diff := diff.(*msg)
			return &msg{
				idxs: append(_source.idxs, _diff.idx),
			}
		},
		interval,
	)
	snapshots := make(map[int]*struct {
		count int
		idxs  []uint64
	})
	mtx := sync.Mutex{}
	wg := sync.WaitGroup{}
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			now := time.Now()
			m.Add(
				func() (msg interface{}, err error) {
					time.Sleep(limit)
					return nil, fmt.Errorf("EOF")
				},
				func(m interface{}) (err error) {
					mtx.Lock()
					defer mtx.Unlock()
					if _, ok := snapshots[i]; !ok {
						snapshots[i] = &struct {
							count int
							idxs  []uint64
						}{
							count: 0,
							idxs:  []uint64{},
						}
					}
					snapshots[i].count++
					snapshots[i].idxs = append(snapshots[i].idxs, m.(*msg).idxs...)
					return nil
				},
				func() {
					defer wg.Done()
					mtx.Lock()
					defer mtx.Unlock()
					d, ok := snapshots[i]
					require.True(t, ok)
					intervals := time.Since(now) / interval
					require.GreaterOrEqual(t, int(intervals)+1, d.count)
					require.Greater(t, len(d.idxs), 0)
					for j, v := range d.idxs {
						require.Equal(t, uint64(j), v)
					}
				},
			)
		}(i)
	}
	deadline := time.Now().Add(limit)
	for i := uint64(0); time.Until(deadline) > 0; i++ {
		m.SendAll(&msg{idx: i})
		time.Sleep(time.Millisecond * 10)
	}
	wg.Wait()
}
