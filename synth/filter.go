package synth

import "math"

func lowpass(m *module) ports {
	audio := m.signal("audio", "audio")
	envelope := m.signal("envelope", "control")
	cutoff := m.number("cutoff_hz", "number", 20, 20000)
	resonance := m.number("resonance", "number", .1, 10)
	amount := m.number("amount", "number", 0, 4)
	// A graph may sample the same filter through several references or placements.
	// Generate each local sample once and retain it for later reads.
	samples := []float64{}
	var x1, x2, y1, y2 float64
	filtered := signal(func(t float64) float64 {
		if t < 0 || t >= maxDuration/1000 {
			return 0
		}
		index := int(math.Round(t * SampleRate))
		for len(samples) <= index {
			time := float64(len(samples)) / SampleRate
			frequency := math.Min(20000, math.Max(20, cutoff*math.Pow(2, amount*envelope(time))))
			// RBJ low-pass biquad, with Q expressed as a linear ratio.
			angle := 2 * math.Pi * frequency / SampleRate
			cosine := math.Cos(angle)
			alpha := math.Sin(angle) / (2 * resonance)
			a0 := 1 + alpha
			b0 := (1 - cosine) / 2 / a0
			b1 := (1 - cosine) / a0
			a1 := -2 * cosine / a0
			a2 := (1 - alpha) / a0
			x := audio(time)
			y := b0*x + b1*x1 + b0*x2 - a1*y1 - a2*y2
			x2, x1, y2, y1 = x1, x, y1, y
			samples = append(samples, y)
		}
		return samples[index]
	})
	return ports{"audio": {"audio", filtered}}
}
