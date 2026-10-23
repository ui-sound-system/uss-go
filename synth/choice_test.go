package synth

import (
	"encoding/json"
	"testing"
)

func TestChoiceValidation(t *testing.T) {
	base := input{Label: "Waveform", Type: "choice", Default: json.RawMessage(`"sine"`), Choices: []string{"sine", "triangle", "saw", "square"}}
	if got, err := bindChoice("waveform", base, Inputs{"waveform": "square"}); err != nil || got != "square" {
		t.Fatalf("override: %q, %v", got, err)
	}
	for _, value := range []any{"noise", 1, true, nil} {
		if _, err := bindChoice("waveform", base, Inputs{"waveform": value}); err == nil {
			t.Fatalf("accepted %v", value)
		}
	}
	for _, change := range []func(*input){
		func(i *input) { i.Choices = []string{"sine", "sine"} },
		func(i *input) { i.Choices = []string{"sine"} },
		func(i *input) { i.Choices = []string{"sine", ""} },
		func(i *input) { i.Default = json.RawMessage(`"noise"`) },
		func(i *input) { i.Default = json.RawMessage(`1`) },
		func(i *input) { i.Range = []json.RawMessage{json.RawMessage("0"), json.RawMessage("3")} },
		func(i *input) { i.Unit = "Hz" },
		func(i *input) { step := 1.0; i.Step = &step },
	} {
		spec := base
		change(&spec)
		if _, err := bindChoice("waveform", spec, nil); err == nil {
			t.Fatal("accepted invalid choices")
		}
	}
}
