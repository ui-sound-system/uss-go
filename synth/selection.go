package synth

import "math"

func selection(kind string) func(*module) ports {
	return func(m *module) ports {
		selector := m.text("selector", "text")
		match := m.text("match", "text")
		read := func(name string) any {
			switch kind {
			case "number", "duration":
				return m.number(name, kind, -math.MaxFloat64, math.MaxFloat64)
			case "audio":
				return m.signal(name, kind)
			default:
				value := m.text(name, kind)
				if kind == "note" {
					if _, err := noteIndex(value); err != nil {
						m.fail(name, err.Error())
					}
				}
				return value
			}
		}
		value, fallback := read("if_true"), read("if_false")
		if selector != match {
			value = fallback
		}
		return ports{"value": {kind, value}}
	}
}
