package synth

import "math"

// Arithmetic permits signed intermediate values. Consumers validate final bounds.
func arithmetic(op, leftKind, rightKind, outputKind string) func(*module) ports {
	return func(m *module) ports {
		a := m.number("a", leftKind, -math.MaxFloat64, math.MaxFloat64)
		b := m.number("b", rightKind, -math.MaxFloat64, math.MaxFloat64)
		var value float64
		switch op {
		case "add":
			value = a + b
		case "subtract":
			value = a - b
		case "multiply":
			value = a * b
		case "divide", "ratio":
			if b == 0 {
				m.fail("b", "division by zero")
			} else {
				value = a / b
			}
		}
		if math.IsNaN(value) || math.IsInf(value, 0) {
			m.fail("value", "expected a finite arithmetic result")
		}
		return ports{"value": {outputKind, value}}
	}
}
