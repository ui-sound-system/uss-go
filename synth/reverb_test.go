package synth

import (
	"encoding/json"
	"testing"
)

func TestReverbRejectsInvalidControlsAndTail(t *testing.T) {
	for _, c := range []struct{ port, value string }{
		{"pre_delay_ms", "-1"}, {"pre_delay_ms", "251"}, {"room_size", "2"},
		{"decay_ms", "0"}, {"decay_ms", "4001"}, {"damping", "-1"}, {"wet", "2"},
		{"duration_ms", "9900"}, {"audio", "1"}, {"wet", `{"ref":"room.duration_ms"}`},
	} {
		t.Run(c.port+"/"+c.value, func(t *testing.T) {
			data := []byte(`{"version":1,"name":"Room","render":{"sample_rate":48000,"channels":1},
				"graph":{"source":{"module":"noise","inputs":{"seed":1}},
				"room":{"module":"reverb","inputs":{"audio":{"ref":"source.audio"},"duration_ms":10,"pre_delay_ms":10,
				"room_size":0.5,"decay_ms":100,"damping":0.4,"wet":0.5}}},
				"output":{"audio":{"ref":"room.audio"},"duration_ms":{"ref":"room.duration_ms"}}}`)
			d, err := Parse(data)
			if err != nil {
				t.Fatal(err)
			}
			d.Graph["room"].Inputs[c.port] = json.RawMessage(c.value)
			if _, err := d.Render(nil); err == nil {
				t.Fatal("expected reverb validation error")
			}
		})
	}
}
