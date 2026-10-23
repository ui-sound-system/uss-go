// Package synth renders USS v1 playback JSON. It has no audio-device dependency.
package synth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"sort"
	"strings"
)

// SampleRate is the USS v1 render rate in Hz. Output is mono.
const SampleRate = 48000
const maxDuration = 10000

// Inputs overrides named playback controls. Nil uses the exported defaults.
type Inputs map[string]any
type signal func(float64) float64
type port struct {
	kind  string
	value any
}
type ports map[string]port
type node struct {
	Module string                     `json:"module"`
	Inputs map[string]json.RawMessage `json:"inputs"`
}
type input struct {
	Label   string            `json:"label"`
	Type    string            `json:"type"`
	Choices []string          `json:"choices,omitempty"`
	Unit    string            `json:"unit,omitempty"`
	Default json.RawMessage   `json:"default"`
	Range   []json.RawMessage `json:"range"`
	Step    *float64          `json:"step,omitempty"`
	Locked  json.RawMessage   `json:"locked,omitempty"`
}

// Definition is parsed USS v1 playback JSON. Render does not change its inputs.
type Definition struct {
	Version int    `json:"version"`
	Name    string `json:"name"`
	Group   string `json:"group,omitempty"`
	Format  struct {
		SampleRate int `json:"sample_rate"`
		Channels   int `json:"channels"`
	} `json:"render"`
	Inputs map[string]input           `json:"inputs,omitempty"`
	Graph  map[string]node            `json:"graph"`
	Output map[string]json.RawMessage `json:"output"`
}

// Sound holds rendered mono Float32 samples and their absolute peak level.
type Sound struct {
	Name    string
	Samples []float32
	Peak    float64
}

// DurationMS returns the sample duration in milliseconds.
func (s *Sound) DurationMS() float64 { return float64(len(s.Samples)) / SampleRate * 1000 }

func decode(data []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return fmt.Errorf("expected one JSON document")
	}
	return nil
}

// Parse reads playback JSON; Render validates the inputs and processing graph.
func Parse(data []byte) (*Definition, error) {
	if len(data) > 1024*1024 {
		return nil, fmt.Errorf("instructions exceed 1 MB")
	}
	var d Definition
	if err := decode(data, &d); err != nil {
		return nil, fmt.Errorf("playback JSON: %w", err)
	}
	if d.Version != 1 || strings.TrimSpace(d.Name) == "" {
		return nil, fmt.Errorf("expected named USS version 1 instructions")
	}
	if d.Format.SampleRate != SampleRate || d.Format.Channels != 1 {
		return nil, fmt.Errorf("render: expected 48000 Hz mono")
	}
	if len(d.Graph) < 1 || len(d.Graph) > 256 {
		return nil, fmt.Errorf("graph: expected 1–256 modules")
	}
	if len(d.Inputs) > 24 {
		return nil, fmt.Errorf("inputs: expected at most 24 inputs")
	}
	for id := range d.Graph {
		if !identifier.MatchString(id) || id == "inputs" {
			return nil, fmt.Errorf("graph: invalid module name %q", id)
		}
	}
	return &d, nil
}

// Render validates and renders the definition with optional named overrides.
func (d *Definition) Render(overrides Inputs) (*Sound, error) {
	inputs, err := d.bind(overrides)
	if err != nil {
		return nil, err
	}
	e := evaluator{definition: d, inputs: inputs, memo: make(map[string]ports), visiting: make(map[string]bool)}
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
	output := module{id: "output", inputs: d.Output, evaluator: &e}
	output.only("audio", "duration_ms")
	audio := output.signal("audio", "audio")
	duration := output.number("duration_ms", "duration", .001, maxDuration)
	if output.err != nil {
		return nil, output.err
	}
	sound := &Sound{Name: d.Name, Samples: make([]float32, max(1, int(math.Round(duration*SampleRate/1000))))}
	for i := range sound.Samples {
		sample := float32(audio(float64(i) / SampleRate))
		if math.IsNaN(float64(sample)) || math.IsInf(float64(sample), 0) {
			return nil, fmt.Errorf("output: non-finite sample %d", i)
		}
		sound.Samples[i] = sample
		sound.Peak = math.Max(sound.Peak, math.Abs(float64(sample)))
	}
	return sound, nil
}

