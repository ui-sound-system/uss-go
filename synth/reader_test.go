package synth

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"testing"
)

func TestFloat32ReaderMatchesSamplesWithArbitraryByteBuffers(t *testing.T) {
	d := streamDefinition(t)
	full, err := d.Render(nil)
	if err != nil {
		t.Fatal(err)
	}
	var want bytes.Buffer
	if err := binary.Write(&want, binary.LittleEndian, full.Samples); err != nil {
		t.Fatal(err)
	}
	for _, size := range []int{1, 2, 3, 4, 7, 511, 1024, 4099} {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			s, err := d.Stream(nil)
			if err != nil {
				t.Fatal(err)
			}
			r := s.Float32LEReader()
			defer r.Close()
			if n, err := r.Read(nil); n != 0 || err != nil || s.position != 0 {
				t.Fatal("empty read advanced synthesis")
			}
			var got []byte
			buffer := make([]byte, size)
			for {
				n, err := r.Read(buffer)
				got = append(got, buffer[:n]...)
				if err == io.EOF {
					break
				}
				if err != nil {
					t.Fatal(err)
				}
			}
			if !bytes.Equal(got, want.Bytes()) {
				t.Fatal("byte stream differs from full render")
			}
			if n, err := r.Read(buffer); n != 0 || err != io.EOF {
				t.Fatal("EOF not retained")
			}
		})
	}
}

func TestFloat32ReaderCloseDiscardsPartialSample(t *testing.T) {
	s, err := streamDefinition(t).Stream(nil)
	if err != nil {
		t.Fatal(err)
	}
	r := s.Float32LEReader()
	var buffer [1]byte
	if n, err := r.Read(buffer[:]); n != 1 || err != nil || s.position != 1 {
		t.Fatal("reader generated more than the requested sample")
	}
	r.Close()
	r.Close()
	if s.evaluator != nil {
		t.Fatal("Close retained DSP state")
	}
	if n, err := r.Read(buffer[:]); n != 0 || err != io.EOF {
		t.Fatal("Close retained pending bytes")
	}
	if n, err := r.Read(nil); n != 0 || err != nil {
		t.Fatal("empty read after Close")
	}
}

func TestFloat32ReaderDeliversErrorPrefixAndStickyError(t *testing.T) {
	d := streamDefinition(t)
	previous := "tone.audio"
	for i := 0; i < 140; i++ {
		id := fmt.Sprintf("double%d", i)
		raw, _ := json.Marshal(map[string]any{"audio_a": map[string]string{"ref": previous}, "audio_b": map[string]string{"ref": previous}, "duration_a_ms": 1000, "duration_b_ms": 1000})
		var inputs map[string]json.RawMessage
		json.Unmarshal(raw, &inputs)
		d.Graph[id] = node{Module: "mix", Inputs: inputs}
		previous = id + ".audio"
	}
	d.Output["audio"], _ = json.Marshal(map[string]string{"ref": previous})
	s, err := d.Stream(nil)
	if err != nil {
		t.Fatal(err)
	}
	r := s.Float32LEReader()
	defer r.Close()
	var buffer [16]byte
	n, failure := r.Read(buffer[:5])
	if n != 4 || failure == nil || failure == io.EOF || binary.LittleEndian.Uint32(buffer[:4]) != math.Float32bits(0) {
		t.Fatalf("invalid error prefix: %d %v", n, failure)
	}
	if n, err := r.Read(buffer[:]); n != 0 || err != failure {
		t.Fatal("synthesis error not retained")
	}
}
