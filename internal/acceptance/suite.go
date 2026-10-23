// Package acceptance reads the language-neutral reference suite used by tests.
package acceptance

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
)

type Suite struct {
	FormatVersion int `json:"format_version"`
	Source        struct {
		Repository string `json:"repository"`
		Revision   string `json:"revision"`
		SHA256     string `json:"sha256"`
	} `json:"source"`
	Tolerance struct {
		Float32Absolute float64 `json:"float32_absolute"`
		PCM16Absolute   int     `json:"pcm16_absolute"`
	} `json:"tolerance"`
	Cases []Case `json:"cases"`
}

type Case struct {
	Name       string         `json:"name"`
	Definition string         `json:"definition"`
	Inputs     map[string]any `json:"inputs"`
	Expected   struct {
		SampleRate  int    `json:"sample_rate"`
		Channels    int    `json:"channels"`
		SampleCount int    `json:"sample_count"`
		Samples     string `json:"samples"`
		WAV         string `json:"wav"`
	} `json:"expected"`
}

func Load(directory string) (*Suite, error) {
	data, err := os.ReadFile(filepath.Join(directory, "manifest.json"))
	if err != nil {
		return nil, err
	}
	var suite Suite
	if err := json.Unmarshal(data, &suite); err != nil {
		return nil, err
	}
	if suite.FormatVersion != 1 || len(suite.Cases) == 0 {
		return nil, fmt.Errorf("expected a nonempty version 1 acceptance suite")
	}
	if suite.Source.Repository == "" || suite.Source.Revision == "" || len(suite.Source.SHA256) != 64 {
		return nil, fmt.Errorf("missing acceptance source identity")
	}
	if suite.Tolerance.Float32Absolute < 0 || suite.Tolerance.PCM16Absolute < 0 {
		return nil, fmt.Errorf("invalid acceptance tolerances")
	}
	names := make(map[string]bool)
	for _, c := range suite.Cases {
		if c.Name == "" || names[c.Name] {
			return nil, fmt.Errorf("empty or duplicate acceptance case %q", c.Name)
		}
		names[c.Name] = true
		if c.Expected.SampleRate != 48000 || c.Expected.Channels != 1 || c.Expected.SampleCount < 1 {
			return nil, fmt.Errorf("%s: invalid audio dimensions", c.Name)
		}
		for _, file := range []string{c.Definition, c.Expected.Samples, c.Expected.WAV} {
			if file == "" || !filepath.IsLocal(filepath.FromSlash(file)) {
				return nil, fmt.Errorf("%s: invalid fixture path", c.Name)
			}
		}
	}
	return &suite, nil
}

func (c Case) Read(directory string) (definition []byte, samples []float32, wav []byte, err error) {
	definition, err = os.ReadFile(filepath.Join(directory, filepath.FromSlash(c.Definition)))
	if err != nil {
		return
	}
	data, err := os.ReadFile(filepath.Join(directory, filepath.FromSlash(c.Expected.Samples)))
	if err != nil {
		return
	}
	if len(data) != c.Expected.SampleCount*4 {
		err = fmt.Errorf("%s: reference sample count mismatch", c.Name)
		return
	}
	samples = make([]float32, c.Expected.SampleCount)
	for i := range samples {
		samples[i] = math.Float32frombits(binary.LittleEndian.Uint32(data[i*4:]))
		if math.IsNaN(float64(samples[i])) || math.IsInf(float64(samples[i]), 0) {
			err = fmt.Errorf("%s: non-finite reference sample", c.Name)
			return
		}
	}
	wav, err = os.ReadFile(filepath.Join(directory, filepath.FromSlash(c.Expected.WAV)))
	if err == nil && len(wav) != 44+c.Expected.SampleCount*2 {
		err = fmt.Errorf("%s: reference WAV length mismatch", c.Name)
	}
	return
}
