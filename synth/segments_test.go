package synth

import (
	"encoding/json"
	"os"
	"testing"
)

func TestSegmentsRejectInvalidDefinitions(t *testing.T) {
	data, err := os.ReadFile("testdata/one-note.resolved.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct{ node, port, value string }{
		{"rise", "duration_ms", "-1"}, {"rise", "duration_ms", "10000"},
		{"rise", "level", "2"}, {"rise", "level", "null"},
		{"rise", "curve", "17"}, {"rise", "curve", `"unknown"`},
		{"rise", "curve", `{"ref":"envelope.duration_ms"}`},
		{"rise", "previous", "1"}, {"rise", "previous", `{"ref":"tone.audio"}`},
		{"tail", "level", "0.2"}, {"envelope", "segments", "1"},
	} {
		t.Run(c.node+"/"+c.port+"/"+c.value, func(t *testing.T) {
			d, err := Parse(data)
			if err != nil {
				t.Fatal(err)
			}
			d.Graph["rise"] = node{Module: "envelope_segment", Inputs: map[string]json.RawMessage{
				"duration_ms": json.RawMessage("10"), "level": json.RawMessage("1"), "curve": json.RawMessage("-4"),
			}}
			d.Graph["tail"] = node{Module: "envelope_segment", Inputs: map[string]json.RawMessage{
				"previous": json.RawMessage(`{"ref":"rise.shape"}`), "duration_ms": json.RawMessage("20"), "level": json.RawMessage("0"), "curve": json.RawMessage("4"),
			}}
			d.Graph["envelope"] = node{Module: "envelope_segments", Inputs: map[string]json.RawMessage{
				"segments": json.RawMessage(`{"ref":"tail.shape"}`),
			}}
			d.Graph[c.node].Inputs[c.port] = json.RawMessage(c.value)
			if _, err := d.Render(nil); err == nil {
				t.Fatalf("expected %s validation error, got %v", c.node, err)
			}
		})
	}
}
