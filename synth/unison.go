package synth

import (
	"encoding/json"
	"math"
)

func unisonOscillator(m *module) ports {
	frequency := m.number("frequency_hz", "frequency", .001, SampleRate/2)
	voices := m.number("voices", "number", 1, 5)
	if voices != math.Trunc(voices) {
		m.fail("voices", "expected an integer voice count")
	}
	spread := m.number("spread_cents", "number", 0, 1200)
	if m.err != nil {
		return nil
	}
	count := int(voices)
	if spread == 0 {
		count = 1
	}
	layers := make([]signal, count)
	for i := range layers {
		cents := 0.0
		if count > 1 {
			cents = (float64(i)/float64(count-1) - .5) * spread
		}
		hz, _ := json.Marshal(frequency * math.Pow(2, cents/1200))
		voice := module{id: m.id, evaluator: m.evaluator, inputs: map[string]json.RawMessage{
			"waveform": m.inputs["waveform"], "frequency_hz": hz, "phase_cycles": m.inputs["phase_cycles"],
		}}
		result := oscillator(&voice)
		if voice.err != nil {
			m.err = voice.err
			return nil
		}
		layers[i] = result["audio"].value.(signal)
	}
	return ports{"audio": {"audio", signal(func(t float64) float64 {
		value := 0.0
		for _, layer := range layers {
			value += layer(t)
		}
		return value / float64(count)
	})}}
}
