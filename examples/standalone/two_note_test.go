package standalone

import (
	"encoding/json"
	"math"
	"strings"
	"testing"

	"github.com/ui-sound-system/uss-go/internal/acceptance"
)

func TestTwoNoteAcceptance(t *testing.T) {
	const directory = "../../acceptance"
	suite, err := acceptance.Load(directory)
	if err != nil {
		t.Fatal(err)
	}

	checked := 0
	for _, c := range suite.Cases {
		if !strings.HasPrefix(c.Name, "two-note/") {
			continue
		}
		checked++
		t.Run(c.Name, func(t *testing.T) {
			data, reference, _, err := c.Read(directory)
			if err != nil {
				t.Fatal(err)
			}
			var definition struct {
				Inputs map[string]struct{ Default any }
			}
			if err := json.Unmarshal(data, &definition); err != nil {
				t.Fatal(err)
			}
			values := map[string]any{}
			for name, spec := range definition.Inputs {
				values[name] = spec.Default
			}
			for name, value := range c.Inputs {
				values[name] = value
			}
			p := TwoNoteInputs{
				Note: values["note"].(string), Interval: values["interval"].(float64),
				SpacingMS: values["spacing"].(float64), FirstDecayMS: values["first_decay"].(float64),
				SecondDecayMS: values["second_decay"].(float64), Level: values["level"].(float64),
			}
			samples, err := TwoNote(p)
			if err != nil {
				t.Fatal(err)
			}
			if len(samples) != len(reference) {
				t.Fatalf("%d samples, want %d", len(samples), len(reference))
			}
			maximum := 0.0
			for i, sample := range samples {
				maximum = math.Max(maximum, math.Abs(float64(sample)-float64(reference[i])))
			}
			if math.IsNaN(maximum) || math.IsInf(maximum, 0) || maximum > suite.Tolerance.Float32Absolute {
				t.Fatalf("maximum Float32 difference: %g", maximum)
			}
			t.Logf("%d samples; max Float32 difference %g", len(samples), maximum)
		})
	}
	if checked < 5 {
		t.Fatalf("expected at least five two-note cases, got %d", checked)
	}
}

func TestTwoNoteDefaultsAndOverrides(t *testing.T) {
	p := DefaultTwoNoteInputs()
	samples, err := TwoNote(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(samples) != 16272 {
		t.Fatalf("default duration: %d samples", len(samples))
	}
	p.Note = "G5"
	pitched, err := TwoNote(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(pitched) != len(samples) || pitched[120] == samples[120] {
		t.Fatal("pitch override did not preserve duration and change the sound")
	}
	if p.Note != "G5" || p.Level != .18 {
		t.Fatal("render mutated the caller's inputs")
	}
}

func TestTwoNoteRejectsInvalidInputs(t *testing.T) {
	for name, change := range map[string]func(*TwoNoteInputs){
		"note":         func(p *TwoNoteInputs) { p.Note = "hello" },
		"note range":   func(p *TwoNoteInputs) { p.Note = "G#9" },
		"interval":     func(p *TwoNoteInputs) { p.Interval = 13 },
		"spacing":      func(p *TwoNoteInputs) { p.SpacingMS = 79 },
		"first decay":  func(p *TwoNoteInputs) { p.FirstDecayMS = 0 },
		"second decay": func(p *TwoNoteInputs) { p.SecondDecayMS = 301 },
		"level":        func(p *TwoNoteInputs) { p.Level = math.NaN() },
		"infinity":     func(p *TwoNoteInputs) { p.Level = math.Inf(1) },
	} {
		t.Run(name, func(t *testing.T) {
			p := DefaultTwoNoteInputs()
			change(&p)
			if _, err := TwoNote(p); err == nil {
				t.Fatal("accepted invalid input")
			}
		})
	}
}
