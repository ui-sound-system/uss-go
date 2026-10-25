package latency

import (
	"fmt"
	"io"
	"time"
)

func benchmark(o options, s *source, p *pressure, out io.Writer) error {
	fmt.Fprintln(out, "CPU-only test: no audio device or physical latency measurement. Setup is measured separately.")
	stats := newStatistics()
	output := make([]float32, o.block*2)
	for _, mode := range []mode{streaming, cached} {
		m := newMixer(o.block, stats)
		for i := 0; i < o.iterations; i++ {
			if m.count == 0 {
				t, err := s.trigger(mode, o.voices, time.Now())
				if err != nil {
					m.close()
					return err
				}
				m.enqueue(t)
			}
			if err := m.render(output); err != nil {
				m.close()
				return err
			}
			m.submitted(time.Now(), 0)
		}
		m.close()
	}
	stats.reportProcessing(out)
	fmt.Fprintf(out, "Stress allocated %.1f MiB during this run. No device timing was measured.\n", float64(p.bytes.Load())/(1024*1024))
	return nil
}
