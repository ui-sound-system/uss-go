package synth

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestFMRejectsInvalidInputs(t *testing.T) {
	data, err := os.ReadFile("testdata/fm-tone.resolved.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct{ port, value, message string }{
		{"frequency_hz", "0", "frequency_hz"},
		{"ratio", "0", "ratio"},
		{"index", "-1", "index"},
		{"index", "9", "index"},
		{"frequency_hz", "10000", "bandwidth_hz"},
		{"frequency_hz", `{"ref":"envelope.duration_ms"}`, "expected frequency"},
		{"envelope", `{"ref":"envelope.duration_ms"}`, "expected control"},
		{"envelope", `{"ref":"tone.audio"}`, "cyclic"},
		{"envelope", "1", "control"},
		{"envelope", "null", "control"},
	} {
		t.Run(c.port+"/"+c.value, func(t *testing.T) {
			definition, err := Parse(data)
			if err != nil {
				t.Fatal(err)
			}
			inputs := definition.Graph["tone"].Inputs
			inputs[c.port] = json.RawMessage(c.value)
			if c.message == "bandwidth_hz" {
				inputs["ratio"] = json.RawMessage("4")
				inputs["index"] = json.RawMessage("3")
			}
			if _, err := definition.Render(nil); err == nil || !strings.Contains(err.Error(), c.message) {
				t.Fatalf("expected %s, got %v", c.message, err)
			}
		})
	}
}
