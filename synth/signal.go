package synth

import "math"

func oscillator(m *module) ports {
	waveform := m.text("waveform", "text")
	if waveform != "sine" && waveform != "triangle" && waveform != "saw" && waveform != "square" {
		m.fail("waveform", "expected sine, triangle, saw, or square")
	}
	_, hasFrequency := m.inputs["frequency_hz"]
	_, hasPhase := m.inputs["phase"]
	if hasFrequency == hasPhase {
		m.fail("phase", "provide exactly one of phase or frequency_hz")
	}
	var phaseAt signal
	var frequency float64
	if hasPhase {
		phaseAt = m.signal("phase", "phase")
	} else {
		frequency = m.number("frequency_hz", "frequency", .001, SampleRate/2)
		phaseAt = func(t float64) float64 { return frequency * t }
	}
	phase := m.number("phase_cycles", "number", 0, 1)
	if waveform == "sine" {
		return ports{"audio": {"audio", signal(func(t float64) float64 { return math.Sin(2 * math.Pi * (phase + phaseAt(t))) })}}
	}
	if hasPhase {
		m.fail("phase", "non-sine waveforms require frequency_hz for harmonic band-limiting")
	}
	if m.err != nil {
		return nil
	}
	type partial struct{ harmonic, gain float64 }
	partials := []partial{}
	last := int(math.Min(128, math.Ceil(SampleRate/2/frequency)-1))
	for n := 1; n <= last; n++ {
		if waveform != "saw" && n%2 == 0 {
			continue
		}
		harmonic := float64(n)
		var gain float64
		if waveform == "saw" {
			sign := 1.0
			if n%2 == 0 {
				sign = -1
			}
			gain = sign * 2 / (math.Pi * harmonic)
		} else if waveform == "square" {
			gain = 4 / (math.Pi * harmonic)
		} else {
			sign := 1.0
			if (n-1)%4 != 0 {
				sign = -1
			}
			gain = sign * 8 / (math.Pi * math.Pi * harmonic * harmonic)
		}
		partials = append(partials, partial{harmonic, gain})
	}
	audio := signal(func(t float64) float64 {
		angle := 2 * math.Pi * (phase + phaseAt(t))
		value := 0.0
		for _, p := range partials {
			value += p.gain * math.Sin(p.harmonic*angle)
		}
		return value
	})
	return ports{"audio": {"audio", audio}}
}
func envelopeAD(m *module) ports {
	attack := m.number("attack_ms", "duration", 0, maxDuration) / 1000
	decay := m.number("decay_ms", "duration", 0, maxDuration) / 1000
	duration := (attack + decay) * 1000
	if duration <= 0 || duration > maxDuration {
		m.fail("duration_ms", "expected duration greater than zero and at most 10000 ms")
	}
	shape := m.curve("curve")
	control := signal(func(t float64) float64 {
		if t < 0 || t >= attack+decay {
			return 0
		}
		if t < attack {
			return shape(t / attack)
		}
		if decay == 0 {
			return 0
		}
		return 1 - shape((t-attack)/decay)
	})
	return ports{"control": {"control", control}, "duration_ms": {"duration", duration}}
}
func amplifier(m *module) ports {
	audio := m.signal("audio", "audio")
	envelope := m.signal("envelope", "control")
	gain := m.number("gain", "number", 0, 1)
	return ports{"audio": {"audio", signal(func(t float64) float64 { return audio(t) * envelope(t) * gain })}}
}
