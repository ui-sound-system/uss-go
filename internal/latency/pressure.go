package latency

import (
	"runtime"
	"sync/atomic"
	"time"
)

type pressure struct {
	enabled    atomic.Bool
	bytes      atomic.Uint64
	stop, done chan struct{}
}

func newPressure(enabled bool) *pressure {
	p := &pressure{stop: make(chan struct{}), done: make(chan struct{})}
	p.enabled.Store(enabled)
	go func() {
		defer close(p.done)
		ticker := time.NewTicker(time.Millisecond)
		defer ticker.Stop()
		retained := make([][]byte, 16)
		index := 0
		for {
			select {
			case <-p.stop:
				return
			case <-ticker.C:
				if !p.enabled.Load() {
					continue
				}
				data := make([]byte, 64*1024)
				for i := 0; i < len(data); i += 4096 {
					data[i] = byte(index)
				}
				retained[index%len(retained)] = data
				index++
				p.bytes.Add(uint64(len(data)))
				runtime.KeepAlive(retained)
			}
		}
	}()
	return p
}
func (p *pressure) close() { close(p.stop); <-p.done }
