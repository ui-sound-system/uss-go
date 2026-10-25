package latency

import (
	"bytes"
	"io"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/ui-sound-system/uss-go/synth"
)

func testSource(t testing.TB) *source {
	t.Helper()
	d, err := synth.Parse(builtIn)
	if err != nil {
		t.Fatal(err)
	}
	p, err := d.Prepare()
	if err != nil {
		t.Fatal(err)
	}
	s, err := p.Render(nil)
	if err != nil {
		t.Fatal(err)
	}
	return &source{prepared: p, cached: s.Samples, name: d.Name}
}
func TestStreamingAndCachedMixerParity(t *testing.T) {
	src := testSource(t)
	for _, voices := range []int{1, 4, 16} {
		for _, block := range []int{1, 64, 256, 1024} {
			render := func(mode mode) []float32 {
				m := newMixer(block, newStatistics())
				defer m.close()
				tr, err := src.trigger(mode, voices, time.Now())
				if err != nil {
					t.Fatal(err)
				}
				m.enqueue(tr)
				var samples []float32
				for first := true; first || m.count > 0; first = false {
					// Device packet sizes need not equal the requested synthesis block.
					buffer := make([]float32, 346)
					if err := m.render(buffer); err != nil {
						t.Fatal(err)
					}
					m.submitted(time.Now(), 0)
					samples = append(samples, buffer...)
				}
				return samples
			}
			streamed, cached := render(streaming), render(cached)
			if len(streamed) != len(cached) {
				t.Fatal("mode duration differs")
			}
			for i := range streamed {
				if streamed[i] != cached[i] {
					t.Fatalf("voices=%d block=%d sample=%d", voices, block, i)
				}
			}
			for i := 0; i < len(streamed); i += 2 {
				if streamed[i] != streamed[i+1] {
					t.Fatal("stereo channels differ")
				}
			}
		}
	}
}
func TestMixerProcessingAllocations(t *testing.T) {
	src := testSource(t)
	m := newMixer(64, newStatistics())
	defer m.close()
	tr, err := src.trigger(streaming, 16, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	m.enqueue(tr)
	buffer := make([]float32, 128)
	if allocations := testing.AllocsPerRun(100, func() {
		if err := m.render(buffer); err != nil {
			t.Error(err)
		}
		m.submitted(time.Now(), 0)
	}); allocations != 0 {
		t.Fatalf("allocations per render=%g", allocations)
	}
}
func TestQueuedAndActiveVoiceLimits(t *testing.T) {
	src := testSource(t)
	stats := newStatistics()
	m := newMixer(64, stats)
	defer m.close()
	for i := 0; i < maxPending; i++ {
		tr, err := src.trigger(streaming, 16, time.Now())
		if err != nil {
			t.Fatal(err)
		}
		if !m.enqueue(tr) {
			t.Fatal("unexpected queue rejection")
		}
	}
	rejected, err := src.trigger(streaming, 1, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if m.enqueue(rejected) {
		t.Fatal("overfull command queue accepted")
	}
	if _, err := rejected.voices[0].stream.ReadSamples(make([]float32, 1)); err != io.EOF {
		t.Fatal("rejected voice not closed")
	}
	buffer := make([]float32, 128)
	if err := m.render(buffer); err != nil {
		t.Fatal(err)
	}
	m.submitted(time.Now(), 128)
	if m.count != maxVoices {
		t.Fatalf("active count=%d", m.count)
	}
	extra, err := src.trigger(streaming, 1, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	m.enqueue(extra)
	if err := m.render(buffer); err != nil {
		t.Fatal(err)
	}
	m.submitted(time.Now(), 128)
	if stats.rejected.Load() != 2 {
		t.Fatal("active limit not reported")
	}
	if _, err := extra.voices[0].stream.ReadSamples(make([]float32, 1)); err != io.EOF {
		t.Fatal("rejected active voice not closed")
	}
	if stats.clipped.Load() == 0 {
		t.Fatal("overlapping trigger clipping not reported")
	}
}
func TestPeriodSelection(t *testing.T) {
	for _, c := range []struct {
		requested                   int
		fundamental, min, max, want uint32
	}{{0, 16, 128, 480, 128}, {64, 16, 128, 480, 128}, {129, 16, 128, 480, 144}, {1000, 16, 128, 480, 480}, {0, 4, 48, 448, 48}} {
		got, err := choosePeriod(c.requested, c.fundamental, c.min, c.max)
		if err != nil || got != c.want {
			t.Fatalf("period %d: %d %v", c.requested, got, err)
		}
	}
	if _, err := choosePeriod(0, 0, 1, 10); err == nil {
		t.Fatal("invalid driver periods accepted")
	}
}
func TestCommandLine(t *testing.T) {
	for _, args := range [][]string{{"--block", "0"}, {"--voices", "33"}, {"--mode", "wrong"}, {"--seconds", "NaN"}, {"--seconds", "+Inf"}, {"--set", "wrong"}, {"one.json", "two.json"}} {
		if err := Run(args, io.Discard, io.Discard); err == nil {
			t.Fatalf("accepted %v", args)
		}
	}
	var out bytes.Buffer
	if err := Run([]string{"--bench", "--iterations", "3", "--set", "note=G5"}, &out, io.Discard); err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{"stream processing:", "cached processing:", "CPU-only test"} {
		if !strings.Contains(out.String(), text) {
			t.Fatalf("missing %s", text)
		}
	}
}
func TestHistogramAndSubmission(t *testing.T) {
	stats := newStatistics()
	m := newMixer(64, stats)
	defer m.close()
	src := testSource(t)
	now := time.Now()
	tr, err := src.trigger(cached, 1, now)
	if err != nil {
		t.Fatal(err)
	}
	m.enqueue(tr)
	if err := m.render(make([]float32, 128)); err != nil {
		t.Fatal(err)
	}
	m.submitted(now.Add(2*time.Millisecond), 48)
	if got := stats.modes[cached].submission.maximum.Load(); got != uint64(2*time.Millisecond) {
		t.Fatal("trigger delay includes wrong boundary")
	}
	if got := stats.modes[cached].queued.maximum.Load(); got != uint64(time.Millisecond) {
		t.Fatal("wrong queue duration")
	}
	for _, sample := range src.cached {
		if math.IsNaN(float64(sample)) {
			t.Fatal("non-finite built-in sound")
		}
	}
}
