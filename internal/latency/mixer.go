package latency

import (
	"fmt"
	"io"
	"math"
	"time"

	"github.com/ui-sound-system/uss-go/synth"
)

const maxVoices = 128
const maxPending = 8

type voice struct {
	stream   *synth.Stream
	samples  []float32
	position int
	gain     float32
	mode     mode
}
type trigger struct {
	voices    []voice
	requested time.Time
	setup     time.Duration
	mode      mode
}

func (t *trigger) close() {
	for i := range t.voices {
		if t.voices[i].stream != nil {
			t.voices[i].stream.Close()
		}
	}
}
func (s *source) trigger(m mode, count int, requested time.Time) (trigger, error) {
	start := time.Now()
	t := trigger{voices: make([]voice, count), requested: requested, mode: m}
	for i := range t.voices {
		v := &t.voices[i]
		v.gain = 1 / float32(count)
		v.mode = m
		if m == cached {
			v.samples = s.cached
		} else {
			var err error
			v.stream, err = s.prepared.Stream(s.inputs)
			if err != nil {
				t.close()
				return trigger{}, err
			}
		}
	}
	t.setup = time.Since(start)
	return t, nil
}

type mixer struct {
	commands      chan trigger
	active        [maxVoices]voice
	count         int
	scratch, mono []float32
	pending       [maxPending]trigger
	pendingCount  int
	stats         *statistics
}

func newMixer(block int, stats *statistics) *mixer {
	return &mixer{commands: make(chan trigger, maxPending), scratch: make([]float32, block), mono: make([]float32, block), stats: stats}
}
func (m *mixer) enqueue(t trigger) bool {
	select {
	case m.commands <- t:
		return true
	default:
		t.close()
		m.stats.rejected.Add(1)
		return false
	}
}
func (m *mixer) accept() {
	for i := 0; i < maxPending; i++ {
		select {
		case t := <-m.commands:
			if len(t.voices) > maxVoices-m.count {
				t.close()
				m.stats.rejected.Add(1)
				continue
			}
			copy(m.active[m.count:], t.voices)
			m.count += len(t.voices)
			m.pending[m.pendingCount] = t
			m.pendingCount++
		default:
			return
		}
	}
}
func (m *mixer) category() mode {
	result := m.active[0].mode
	for i := 1; i < m.count; i++ {
		if m.active[i].mode != result {
			return mixed
		}
	}
	return result
}

// render writes stereo interleaved Float32 frames. Only this owner touches voice
// state. Preparing commands, reporting and heap pressure run on other goroutines.
func (m *mixer) render(output []float32) error {
	if m.pendingCount != 0 {
		return fmt.Errorf("previous buffer was not submitted")
	}
	m.accept()
	for offset := 0; offset < len(output)/2; {
		n := min(len(m.mono), len(output)/2-offset)
		clear(m.mono[:n])
		active := m.count > 0
		category := streaming
		if active {
			category = m.category()
		}
		start := time.Now()
		for i := 0; i < m.count; {
			v := &m.active[i]
			count := 0
			done := false
			if v.stream != nil {
				var err error
				count, err = v.stream.ReadSamples(m.scratch[:n])
				done = err == io.EOF
				if err != nil && !done {
					return err
				}
			} else {
				count = copy(m.scratch[:n], v.samples[v.position:])
				v.position += count
				done = v.position == len(v.samples)
			}
			for j := 0; j < count; j++ {
				m.mono[j] += m.scratch[j] * v.gain
			}
			if done {
				if v.stream != nil {
					v.stream.Close()
				}
				m.count--
				m.active[i] = m.active[m.count]
				m.active[m.count] = voice{}
			} else {
				i++
			}
		}
		clipped := uint64(0)
		for i := 0; i < n; i++ {
			sample := m.mono[i]
			if math.IsNaN(float64(sample)) || math.IsInf(float64(sample), 0) {
				return fmt.Errorf("non-finite mixed output")
			}
			if sample > 1 {
				sample = 1
				clipped++
			}
			if sample < -1 {
				sample = -1
				clipped++
			}
			output[(offset+i)*2], output[(offset+i)*2+1] = sample, sample
		}
		if active {
			elapsed := time.Since(start)
			stats := &m.stats.modes[category]
			stats.processing.add(elapsed)
			if elapsed > time.Second*time.Duration(n)/synth.SampleRate {
				stats.overBudget.Add(1)
			}
			m.stats.clipped.Add(clipped)
		}
		offset += n
	}
	return nil
}
func (m *mixer) submitted(now time.Time, padding uint32) {
	for i := 0; i < m.pendingCount; i++ {
		t := &m.pending[i]
		stats := &m.stats.modes[t.mode]
		stats.submission.add(now.Sub(t.requested))
		stats.setup.add(t.setup)
		stats.queued.add(time.Second * time.Duration(padding) / synth.SampleRate)
		*t = trigger{}
	}
	m.pendingCount = 0
}
func (m *mixer) close() {
	for i := 0; i < m.count; i++ {
		if m.active[i].stream != nil {
			m.active[i].stream.Close()
		}
		m.active[i] = voice{}
	}
	for {
		select {
		case t := <-m.commands:
			t.close()
		default:
			return
		}
	}
}
