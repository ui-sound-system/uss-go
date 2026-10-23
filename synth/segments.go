package synth

type envelopeSegment struct {
	start, end, level float64
	shape             signal
}

func envelopeSegmentNode(m *module) ports {
	var previous []envelopeSegment
	if _, exists := m.inputs["previous"]; exists {
		previous, _ = m.read("previous", "envelope").([]envelopeSegment)
	}
	if len(previous) >= 32 {
		m.fail("previous", "at most 32 envelope segments are supported")
	}
	start := 0.0
	if len(previous) > 0 {
		start = previous[len(previous)-1].end
	}
	duration := m.number("duration_ms", "duration", 0, maxDuration)
	level := m.number("level", "number", 0, 1)
	shape := m.curve("curve")
	end := start + duration
	if end > maxDuration {
		m.fail("duration_ms", "total envelope duration exceeds 10000 ms")
	}
	segments := make([]envelopeSegment, len(previous)+1)
	copy(segments, previous)
	segments[len(previous)] = envelopeSegment{start, end, level, shape}
	return ports{"shape": {"envelope", segments}}
}

func envelopeSegments(m *module) ports {
	segments, _ := m.read("segments", "envelope").([]envelopeSegment)
	if m.err != nil {
		return nil
	}
	last := segments[len(segments)-1]
	if last.end <= 0 {
		m.fail("duration_ms", "envelope duration must be greater than 0")
	}
	if last.level != 0 {
		m.fail("segments", "the final envelope segment must reach level 0")
	}
	control := signal(func(t float64) float64 {
		if t < 0 || t >= last.end/1000 {
			return 0
		}
		from := 0.0
		for _, segment := range segments {
			if t < segment.end/1000 {
				position := (t - segment.start/1000) / ((segment.end - segment.start) / 1000)
				return from + (segment.level-from)*segment.shape(position)
			}
			from = segment.level
		}
		return 0
	})
	return ports{"control": {"control", control}, "duration_ms": {"duration", last.end}}
}
