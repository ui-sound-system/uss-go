package synth

import (
	"encoding/json"
	"testing"
)

func TestTrimRejectsInvalidControls(t *testing.T) {
	for _, c := range []struct{ port, value string }{
		{"threshold_db", "-1"}, {"threshold_db", "121"}, {"threshold_db", "null"},
		{"enabled", "0.5"}, {"enabled", "2"}, {"keep_ms", "200"},
		{"duration_ms", "0"}, {"audio", "1"},
	} {
		t.Run(c.port+"/"+c.value, func(t *testing.T) {
			data := []byte(`{"version":1,"name":"Trim","render":{"sample_rate":48000,"channels":1},
				"graph":{"source":{"module":"noise","inputs":{"seed":1}},
				"trim":{"module":"trim","inputs":{"audio":{"ref":"source.audio"},"duration_ms":100,
				"keep_ms":10,"enabled":1,"threshold_db":60}}},
				"output":{"audio":{"ref":"trim.audio"},"duration_ms":{"ref":"trim.duration_ms"}}}`)
			d, err := Parse(data)
			if err != nil {
				t.Fatal(err)
			}
			d.Graph["trim"].Inputs[c.port] = json.RawMessage(c.value)
			if _, err := d.Render(nil); err == nil {
				t.Fatal("expected trim validation error")
			}
		})
	}
}
