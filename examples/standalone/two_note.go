// Package standalone demonstrates a sound-specific export with no USS runtime.
// This hand-written example corresponds to docs/examples/two-note.yaml.
package standalone

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
)

// TwoNoteInputs exposes the recipe's six controls; durations are milliseconds.
type TwoNoteInputs struct {
	Note          string
	Interval      float64
	SpacingMS     float64
	FirstDecayMS  float64
	SecondDecayMS float64
	Level         float64
}

func DefaultTwoNoteInputs() TwoNoteInputs {
	return TwoNoteInputs{Note: "C5", Interval: 7, SpacingMS: 140, FirstDecayMS: 95, SecondDecayMS: 190, Level: .18}
}

// TwoNote returns mono Float32 samples at 48000 Hz. Start with the defaults and
// override fields as needed. Ranges and steps match the exported recipe.
func TwoNote(p TwoNoteInputs) ([]float32, error) {
	note := regexp.MustCompile(`^([A-G])([#b]?)(-?\d)$`).FindStringSubmatch(p.Note)
	if note == nil {
		return nil, fmt.Errorf("note: expected C5, F#4, etc.")
	}
	octave, _ := strconv.Atoi(note[3])
	pitch := (octave+1)*12 + map[string]int{"C": 0, "D": 2, "E": 4, "F": 5, "G": 7, "A": 9, "B": 11}[note[1]]
	if note[2] == "#" {
		pitch++
	} else if note[2] == "b" {
		pitch--
	}
	if pitch < 0 || pitch > 127 {
		return nil, fmt.Errorf("note: outside supported range")
	}
	pitch = min(84, max(60, pitch)) // This recipe exposes C4 through C6.

	for _, c := range []struct {
		name            string
		value           *float64
		low, high, step float64
	}{
		{"interval", &p.Interval, 3, 12, 1},
		{"spacing", &p.SpacingMS, 80, 240, 10},
		{"first_decay", &p.FirstDecayMS, 40, 140, 5},
		{"second_decay", &p.SecondDecayMS, 100, 300, 10},
		{"level", &p.Level, .08, .3, .01},
	} {
		if math.IsNaN(*c.value) || math.IsInf(*c.value, 0) || *c.value < c.low || *c.value > c.high {
			return nil, fmt.Errorf("%s: expected %g–%g", c.name, c.low, c.high)
		}
		value := c.low + math.Floor((*c.value-c.low)/c.step+.5)*c.step
		*c.value, _ = strconv.ParseFloat(strconv.FormatFloat(value, 'g', 12, 64), 64)
	}

	// Each voice starts its own oscillator phase and linear AD envelope.
	voice := func(t, hz, attackMS, decayMS float64) float64 {
		if t < 0 || t >= (attackMS+decayMS)/1000 {
			return 0
		}
		attack, decay := attackMS/1000, decayMS/1000
		if t >= attack+decay {
			return 0
		}
		envelope := 1 - (t-attack)/decay
		if t < attack {
			envelope = t / attack
		}
		return math.Sin(2*math.Pi*(hz*t)) * envelope * p.Level
	}
	firstHz := 440 * math.Pow(2, float64(pitch-69)/12)
	secondHz := 440 * math.Pow(2, (float64(pitch-69)+p.Interval)/12)
	durationMS := math.Max(3+p.FirstDecayMS, p.SpacingMS+9+p.SecondDecayMS)
	samples := make([]float32, int(math.Floor(durationMS*48000/1000+.5)))
	for i := range samples {
		t := float64(i) / 48000
		samples[i] = float32(voice(t, firstHz, 3, p.FirstDecayMS) +
			voice(t-p.SpacingMS/1000, secondHz, 9, p.SecondDecayMS))
	}
	return samples, nil
}
