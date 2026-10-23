package synth

import (
	"bytes"
	"encoding/binary"
	"math"
	"testing"

	"github.com/ui-sound-system/uss-go/internal/acceptance"
)

func TestAcceptance(t *testing.T) {
	const directory = "../acceptance"
	suite, err := acceptance.Load(directory)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("generator revision %s; source %s", suite.Source.Revision, suite.Source.SHA256)

	for _, c := range suite.Cases {
		t.Run(c.Name, func(t *testing.T) {
			data, samples, referenceWAV, err := c.Read(directory)
			if err != nil {
				t.Fatal(err)
			}
			definition, err := Parse(data)
			if err != nil {
				t.Fatal(err)
			}
			sound, err := definition.Render(Inputs(c.Inputs))
			if err != nil {
				t.Fatal(err)
			}
			if definition.Format.SampleRate != c.Expected.SampleRate || definition.Format.Channels != c.Expected.Channels {
				t.Fatal("audio format mismatch")
			}
			if len(sound.Samples) != len(samples) {
				t.Fatalf("sample count: %d, want %d", len(sound.Samples), len(samples))
			}
			maximum := 0.0
			for i, sample := range sound.Samples {
				maximum = math.Max(maximum, math.Abs(float64(sample)-float64(samples[i])))
			}
			if math.IsNaN(maximum) || math.IsInf(maximum, 0) || maximum > suite.Tolerance.Float32Absolute {
				t.Fatalf("maximum Float32 difference: %g", maximum)
			}
			wav := sound.WAV()
			if len(wav) != len(referenceWAV) || !bytes.Equal(wav[:44], referenceWAV[:44]) {
				t.Fatal("WAV header mismatch")
			}
			maxPCM := 0
			for i := 44; i < len(wav); i += 2 {
				difference := int(int16(binary.LittleEndian.Uint16(wav[i:]))) - int(int16(binary.LittleEndian.Uint16(referenceWAV[i:])))
				maxPCM = max(maxPCM, abs(difference))
			}
			if maxPCM > suite.Tolerance.PCM16Absolute {
				t.Fatalf("maximum PCM difference: %d", maxPCM)
			}
			t.Logf("%d samples; max Float32 difference %g; max PCM difference %d", len(sound.Samples), maximum, maxPCM)
		})
	}
}
func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func TestPCMNegativeHalfTie(t *testing.T) {
	sound := Sound{Samples: []float32{-1.0 / 65536, -1, 1}}
	data := sound.WAV()
	for i, want := range []int16{0, -32768, 32767} {
		if got := int16(binary.LittleEndian.Uint16(data[44+2*i:])); got != want {
			t.Fatalf("sample %d: got %d, want %d", i, got, want)
		}
	}
}
