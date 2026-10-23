package synth

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestNoiseRejectsInvalidSeeds(t *testing.T) {
	for _, seed := range []string{"-1", "4294967296", "1.5", `"one"`, "null", ""} {
		t.Run(seed, func(t *testing.T) {
			inputs := map[string]json.RawMessage{}
			if seed != "" {
				inputs["seed"] = json.RawMessage(seed)
			}
			data, err := json.Marshal(map[string]any{
				"version": 1, "name": "Invalid noise",
				"render": map[string]int{"sample_rate": SampleRate, "channels": 1},
				"graph":  map[string]any{"noise": map[string]any{"module": "noise", "inputs": inputs}},
				"output": map[string]any{"audio": map[string]string{"ref": "noise.audio"}, "duration_ms": 10},
			})
			if err != nil {
				t.Fatal(err)
			}
			definition, err := Parse(data)
			if err == nil {
				_, err = definition.Render(nil)
			}
			if err == nil || !strings.Contains(err.Error(), "seed") {
				t.Fatalf("expected seed error, got %v", err)
			}
		})
	}
}
