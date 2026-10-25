package synth

import "math"

func place(m *module) ports {
	start := m.number("start_ms", "duration", 0, maxDuration)
	duration := m.number("duration_ms", "duration", .001, maxDuration)
	end := start + duration
	if end > maxDuration {
		m.fail("end_ms", "scheduled sound exceeds 10000 ms")
	}
	if m.err != nil {
		return nil
	}
	child := m.evaluator.child()
	audio, err := child.audio(m.inputs["audio"])
	if err != nil {
		m.fail("audio", err.Error())
		return nil
	}
	plan := processingPlan(audio.dependencies)
	child.seal()
	onset := int(math.Round(start * SampleRate / 1000))
	limit := max(onset+1, int(math.Round(end*SampleRate/1000)))
	m.process = func(first, count int, output []float64) {
		clear(output)
		begin, finish := max(first, onset), min(first+count, limit)
		if begin >= finish {
			return
		}
		runPlan(plan, begin-onset, finish-begin)
		for i := 0; i < finish-begin; i++ {
			child.frame = i
			output[begin-first+i] = audio.sample(0)
		}
	}
	return ports{"audio": {"audio", signal(func(float64) float64 { return 0 })}, "end_ms": {"duration", end}}
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
