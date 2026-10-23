package synth

import (
	"encoding/json"
	"math"
	"strings"
	"testing"
)

func timingModule() module {
	return module{id: "timing", evaluator: &evaluator{}, inputs: map[string]json.RawMessage{
		"from_ms": json.RawMessage("100"), "to_ms": json.RawMessage("50"),
		"position": json.RawMessage("0.25"), "multiplier": json.RawMessage("4"), "offset_ms": json.RawMessage("10"),
	}}
}

func TestSequenceTimingAndLevelOperations(t *testing.T) {
	m := timingModule()
	result := durationLerp(&m)
	if m.err != nil || result["duration_ms"].value != 360.0 {
		t.Fatalf("timing: %v, %v", result, m.err)
	}
	m = module{id: "level", evaluator: &evaluator{}, inputs: map[string]json.RawMessage{
		"from": json.RawMessage("0.1"), "to": json.RawMessage("0.3"), "position": json.RawMessage("0.25"),
	}}
	result = numberLerp(&m)
	if m.err != nil || math.Abs(result["value"].value.(float64)-.15) > 1e-12 {
		t.Fatalf("level: %v, %v", result, m.err)
	}
}

func TestSequenceTimingRejectsInvalidOrExcessiveTimes(t *testing.T) {
	for _, c := range []struct{ port, value string }{
		{"from_ms", "-1"}, {"to_ms", "10001"}, {"position", "2"},
		{"multiplier", "33"}, {"offset_ms", "-1"},
		{"offset_ms", "10000"},
	} {
		t.Run(c.port+"/"+c.value, func(t *testing.T) {
			m := timingModule()
			m.inputs[c.port] = json.RawMessage(c.value)
			durationLerp(&m)
			if m.err == nil || !strings.Contains(m.err.Error(), "timing") {
				t.Fatalf("expected timing validation error, got %v", m.err)
			}
		})
	}
}
