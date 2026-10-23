package synth

import "math"

func noise(m *module) ports {
	seed := m.number("seed", "number", 0, 4294967295)
	if seed != math.Trunc(seed) {
		m.fail("seed", "expected an integer seed")
	}
	if m.err != nil {
		return nil
	}
	offset := uint32(seed) * uint32(0x9e3779b9)
	return ports{"audio": {"audio", signal(func(t float64) float64 {
		if t < 0 || t >= maxDuration/1000 {
			return 0
		}
		value := uint32(math.Round(t*SampleRate)) + offset
		value = (value ^ (value >> 16)) * uint32(0x7feb352d)
		value = (value ^ (value >> 15)) * uint32(0x846ca68b)
		value ^= value >> 16
		return float64(value)/2147483648 - 1
	})}}
}
