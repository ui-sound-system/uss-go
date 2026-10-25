package synth

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"sort"
)

// BlockSize bounds internal scratch space. ReadSamples accepts any buffer size.
const BlockSize = 256
const maxProcessors = 4096

// Prepared owns a snapshot of a validated definition. It may create concurrent,
// independent voices. Neither the definition nor caller input maps are retained.
type Prepared struct{ definition *Definition }

// Prepare snapshots and validates the graph without synthesizing audio.
// Parameter-dependent validation is repeated when a voice is instantiated.
func (d *Definition) Prepare() (*Prepared, error) {
	data, err := json.Marshal(d)
	if err != nil {
		return nil, err
	}
	snapshot, err := Parse(data)
	if err != nil {
		return nil, err
	}
	voice, err := snapshot.Stream(nil)
	if err != nil {
		return nil, err
	}
	voice.Close()
	return &Prepared{snapshot}, nil
}

// Stream creates an independent voice, binding inputs and allocating DSP state.
// Instantiate voices outside the audio callback. Concurrent modification of d
// during this call is unsupported; use Prepare for an owned reusable snapshot.
func (d *Definition) Stream(inputs Inputs) (*Stream, error) {
	bound, err := d.bind(inputs)
	if err != nil {
		return nil, err
	}
	budget := 0
	e := &evaluator{definition: d, inputs: bound, visiting: make(map[string]bool), budget: &budget}
	e = e.child()
	// Validate every module, including disconnected nodes and unselected branches.
	ids := make([]string, 0, len(d.Graph))
	for id := range d.Graph {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		if _, err := e.evaluate(id); err != nil {
			return nil, err
		}
	}
	output := module{id: "output", inputs: d.Output, evaluator: e}
	output.only("audio", "duration_ms")
	audio := output.signal("audio", "audio")
	duration := output.number("duration_ms", "duration", .001, maxDuration)
	if output.err != nil {
		return nil, output.err
	}
	plan := processingPlan(output.dependencies)
	e.seal()
	return &Stream{evaluator: e, plan: plan, audio: audio,
		total: max(1, int(math.Round(duration*SampleRate/1000)))}, nil
}

// Stream creates a voice with a snapshot of the selected named input values.
func (p *Prepared) Stream(inputs Inputs) (*Stream, error) { return p.definition.Stream(inputs) }

// Render collects a voice into a complete sound for application-owned caching.
func (p *Prepared) Render(inputs Inputs) (*Sound, error) { return p.definition.Render(inputs) }

type processor struct {
	sample       signal
	buffer       [BlockSize]float64
	dependencies []*processor
	run          func(start, count int)
}

func (e *evaluator) child() *evaluator {
	return &evaluator{definition: e.definition, inputs: e.inputs, visiting: e.visiting,
		budget: e.budget, memo: make(map[string]ports), processors: make(map[string][]*processor)}
}

// Only the sample cursor is needed after compilation. Drop validation maps and
// unreachable processors before the voice enters the processing path.
func (e *evaluator) seal() {
	e.definition, e.inputs, e.memo, e.visiting, e.processors, e.budget = nil, nil, nil, nil, nil, nil
}

type audioOutput struct {
	sample       signal
	dependencies []*processor
}

func (e *evaluator) audio(raw json.RawMessage) (*audioOutput, error) {
	m := &module{id: "instance", inputs: map[string]json.RawMessage{"audio": raw}, evaluator: e}
	sample := m.signal("audio", "audio")
	return &audioOutput{sample, m.dependencies}, m.err
}

func processingPlan(outputs []*processor) []*processor {
	var plan []*processor
	seen := make(map[*processor]bool)
	var visit func(*processor)
	visit = func(p *processor) {
		if seen[p] {
			return
		}
		seen[p] = true
		for _, input := range p.dependencies {
			visit(input)
		}
		plan = append(plan, p)
	}
	for _, output := range outputs {
		visit(output)
	}
	return plan
}

func runPlan(plan []*processor, start, count int) {
	for _, p := range plan {
		p.run(start, count)
	}
}

// Stream produces 48000 Hz mono Float32 audio. A voice is owned by one goroutine;
// separate voices can be processed concurrently. Processing uses bounded scratch
// and delay buffers, with no background workers or automatic waveform cache.
type Stream struct {
	evaluator       *evaluator
	plan            []*processor
	audio           signal
	total, position int
	peak            float64
	err             error
	closed          bool
}

// ReadSamples fills caller-owned storage. It returns (n, io.EOF) with the final
// samples, then (0, io.EOF). A zero-length read returns (0, nil). Entries after n
// are untouched. Non-finite output terminates the voice with the valid prefix
// and a sticky error. Successful processing performs no heap allocations.
func (s *Stream) ReadSamples(buffer []float32) (int, error) {
	if len(buffer) == 0 {
		return 0, nil
	}
	if s.err != nil {
		return 0, s.err
	}
	if s.closed || s.position >= s.total {
		return 0, io.EOF
	}
	written := 0
	for written < len(buffer) && s.position < s.total {
		count := min(BlockSize, len(buffer)-written, s.total-s.position)
		runPlan(s.plan, s.position, count)
		for i := 0; i < count; i++ {
			s.evaluator.frame = i
			sample := float32(s.audio(0))
			if math.IsNaN(float64(sample)) || math.IsInf(float64(sample), 0) {
				s.err = fmt.Errorf("output: non-finite sample %d", s.position)
				return written, s.err
			}
			buffer[written] = sample
			s.peak = math.Max(s.peak, math.Abs(float64(sample)))
			written++
			s.position++
		}
	}
	if s.position == s.total {
		return written, io.EOF
	}
	return written, nil
}

// Peak is the absolute peak of the samples delivered so far.
func (s *Stream) Peak() float64 { return s.peak }

// DurationMS is the explicit duration, including authored effect tails.
func (s *Stream) DurationMS() float64 { return float64(s.total) / SampleRate * 1000 }

// Close abandons the voice and releases its references to DSP storage. It is
// idempotent. It must not race with ReadSamples. Garbage collection reclaims it.
func (s *Stream) Close() error {
	s.closed = true
	s.evaluator, s.plan, s.audio = nil, nil, nil
	return nil
}
