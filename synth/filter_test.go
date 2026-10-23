package synth

import (
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestLowpassRejectsInvalidInputs(t *testing.T) {
	data, err := os.ReadFile("testdata/delayed-bleep.resolved.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct{ port, value string }{
		{"cutoff_hz", "0"}, {"cutoff_hz", "24000"},
		{"resonance", "0"}, {"resonance", "11"},
		{"amount", "-1"}, {"amount", "5"},
		{"envelope", `{"ref":"first_tone.audio"}`},
	} {
		t.Run(c.port+"/"+c.value, func(t *testing.T) {
			definition, err := Parse(data)
			if err != nil {
				t.Fatal(err)
			}
			definition.Graph["first_filter"].Inputs[c.port] = json.RawMessage(c.value)
			if _, err := definition.Render(nil); err == nil || !strings.Contains(err.Error(), c.port) {
				t.Fatalf("expected %s error, got %v", c.port, err)
			}
		})
	}
}

func TestLowpassStartsFreshForEachRender(t *testing.T) {
	data, err := os.ReadFile("testdata/delayed-bleep.resolved.json")
	if err != nil {
		t.Fatal(err)
	}
	definition, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	first, err := definition.Render(nil)
	if err != nil {
		t.Fatal(err)
	}
	second, err := definition.Render(nil)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first.Samples, second.Samples) {
		t.Fatal("filter retained state between renders")
	}
}
