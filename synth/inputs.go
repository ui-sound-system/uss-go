package synth

import (
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

var identifier = regexp.MustCompile(`^[A-Za-z][\w-]*$`)
var notePattern = regexp.MustCompile(`^([A-G])([#b]?)(-?\d)$`)
var noteNames = []string{"C", "C#", "D", "D#", "E", "F", "F#", "G", "G#", "A", "A#", "B"}

func noteIndex(note string) (int, error) {
	match := notePattern.FindStringSubmatch(note)
	if match == nil {
		return 0, fmt.Errorf("expected a note such as C5 or F#4")
	}
	octave, _ := strconv.Atoi(match[3])
	index := (octave+1)*12 + map[string]int{"C": 0, "D": 2, "E": 4, "F": 5, "G": 7, "A": 9, "B": 11}[match[1]]
	if match[2] == "#" {
		index++
	} else if match[2] == "b" {
		index--
	}
	if index < 0 || index > 127 {
		return 0, fmt.Errorf("note outside supported range")
	}
	return index, nil
}
func noteName(index int) string { return noteNames[index%12] + strconv.Itoa(index/12-1) }
func numericInput(kind string, value any) (float64, error) {
	if kind == "note" {
		text, ok := value.(string)
		if !ok {
			return 0, fmt.Errorf("expected note text")
		}
		index, err := noteIndex(text)
		return float64(index), err
	}
	// Accept ordinary Go numeric arguments as well as JSON's float64 values.
	switch value := value.(type) {
	case int:
		return float64(value), nil
	case float32:
		return numericInput(kind, float64(value))
	case float64:
		if !math.IsNaN(value) && !math.IsInf(value, 0) {
			return value, nil
		}
	}
	return 0, fmt.Errorf("expected a finite number")
}
func (d *Definition) bind(overrides Inputs) (map[string]port, error) {
	for name := range overrides {
		if _, exists := d.Inputs[name]; !exists {
			return nil, fmt.Errorf("unknown input %q; resolved-only files have no adjustable inputs", name)
		}
	}
	result := make(map[string]port, len(d.Inputs))
	for name, spec := range d.Inputs {
		if !identifier.MatchString(name) || spec.Label == "" {
			return nil, fmt.Errorf("inputs.%s: invalid name or missing label", name)
		}
		if spec.Locked != nil {
			var flag bool
			if string(spec.Locked) == "null" || json.Unmarshal(spec.Locked, &flag) != nil {
				return nil, fmt.Errorf("inputs.%s.locked: expected true or false", name)
			}
		}
		if spec.Type == "choice" {
			value, err := bindChoice(name, spec, overrides)
			if err != nil {
				return nil, err
			}
			result[name] = port{kind: "text", value: value}
			continue
		}
		if spec.Choices != nil {
			return nil, fmt.Errorf("inputs.%s: only choice inputs use choices", name)
		}
		if spec.Type != "note" && spec.Type != "duration" && spec.Type != "float" {
			return nil, fmt.Errorf("inputs.%s: unsupported type %q", name, spec.Type)
		}
		if spec.Type == "duration" && spec.Unit != "ms" {
			return nil, fmt.Errorf("inputs.%s: duration must use ms", name)
		}
		if spec.Type == "note" && spec.Unit != "" {
			return nil, fmt.Errorf("inputs.%s: notes do not use a unit", name)
		}
		if len(spec.Range) != 2 {
			return nil, fmt.Errorf("inputs.%s: expected a two-value range", name)
		}
		values := make([]float64, 3)
		for i, raw := range []json.RawMessage{spec.Range[0], spec.Range[1], spec.Default} {
			var value any
			if err := json.Unmarshal(raw, &value); err != nil {
				return nil, fmt.Errorf("inputs.%s: %w", name, err)
			}
			numeric, err := numericInput(spec.Type, value)
			if err != nil {
				return nil, fmt.Errorf("inputs.%s: %w", name, err)
			}
			values[i] = numeric
		}
		minimum, maximum, chosen := values[0], values[1], values[2]
		step := 1.0
		if spec.Step != nil {
			step = *spec.Step
		}
		if step < 1e-6 || step > 1e6 || math.IsNaN(step) || math.IsInf(step, 0) || maximum < minimum || minimum < -1e6 || maximum > 1e6 {
			return nil, fmt.Errorf("inputs.%s: invalid range or step", name)
		}
		if spec.Type == "note" && step != 1 {
			return nil, fmt.Errorf("inputs.%s: notes use semitone steps", name)
		}
		if spec.Type == "duration" && (minimum < 0 || maximum > maxDuration) {
			return nil, fmt.Errorf("inputs.%s: duration outside 0–10000 ms", name)
		}
		if chosen < minimum || chosen > maximum || math.Abs((chosen-minimum)/step-math.Round((chosen-minimum)/step)) > 1e-6 {
			return nil, fmt.Errorf("inputs.%s: default outside range or step grid", name)
		}
		if override, ok := overrides[name]; ok {
			var err error
			chosen, err = numericInput(spec.Type, override)
			if err != nil {
				return nil, fmt.Errorf("inputs.%s: %w", name, err)
			}
			if spec.Type != "note" && (chosen < minimum || chosen > maximum) {
				return nil, fmt.Errorf("inputs.%s: value outside %g–%g", name, minimum, maximum)
			}
		}
		index := math.Min(math.Floor((maximum-minimum)/step+1e-8), math.Max(0, math.Floor((chosen-minimum)/step+.5)))
		chosen = minimum + index*step
		// Match JS normalization's 12 significant digits before binding the graph.
		rounded, _ := strconv.ParseFloat(strconv.FormatFloat(chosen, 'g', 12, 64), 64)
		kind := spec.Type
		var value any = rounded
		if kind == "float" {
			kind = "number"
		}
		if kind == "note" {
			value = noteName(int(rounded))
		}
		result[name] = port{kind: kind, value: value}
	}
	return result, nil
}

func bindChoice(name string, spec input, overrides Inputs) (string, error) {
	if len(spec.Choices) < 2 || len(spec.Choices) > 16 || spec.Range != nil || spec.Step != nil || spec.Unit != "" {
		return "", fmt.Errorf("inputs.%s: choices need 2–16 strings and no range, step, or unit", name)
	}
	seen := make(map[string]bool)
	for _, choice := range spec.Choices {
		if strings.TrimSpace(choice) == "" || len(choice) > 32 || seen[choice] {
			return "", fmt.Errorf("inputs.%s: choices must be unique nonempty strings, up to 32 characters", name)
		}
		seen[choice] = true
	}
	var value string
	if err := json.Unmarshal(spec.Default, &value); err != nil || !seen[value] {
		return "", fmt.Errorf("inputs.%s: default must be one of the choices", name)
	}
	if override, ok := overrides[name]; ok {
		text, ok := override.(string)
		if !ok || !seen[text] {
			return "", fmt.Errorf("inputs.%s: value must be one of the choices", name)
		}
		value = text
	}
	return value, nil
}
