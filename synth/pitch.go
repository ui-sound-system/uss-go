package synth

import "math"

func noteFrequency(m *module) ports {
	midi, err := noteIndex(m.text("note", "note"))
	if err != nil {
		m.fail("note", err.Error())
	}
	return ports{"hz": {"frequency", 440 * math.Pow(2, float64(midi-69)/12)}}
}
func transpose(m *module) ports {
	midi, err := noteIndex(m.text("note", "note"))
	if err != nil {
		m.fail("note", err.Error())
	}
	semitones := m.number("semitones", "number", -48, 48)
	if semitones != math.Trunc(semitones) {
		m.fail("semitones", "expected an integer")
	}
	midi += int(semitones)
	if midi < 0 || midi > 127 {
		m.fail("note", "transposed note outside supported range")
		midi = 0
	}
	return ports{"note": {"note", noteName(midi)}}
}
func pitchSweep(m *module) ports {
	frequency := m.number("frequency_hz", "frequency", .001, SampleRate/2)
	semitones := m.number("semitones", "number", -48, 48)
	duration := m.number("duration_ms", "duration", 0, maxDuration) / 1000
	endFrequency := frequency * math.Pow(2, semitones/12)
	if endFrequency < .001 || endFrequency > SampleRate/2 {
		m.fail("frequency_hz", "final frequency outside supported range")
	}
	var phase signal
	if duration == 0 {
		phase = func(t float64) float64 { return endFrequency * t }
	} else if semitones == 0 {
		phase = func(t float64) float64 { return frequency * t }
	} else {
		rate := math.Ln2 * semitones / (12 * duration)
		ramp := func(t float64) float64 { return frequency * math.Expm1(rate*t) / rate }
		endPhase := ramp(duration)
		phase = func(t float64) float64 {
			if t < duration {
				return ramp(t)
			}
			return endPhase + endFrequency*(t-duration)
		}
	}
	return ports{"phase": {"phase", phase}}
}
