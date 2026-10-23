package synth

import "math"

func place(m *module) ports {
	audio := m.signal("audio", "audio")
	start := m.number("start_ms", "duration", 0, maxDuration)
	duration := m.number("duration_ms", "duration", .001, maxDuration)
	end := start + duration
	if end > maxDuration {
		m.fail("end_ms", "scheduled sound exceeds 10000 ms")
	}
	placed := signal(func(t float64) float64 {
		local := t - start/1000
		if local < 0 || local >= duration/1000 {
			return 0
		}
		return audio(local)
	})
	return ports{"audio": {"audio", placed}, "end_ms": {"duration", end}}
}
func mix(m *module) ports {
	a := m.signal("audio_a", "audio")
	b := m.signal("audio_b", "audio")
	durationA := m.number("duration_a_ms", "duration", 0, maxDuration)
	durationB := m.number("duration_b_ms", "duration", 0, maxDuration)
	return ports{
		"audio":       {"audio", signal(func(t float64) float64 { return a(t) + b(t) })},
		"duration_ms": {"duration", math.Max(durationA, durationB)},
	}
}
