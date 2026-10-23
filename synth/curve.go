package synth

import "math"

func (m *module) curve(name string) signal {
	value := m.read(name, "curve")
	linear := signal(func(x float64) float64 { return x })
	switch value {
	case "linear", float64(0):
		return linear
	case "ease-in":
		return func(x float64) float64 { return x * x }
	case "ease-out":
		return func(x float64) float64 { return 1 - (1-x)*(1-x) }
	}
	curvature, ok := value.(float64)
	if !ok || math.IsNaN(curvature) || math.IsInf(curvature, 0) || curvature < -16 || curvature > 16 {
		m.fail(name, "expected linear, ease-in, ease-out, or numeric curvature from -16 to 16")
		return linear
	}
	denominator := math.Expm1(curvature)
	return func(x float64) float64 { return math.Expm1(curvature*x) / denominator }
}
