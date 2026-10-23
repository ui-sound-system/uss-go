package app

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/ui-sound-system/uss-go/internal/acceptance"
	"github.com/ui-sound-system/uss-go/synth"
)

func TestEmbeddedAcceptanceDefinition(t *testing.T) {
	const directory = "../../../../acceptance"
	suite, err := acceptance.Load(directory)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range suite.Cases {
		if !strings.HasPrefix(c.Name, "two-note/") || len(c.Inputs) != 0 {
			continue
		}
		data, _, _, err := c.Read(directory)
		if err != nil {
			t.Fatal(err)
		}
		var embedded, reference any
		if err := json.Unmarshal(defaultSound, &embedded); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(data, &reference); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(embedded, reference) {
			t.Fatal("embedded example differs from the accepted Two note export")
		}
		return
	}
	t.Fatal("missing default Two note acceptance case")
}

func TestRuntimeInputs(t *testing.T) {
	definition, err := synth.Parse(defaultSound)
	if err != nil {
		t.Fatal(err)
	}
	base, err := definition.Render(nil)
	if err != nil {
		t.Fatal(err)
	}
	pitched, err := definition.Render(synth.Inputs{"note": "G5"})
	if err != nil {
		t.Fatal(err)
	}
	if base.DurationMS() != 339 || pitched.DurationMS() != base.DurationMS() {
		t.Fatal("pitch override changed duration")
	}
	if bytes.Equal(base.WAV(), pitched.WAV()) {
		t.Fatal("pitch override had no effect")
	}
	for _, inputs := range []synth.Inputs{{"unknown": 1}, {"note": 5}, {"spacing": -1}, {"level": "loud"}} {
		if _, err := definition.Render(inputs); err == nil {
			t.Fatalf("accepted invalid inputs: %v", inputs)
		}
	}
}

func TestRejectInvalidGraph(t *testing.T) {
	for _, mutate := range []func(map[string]any){
		func(d map[string]any) { d["version"] = 2 },
		func(d map[string]any) { d["surprise"] = true },
		func(d map[string]any) {
			d["graph"].(map[string]any)["first_pitch"].(map[string]any)["module"] = "unknown"
		},
		func(d map[string]any) {
			d["graph"].(map[string]any)["first_pitch"].(map[string]any)["inputs"] = map[string]any{"note": map[string]any{"ref": "first_pitch.hz"}}
		},
		func(d map[string]any) {
			d["output"].(map[string]any)["audio"] = map[string]any{"ref": "first_envelope.control"}
		},
		func(d map[string]any) { d["output"].(map[string]any)["duration_ms"] = 10001 },
	} {
		var document map[string]any
		if err := json.Unmarshal(defaultSound, &document); err != nil {
			t.Fatal(err)
		}
		mutate(document)
		data, _ := json.Marshal(document)
		definition, err := synth.Parse(data)
		if err == nil {
			_, err = definition.Render(nil)
		}
		if err == nil {
			t.Fatal("accepted invalid graph")
		}
	}
}

func TestRenderCommand(t *testing.T) {
	file := filepath.Join(t.TempDir(), "sound.wav")
	var output, diagnostics bytes.Buffer
	err := Run([]string{"--set", "note=G5", "--wav", file}, &output, &diagnostics)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) != 44+16272*2 || binary.LittleEndian.Uint32(data[24:]) != 48000 {
		t.Fatal("incorrect WAV")
	}
	if !strings.Contains(output.String(), "339 ms") {
		t.Fatal(output.String())
	}
	if err := Run([]string{"--set", "unknown=1", "--wav", file}, &output, &diagnostics); err == nil {
		t.Fatal("accepted unknown input")
	}
}
