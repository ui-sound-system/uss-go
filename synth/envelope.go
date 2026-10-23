package synth

func envelopeAHD(m *module) ports {
	attackMS := m.number("attack_ms", "duration", 0, maxDuration)
	decayMS := m.number("decay_ms", "duration", 0, maxDuration)
	_, hasHold := m.inputs["hold_ms"]
	_, hasUntil := m.inputs["hold_until_ms"]
	if hasHold == hasUntil {
		m.fail("hold_ms", "provide exactly one of hold_ms or hold_until_ms")
	}
	var holdEndMS float64
	if hasHold {
		holdEndMS = attackMS + m.number("hold_ms", "duration", 0, maxDuration)
	} else {
		holdEndMS = m.number("hold_until_ms", "duration", 0, maxDuration)
	}
	if holdEndMS < attackMS {
		m.fail("hold_until_ms", "hold must end at or after the attack")
	}
	duration := holdEndMS + decayMS
	if duration <= 0 || duration > maxDuration {
		m.fail("duration_ms", "expected duration greater than zero and at most 10000 ms")
	}
	attackShape, decayShape := m.curve("attack_curve"), m.curve("decay_curve")
	attack, holdEnd, decay := attackMS/1000, holdEndMS/1000, decayMS/1000
	control := signal(func(t float64) float64 {
		if t < 0 || t >= duration/1000 {
			return 0
		}
		if t < attack {
			return attackShape(t / attack)
		}
		if t < holdEnd {
			return 1
		}
		if decay == 0 {
			return 0
		}
		return 1 - decayShape((t-holdEnd)/decay)
	})
	return ports{"control": {"control", control}, "duration_ms": {"duration", duration}, "hold_end_ms": {"duration", holdEndMS}}
}
