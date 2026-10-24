package synth

import (
	"encoding/json"
	"testing"
)

func TestSelection(t *testing.T) {
	for _, c := range []struct{ kind, first, second string }{
		{"number", "7", "-7"}, {"duration", "100", "200"},
		{"note", `"C4"`, `"G5"`}, {"text", `"sine"`, `"triangle"`},
	} {
		for _, label := range []string{"up", "down"} {
			t.Run(c.kind+"/"+label, func(t *testing.T) {
				m := module{id: "select", evaluator: &evaluator{}, inputs: map[string]json.RawMessage{
					"selector": json.RawMessage(`"` + label + `"`), "match": json.RawMessage(`"up"`),
					"if_true": json.RawMessage(c.first), "if_false": json.RawMessage(c.second),
				}}
				result := modules[c.kind+"_select"].build(&m)["value"]
				var expected any
				raw := c.second
				if label == "up" {
					raw = c.first
				}
				if err := json.Unmarshal([]byte(raw), &expected); err != nil {
					t.Fatal(err)
				}
				if m.err != nil || result.kind != c.kind || result.value != expected {
					t.Fatalf("got %v, %v; want %s %v", result, m.err, c.kind, expected)
				}
			})
		}
	}
}

func TestAudioSelection(t *testing.T) {
	for _, label := range []string{"up", "down"} {
		e := evaluator{memo: map[string]ports{
			"a": {"audio": {"audio", signal(func(float64) float64 { return .1 })}},
			"b": {"audio": {"audio", signal(func(float64) float64 { return .2 })}},
		}}
		m := module{id: "select", evaluator: &e, inputs: map[string]json.RawMessage{
			"selector": json.RawMessage(`"` + label + `"`), "match": json.RawMessage(`"up"`),
			"if_true": json.RawMessage(`{"ref":"a.audio"}`), "if_false": json.RawMessage(`{"ref":"b.audio"}`),
		}}
		result := modules["audio_select"].build(&m)["value"]
		want := .2
		if label == "up" {
			want = .1
		}
		if m.err != nil || result.kind != "audio" || result.value.(signal)(0) != want {
			t.Fatalf("audio: %v, %v", result, m.err)
		}
	}
}

func TestSelectionRejectsInvalidBranchesAndSelectors(t *testing.T) {
	for _, c := range []struct{ kind, key, value string }{
		{"number", "if_true", `"text"`}, {"duration", "if_false", `"text"`},
		{"note", "if_true", `"C99"`}, {"text", "if_false", "3"},
		{"number", "selector", "1"}, {"number", "match", "null"},
		{"number", "if_false", "1e309"}, {"audio", "if_true", "0"},
		{"number", "if_true", `{"ref":"inputs.length"}`},
	} {
		t.Run(c.kind+"/"+c.key+"/"+c.value, func(t *testing.T) {
			m := module{id: "select", evaluator: &evaluator{inputs: map[string]port{"length": {"duration", 100.0}}}, inputs: map[string]json.RawMessage{
				"selector": json.RawMessage(`"down"`), "match": json.RawMessage(`"up"`),
				"if_true": json.RawMessage("1"), "if_false": json.RawMessage("2"),
			}}
			m.inputs[c.key] = json.RawMessage(c.value)
			modules[c.kind+"_select"].build(&m)
			if m.err == nil {
				t.Fatal("expected invalid selection to fail")
			}
		})
	}
}