type evaluator struct {
	definition *Definition
	inputs     map[string]port
	memo       map[string]ports
	visiting   map[string]bool
}

func (e *evaluator) evaluate(id string) (ports, error) {
	if result, ok := e.memo[id]; ok {
		return result, nil
	}
	if e.visiting[id] {
		return nil, fmt.Errorf("%s: cyclic connection", id)
	}
	n, ok := e.definition.Graph[id]
	if !ok {
		return nil, fmt.Errorf("unknown module %q", id)
	}
	spec, ok := modules[n.Module]
	if !ok {
		return nil, fmt.Errorf("%s: unsupported module %q", id, n.Module)
	}
	e.visiting[id] = true
	defer delete(e.visiting, id)
	m := module{id: id, inputs: n.Inputs, evaluator: e}
	m.only(spec.inputs...)
	result := spec.build(&m)
	if m.err != nil {
		return nil, m.err
	}
	e.memo[id] = result
	return result, nil
}
func (e *evaluator) read(raw json.RawMessage, expected string) (any, error) {
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, err
	}
	if object, ok := value.(map[string]any); ok {
		ref, ok := object["ref"].(string)
		if !ok || len(object) != 1 {
			return nil, fmt.Errorf("expected a reference object")
		}
		parts := strings.Split(ref, ".")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid reference %q", ref)
		}
		var p port
		var found bool
		if parts[0] == "inputs" {
			p, found = e.inputs[parts[1]]
		} else {
			outputs, err := e.evaluate(parts[0])
			if err != nil {
				return nil, err
			}
			p, found = outputs[parts[1]]
		}
		if !found {
			return nil, fmt.Errorf("unknown reference %q", ref)
		}
		if p.kind != expected && !(expected == "curve" && (p.kind == "number" || p.kind == "text")) {
			return nil, fmt.Errorf("%s: expected %s, got %s", ref, expected, p.kind)
		}
		return p.value, nil
	}
	if expected == "audio" || expected == "control" || expected == "phase" || expected == "envelope" {
		return nil, fmt.Errorf("expected %s reference", expected)
	}
	return value, nil
}

// Module accessors retain the first validation error; a failed module is never sampled.
type module struct {
	id        string
	inputs    map[string]json.RawMessage
	evaluator *evaluator
	err       error
}

func (m *module) fail(name, message string) {
	if m.err == nil {
		m.err = fmt.Errorf("%s.%s: %s", m.id, name, message)
	}
}
func (m *module) only(names ...string) {
	for key := range m.inputs {
		found := false
		for _, name := range names {
			if key == name {
				found = true
				break
			}
		}
		if !found {
			m.fail(key, "unknown input")
		}
	}
}
func (m *module) read(name, kind string) any {
	if m.err != nil {
		return nil
	}
	value, err := m.evaluator.read(m.inputs[name], kind)
	if err != nil {
		m.fail(name, err.Error())
	}
	return value
}
func (m *module) number(name, kind string, minimum, maximum float64) float64 {
	value, ok := m.read(name, kind).(float64)
	if !ok || math.IsNaN(value) || math.IsInf(value, 0) || value < minimum || value > maximum {
		m.fail(name, fmt.Sprintf("expected %s within %g–%g", kind, minimum, maximum))
		return 0
	}
	return value
}
func (m *module) text(name, kind string) string {
	value, ok := m.read(name, kind).(string)
	if !ok {
		m.fail(name, "expected "+kind)
	}
	return value
}
func (m *module) signal(name, kind string) signal {
	value, ok := m.read(name, kind).(signal)
	if !ok {
		m.fail(name, "expected "+kind)
		return func(float64) float64 { return 0 }
	}
	return value
}
