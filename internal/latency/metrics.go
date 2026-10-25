package latency

import (
	"fmt"
	"io"
	"sync/atomic"
	"time"
)

// Fixed histograms avoid allocating or sorting on the audio thread. Percentile
// values are bucket upper bounds; final-bin overflow uses the observed maximum.
type histogram struct {
	count, total, maximum atomic.Uint64
	bins                  [2048]atomic.Uint64
	quantum               uint64
}

func (h *histogram) add(d time.Duration) {
	n := uint64(max(0, d))
	h.total.Add(n)
	h.count.Add(1)
	for old := h.maximum.Load(); n > old && !h.maximum.CompareAndSwap(old, n); old = h.maximum.Load() {
	}
	h.bins[min(n/h.quantum, uint64(len(h.bins)-1))].Add(1)
}
func (h *histogram) summary() string {
	count := h.count.Load()
	if count == 0 {
		return "no samples"
	}
	rank := (count*99 + 99) / 100
	sum := uint64(0)
	p99 := h.maximum.Load()
	for i := range h.bins {
		sum += h.bins[i].Load()
		if sum >= rank {
			p99 = min(uint64(i+1)*h.quantum, h.maximum.Load())
			if i == len(h.bins)-1 {
				p99 = h.maximum.Load()
			}
			break
		}
	}
	return fmt.Sprintf("n=%d mean=%.3f ms p99<=%.3f ms max=%.3f ms", count, float64(h.total.Load())/float64(count)/1e6, float64(p99)/1e6, float64(h.maximum.Load())/1e6)
}

type mode uint8

const (
	streaming mode = iota
	cached
	mixed
)

func (m mode) String() string { return [...]string{"stream", "cached", "mixed"}[m] }

type modeStats struct {
	processing, submission, setup, queued histogram
	overBudget                            atomic.Uint64
}
type statistics struct {
	modes                         [3]modeStats
	emptyQueue, clipped, rejected atomic.Uint64
	wakeGap                       histogram
}

func newStatistics() *statistics {
	s := &statistics{}
	s.wakeGap.quantum = 10000
	for i := range s.modes {
		m := &s.modes[i]
		m.processing.quantum = 1000
		m.submission.quantum = 10000
		m.setup.quantum = 1000
		m.queued.quantum = 10000
	}
	return s
}
func (s *statistics) reportProcessing(w io.Writer) {
	for i := range s.modes {
		m := &s.modes[i]
		if m.processing.count.Load() == 0 {
			continue
		}
		fmt.Fprintf(w, "%s processing: %s; over block budget=%d\n", mode(i), m.processing.summary(), m.overBudget.Load())
		fmt.Fprintf(w, "%s voice setup: %s\n", mode(i), m.setup.summary())
	}
}
func (s *statistics) report(w io.Writer, p *pressure) {
	fmt.Fprintln(w, "\n--- Measurements since launch ---")
	s.reportProcessing(w)
	for i := range s.modes {
		m := &s.modes[i]
		if m.submission.count.Load() == 0 {
			continue
		}
		fmt.Fprintf(w, "%s trigger received -> first buffer submitted: %s\n", mode(i), m.submission.summary())
		fmt.Fprintf(w, "%s queued audio observed before refill: %s\n", mode(i), m.queued.summary())
	}

	fmt.Fprintf(w, "Device wake gap: %s\n", s.wakeGap.summary())
	fmt.Fprintf(w, "Empty queue observations while audio was active: %d (possible starvation; not an exact glitch count)\n", s.emptyQueue.Load())
	fmt.Fprintf(w, "Clipped output samples: %d; rejected triggers: %d; stress allocation: %.1f MiB\n", s.clipped.Load(), s.rejected.Load(), float64(p.bytes.Load())/(1024*1024))
	fmt.Fprintln(w, "Trigger timing starts when this program reads the key or handles its automatic timer, includes voice setup, and excludes unknown device/physical latency.")
}
