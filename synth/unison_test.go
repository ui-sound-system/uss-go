package synth

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestUnisonRejectsInvalidInputs(t *testing.T) {
	data, err := os.ReadFile("testdata/delayed-bleep.resolved.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct{ port, value string }{
		{"voices", "0"}, {"voices", "6"}, {"voices", "1.5"},
		{"spread_cents", "-1"}, {"spread_cents", "1201"},
		{"phase_cycles", "2"}, {"waveform", `"noise"`},
		{"frequency_hz", "23999"},
	} {
		t.Run(c.port+"/"+c.value, func(t *testing.T) {
			definition, err := Parse(data)
			if err != nil {
				t.Fatal(err)
			}
			inputs := definition.Graph["second_tone"].Inputs
			inputs[c.port] = json.RawMessage(c.value)
			if c.port == "frequency_hz" {
				inputs["spread_cents"] = json.RawMessage("50")
			}
			if _, err := definition.Render(nil); err == nil || !strings.Contains(err.Error(), c.port) {
				t.Fatalf("expected %s error, got %v", c.port, err)
			}
		})
	}
}
