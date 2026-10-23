package synth

import "math"

func fmOscillator(m *module) ports {
	frequency := m.number("frequency_hz", "frequency", .001, SampleRate/2)
	ratio := m.number("ratio", "number", .125, 16)
	index := m.number("index", "number", 0, 8)
	envelope := signal(func(float64) float64 { return 1 })
	if _, ok := m.inputs["envelope"]; ok {
		envelope = m.signal("envelope", "control")
	}
	modulatorHz := frequency * ratio
	if index > 0 && frequency+modulatorHz*(index+1) > SampleRate/2 {
		m.fail("bandwidth_hz", "dominant upper sidebands exceed 24000 Hz")
	}
	audio := signal(func(t float64) float64 {
		depth := index * math.Max(0, math.Min(1, envelope(t)))
		return math.Sin(2*math.Pi*(frequency*t) + depth*math.Sin(2*math.Pi*(modulatorHz*t)))
	})
	return ports{"audio": {"audio", audio}}
}
