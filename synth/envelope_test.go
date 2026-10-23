package synth

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestAHDRejectsInvalidStagesAndCurves(t *testing.T) {
	data, err := os.ReadFile("testdata/delayed-bleep.resolved.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct{ port, value string }{
		{"attack_ms", "-1"}, {"decay_ms", "-1"},
		{"hold_until_ms", "10"}, {"hold_until_ms", "10000"},
		{"hold_ms", "20"},
		{"attack_curve", `"unknown"`}, {"decay_curve", `"unknown"`},
	} {
		t.Run(c.port+"/"+c.value, func(t *testing.T) {
			definition, err := Parse(data)
			if err != nil {
				t.Fatal(err)
			}
			definition.Graph["first_envelope"].Inputs[c.port] = json.RawMessage(c.value)
			_, err = definition.Render(nil)
			if err == nil || !strings.Contains(err.Error(), "first_envelope") {
				t.Fatalf("expected envelope validation error, got %v", err)
			}
		})
	}
	for _, zero := range []bool{false, true} {
		definition, err := Parse(data)
		if err != nil {
			t.Fatal(err)
		}
		inputs := definition.Graph["first_envelope"].Inputs
		delete(inputs, "hold_until_ms")
		if zero {
			for _, name := range []string{"attack_ms", "hold_ms", "decay_ms"} {
				inputs[name] = json.RawMessage("0")
			}
		}
		if _, err := definition.Render(nil); err == nil {
			t.Fatalf("invalid hold or zero total duration accepted (zero=%v)", zero)
		}
	}
}
