package synth

import "math"

func trim(m *module) ports {
	audio := m.signal("audio", "audio")
	duration := m.number("duration_ms", "duration", .001, maxDuration)
	keep := m.number("keep_ms", "duration", 0, duration)
	enabled := m.number("enabled", "number", 0, 1)
	if enabled != 0 && enabled != 1 {
		m.fail("enabled", "expected 0 or 1")
	}
	threshold := m.number("threshold_db", "number", 0, 120)
	if m.err != nil {
		return nil
	}
	count := max(1, int(math.Round(duration*SampleRate/1000)))
	protectedCount := min(count, max(1, int(math.Round(keep*SampleRate/1000))))
	if enabled == 0 || protectedCount == count {
		return ports{"audio": {"audio", audio}, "duration_ms": {"duration", duration}}
	}
	samples := make([]float32, count)
	peak := 0.0
	for i := range samples {
		samples[i] = float32(audio(float64(i) / SampleRate))
		value := float64(samples[i])
		if math.IsNaN(value) || math.IsInf(value, 0) {
			m.fail("audio", "non-finite sample")
			return nil
		}
		peak = math.Max(peak, math.Abs(value))
	}
	cutoff := peak * math.Pow(10, -threshold/20)
	last := count - 1
	for last >= 0 && (samples[last] == 0 || math.Abs(float64(samples[last])) < cutoff) {
		last--
	}
	candidate := 0
	if last >= 0 {
		candidate = last + 1 + SampleRate/20
	}
	end := min(count, max(protectedCount, candidate))
	if end < count {
		start := max(protectedCount, end-SampleRate/100)
		for i := start; i < end; i++ {
			samples[i] = float32(float64(samples[i]) * (float64(end-1-i) / float64(max(1, end-1-start))))
		}
	}
	output := signal(func(t float64) float64 {
		if t < 0 || t >= float64(end)/SampleRate {
			return 0
		}
		index := int(math.Round(t * SampleRate))
		if index >= end {
			return 0
		}
		return float64(samples[index])
	})
	return ports{"audio": {"audio", output}, "duration_ms": {"duration", float64(end) / SampleRate * 1000}}
}
