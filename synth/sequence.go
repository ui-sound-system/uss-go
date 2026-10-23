package synth

func numberLerp(m *module) ports {
	from := m.number("from", "number", -1e6, 1e6)
	to := m.number("to", "number", -1e6, 1e6)
	position := m.number("position", "number", 0, 1)
	return ports{"value": {"number", from + (to-from)*position}}
}

func durationLerp(m *module) ports {
	from := m.number("from_ms", "duration", 0, maxDuration)
	to := m.number("to_ms", "duration", 0, maxDuration)
	position := m.number("position", "number", 0, 1)
	multiplier := m.number("multiplier", "number", 0, 32)
	offset := m.number("offset_ms", "duration", 0, maxDuration)
	duration := offset + (from+(to-from)*position)*multiplier
	if duration > maxDuration {
		m.fail("duration_ms", "scheduled time exceeds 10000 ms")
	}
	return ports{"duration_ms": {"duration", duration}}
}
