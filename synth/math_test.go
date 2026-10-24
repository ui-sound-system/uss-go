package synth

import (
	"encoding/json"
	"math"
	"testing"
)

func TestArithmetic(t *testing.T) {
	for _, c := range []struct {
		name, a, b, kind string
		want             float64
	}{
		{"number_add", "0.1", "0.2", "number", .1 + .2},
		{"number_subtract", "3", "5", "number", -2},
		{"number_multiply", "0.3", "100", "number", 30},
		{"number_divide", "3", "2", "number", 1.5},
		{"duration_add", "100", "20", "duration", 120},
		{"duration_subtract", "20", "50", "duration", -30},
		{"duration_multiply", "-30", "-1", "duration", 30},
		{"duration_divide", "100", "2", "duration", 50},
		{"duration_ratio", "100", "200", "number", .5},
	} {
		t.Run(c.name, func(t *testing.T) {
			m := module{id: "math", evaluator: &evaluator{}, inputs: map[string]json.RawMessage{
				"a": json.RawMessage(c.a), "b": json.RawMessage(c.b),
			}}
			result := modules[c.name].build(&m)["value"]
			if m.err != nil || result.kind != c.kind || math.Abs(result.value.(float64)-c.want) > 1e-12 {
				t.Fatalf("got %v, %v; want %s %g", result, m.err, c.kind, c.want)
			}
		})
	}
}

func TestArithmeticValidation(t *testing.T) {
	for _, c := range []struct{ name, a, b string }{
		{"number_divide", "1", "0"}, {"duration_divide", "1", "-0"},
		{"duration_ratio", "0", "0"}, {"number_multiply", "1e308", "100"},
		{"duration_add", "1e308", "1e308"}, {"number_add", `"text"`, "2"},
		{"duration_add", `{"ref":"inputs.factor"}`, "2"},
		{"number_add", `{"ref":"inputs.length"}`, "2"},
	} {
		t.Run(c.name+"/"+c.a+"/"+c.b, func(t *testing.T) {
			m := module{id: "math", evaluator: &evaluator{inputs: map[string]port{
				"factor": {"number", .3}, "length": {"duration", 100.0},
			}}, inputs: map[string]json.RawMessage{"a": json.RawMessage(c.a), "b": json.RawMessage(c.b)}}
			modules[c.name].build(&m)
			if m.err == nil {
				t.Fatal("expected invalid arithmetic to fail")
			}
		})
	}
}
