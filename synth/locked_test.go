package synth

import (
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestLockedMetadataIsBooleanAndDoesNotChangePlayback(t *testing.T) {
	data, err := os.ReadFile("testdata/two-note.json")
	if err != nil {
		t.Fatal(err)
	}
	definition, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	baseline, err := definition.Render(nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{"true", "false", `"uniform"`, "1", "null"} {
		t.Run(value, func(t *testing.T) {
			spec := definition.Inputs["note"]
			spec.Locked = json.RawMessage(value)
			definition.Inputs["note"] = spec
			sound, err := definition.Render(nil)
			if value == "true" || value == "false" {
				if err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(sound.Samples, baseline.Samples) {
					t.Fatal("lock metadata changed rendered audio")
				}
			} else if err == nil || !strings.Contains(err.Error(), "locked") {
				t.Fatalf("expected locked validation error, got %v", err)
			}
		})
	}
}
