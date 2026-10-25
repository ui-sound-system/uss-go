package synth

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/ui-sound-system/uss-go/internal/acceptance"
)

func streamDefinition(t testing.TB) *Definition {
	t.Helper()
	d, err := Parse([]byte(`{"version":1,"name":"Streaming room","render":{"sample_rate":48000,"channels":1},
 "inputs":{"level":{"label":"Level","type":"float","default":0.2,"range":[0,1],"step":0.1}},
 "graph":{
 "tone":{"module":"oscillator","inputs":{"waveform":"sine","frequency_hz":440,"phase_cycles":0}},
 "env":{"module":"envelope_ad","inputs":{"attack_ms":3,"decay_ms":997,"curve":"linear"}},
 "amp":{"module":"amplifier","inputs":{"audio":{"ref":"tone.audio"},"envelope":{"ref":"env.control"},"gain":{"ref":"inputs.level"}}},
 "filter":{"module":"lowpass","inputs":{"audio":{"ref":"amp.audio"},"envelope":{"ref":"env.control"},"cutoff_hz":1000,"resonance":0.7,"amount":2}},
 "room":{"module":"reverb","inputs":{"audio":{"ref":"filter.audio"},"duration_ms":1000,"pre_delay_ms":10,"room_size":0.5,"decay_ms":100,"damping":0.4,"wet":0.5}},
 "mix":{"module":"mix","inputs":{"audio_a":{"ref":"filter.audio"},"audio_b":{"ref":"room.audio"},"duration_a_ms":1000,"duration_b_ms":{"ref":"room.duration_ms"}}}},
 "output":{"audio":{"ref":"mix.audio"},"duration_ms":{"ref":"mix.duration_ms"}}}`))
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func collect(t testing.TB, s *Stream, sizes []int) []float32 {
	t.Helper()
	var result []float32
	for i := 0; ; i++ {
		buffer := make([]float32, sizes[i%len(sizes)])
		for j := range buffer {
			buffer[j] = 123
		}
		n, err := s.ReadSamples(buffer)
		result = append(result, buffer[:n]...)
		for _, v := range buffer[n:] {
			if v != 123 {
				t.Fatal("changed unused buffer suffix")
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
	}
	if n, err := s.ReadSamples(make([]float32, 1)); n != 0 || err != io.EOF {
		t.Fatalf("EOF: %d %v", n, err)
	}
	return result
}

func TestStreamBlockSizesAndOwnership(t *testing.T) {
	d := streamDefinition(t)
	p, err := d.Prepare()
	if err != nil {
		t.Fatal(err)
	}
	inputs := Inputs{"level": 0.3}
	first, err := p.Stream(inputs)
	if err != nil {
		t.Fatal(err)
	}
	inputs["level"] = 1.0
	second, err := p.Stream(Inputs{"level": 0.3})
	if err != nil {
		t.Fatal(err)
	}
	// Mutation of the original definition cannot affect the prepared snapshot.
	d.Graph["tone"].Inputs["frequency_hz"] = json.RawMessage(`880`)
	want := collect(t, first, []int{256})
	got := collect(t, second, []int{1, 64, 17, 1024, 3})
	if !reflect.DeepEqual(got, want) {
		t.Fatal("block size, voice state, or input snapshot changed output")
	}
	for _, size := range []int{1, 64, 256, 1024} {
		voice, err := p.Stream(Inputs{"level": 0.3})
		if err != nil {
			t.Fatal(err)
		}
		if got := collect(t, voice, []int{size}); !reflect.DeepEqual(got, want) {
			t.Fatalf("block size %d changed output", size)
		}
	}
	if first.Peak() == 0 {
		t.Fatal("peak not accumulated")
	}
	if n, err := first.ReadSamples(nil); n != 0 || err != nil {
		t.Fatal("zero read")
	}
	first.Close()
	first.Close()
	if first.plan != nil || first.evaluator != nil {
		t.Fatal("Close retained processing state")
	}
}

func TestStreamingReferenceSamples(t *testing.T) {
	suite, err := acceptance.Load("../acceptance")
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range suite.Cases {
		data, samples, _, err := c.Read("../acceptance")
		if err != nil {
			t.Fatal(err)
		}
		d, err := Parse(data)
		if err != nil {
			t.Fatal(err)
		}
		t.Run(c.Name, func(t *testing.T) {
			for _, sizes := range [][]int{{1}, {64}, {256}, {1024}, {13, 1, 333, 64}} {
				s, err := d.Stream(Inputs(c.Inputs))
				if err != nil {
					t.Fatal(err)
				}
				got := collect(t, s, sizes)
				if len(got) != len(samples) {
					t.Fatalf("sample count %d != %d", len(got), len(samples))
				}
				for i := range got {
					if math.Abs(float64(got[i])-float64(samples[i])) > suite.Tolerance.Float32Absolute {
						t.Fatalf("sample %d differs", i)
					}
				}
			}
		})
	}
}

func TestStreamNoProcessingAllocations(t *testing.T) {
	s, err := streamDefinition(t).Stream(nil)
	if err != nil {
		t.Fatal(err)
	}
	buffer := make([]float32, 64)
	if got := testing.AllocsPerRun(100, func() {
		if _, err := s.ReadSamples(buffer); err != nil {
			t.Fatal(err)
		}
	}); got != 0 {
		t.Fatalf("allocations/block: %g", got)
	}

}

func TestStreamValidationAndClose(t *testing.T) {
	d := streamDefinition(t)
	if _, err := d.Stream(Inputs{"missing": 1}); err == nil {
		t.Fatal("unknown input accepted")
	}
	d.Graph["unused"] = node{Module: "unknown", Inputs: map[string]json.RawMessage{}}
	if _, err := d.Stream(nil); err == nil {
		t.Fatal("invalid disconnected module accepted")
	}
	delete(d.Graph, "unused")
	s, err := d.Stream(nil)
	if err != nil {
		t.Fatal(err)
	}
	s.Close()
	if n, err := s.ReadSamples(make([]float32, 10)); n != 0 || err != io.EOF {
		t.Fatal("closed read")
	}
	// A cycle through an instance must be detected across evaluator boundaries.
	d.Graph["cycle"] = node{Module: "place", Inputs: map[string]json.RawMessage{"audio": json.RawMessage(`{"ref":"cycle.audio"}`), "start_ms": json.RawMessage(`0`), "duration_ms": json.RawMessage(`10`)}}
	if _, err := d.Stream(nil); err == nil {
		t.Fatal("placement cycle accepted")
	}
}

func BenchmarkStream(b *testing.B) {
	d := streamDefinition(b)
	for _, voices := range []int{1, 16, 100} {
		for _, size := range []int{64, 256} {
			b.Run(fmtName(voices, size), func(b *testing.B) {
				streams := make([]*Stream, voices)
				for i := range streams {
					streams[i], _ = d.Stream(nil)
				}
				buffer := make([]float32, size)
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					for j, s := range streams {
						if s.position >= s.total {
							b.StopTimer()
							streams[j], _ = d.Stream(nil)
							s = streams[j]
							b.StartTimer()
						}
						if _, err := s.ReadSamples(buffer); err != nil && err != io.EOF {
							b.Fatal(err)
						}
					}
				}
				b.ReportMetric(float64(size)/SampleRate*1e9/(float64(b.Elapsed().Nanoseconds())/float64(b.N)), "deadline/headroom")
			})
		}
	}
}
func fmtName(voices, size int) string { return fmt.Sprintf("voices=%d/block=%d", voices, size) }
func BenchmarkVoiceStart(b *testing.B) {
	d := streamDefinition(b)
	p, err := d.Prepare()
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s, err := p.Stream(nil)
		if err != nil {
			b.Fatal(err)
		}
		s.Close()
	}
}
func BenchmarkCatalogue100(b *testing.B) {
	data, err := os.ReadFile("testdata/delayed-bleep.resolved.json")
	if err != nil {
		b.Fatal(err)
	}
	registered := make([]*Prepared, 100)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < 100; j++ {
			d, err := Parse(data)
			if err != nil {
				b.Fatal(err)
			}
			p, err := d.Prepare()
			if err != nil {
				b.Fatal(err)
			}
			registered[j] = p
		}
	}
	runtime.KeepAlive(registered)
}

func TestStreamPreMigrationReference(t *testing.T) {
	const directory = "../acceptance/streaming-baseline"
	suite, err := acceptance.Load(directory)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range suite.Cases {
		t.Run(c.Name, func(t *testing.T) {
			data, want, _, err := c.Read(directory)
			if err != nil {
				t.Fatal(err)
			}
			d, err := Parse(data)
			if err != nil {
				t.Fatal(err)
			}
			for _, size := range []int{1, 64, 256, 1024} {
				voice, err := d.Stream(Inputs(c.Inputs))
				if err != nil {
					t.Fatal(err)
				}
				buffer := make([]float32, size)
				for offset := 0; offset < len(want); {
					n, err := voice.ReadSamples(buffer[:min(size, len(want)-offset)])
					if n == 0 || err != nil && err != io.EOF {
						t.Fatalf("read: %d %v", n, err)
					}
					for i := 0; i < n; i++ {
						if math.Abs(float64(buffer[i])-float64(want[offset+i])) > suite.Tolerance.Float32Absolute {
							t.Fatalf("sample %d differs from pre-migration reference", offset+i)
						}
					}
					offset += n
				}
			}
		})
	}
}

func TestStreamProcessingErrorIsSticky(t *testing.T) {
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
	buffer := make([]float32, 64)
	buffer[1] = 123
	n, err := s.ReadSamples(buffer)
	if n != 1 || err == nil || buffer[0] != 0 || buffer[1] != 123 {
		t.Fatalf("nonfinite prefix: n=%d err=%v", n, err)
	}
	if n, again := s.ReadSamples(buffer); n != 0 || again != err {
		t.Fatal("processing error was not sticky")
	}
}

func TestPreparedConcurrentVoices(t *testing.T) {
	p, err := streamDefinition(t).Prepare()
	if err != nil {
		t.Fatal(err)
	}
	expected, err := p.Render(Inputs{"level": 0.3})
	if err != nil {
		t.Fatal(err)
	}
	type result struct {
		samples []float32
		err     error
	}
	results := make(chan result, 4)
	for i := 0; i < 4; i++ {
		go func() {
			s, err := p.Stream(Inputs{"level": 0.3})
			if err != nil {
				results <- result{err: err}
				return
			}
			defer s.Close()
			samples := make([]float32, s.total)
			_, err = s.ReadSamples(samples)
			if err == io.EOF {
				err = nil
			}
			results <- result{samples, err}
		}()
	}
	for i := 0; i < 4; i++ {
		got := <-results
		if got.err != nil {
			t.Fatal(got.err)
		}
		if !reflect.DeepEqual(got.samples, expected.Samples) {
			t.Fatal("concurrent voice changed output")
		}
	}
}

func BenchmarkFirstBlock(b *testing.B) {
	d := streamDefinition(b)
	buffer := make([]float32, 64)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		s, err := d.Stream(nil)
		if err != nil {
			b.Fatal(err)
		}
		b.StartTimer()
		s.ReadSamples(buffer)
	}
}

// Diagnostic, not a timing assertion: ordinary desktop scheduling has outliers.
// Run with -benchtime=10000x to retain a useful tail sample under heap pressure.
func BenchmarkStreamScheduling(b *testing.B) {
	d := streamDefinition(b)
	streams := make([]*Stream, 16)
	for i := range streams {
		streams[i], _ = d.Stream(nil)
	}
	stop, stopped := make(chan struct{}), make(chan struct{})
	go func() {
		defer close(stopped)
		ticker := time.NewTicker(time.Millisecond)
		defer ticker.Stop()
		retained := make([][]byte, 16)
		index := 0
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				data := make([]byte, 64*1024)
				for i := 0; i < len(data); i += 4096 {
					data[i] = byte(index)
				}
				retained[index%len(retained)] = data
				index++
				runtime.KeepAlive(retained)
			}
		}
	}()
	defer func() { close(stop); <-stopped }()
	buffer := make([]float32, 64)
	durations := make([]int64, b.N)
	missed := 0
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if streams[0].position >= streams[0].total {
			b.StopTimer()
			for j := range streams {
				streams[j], _ = d.Stream(nil)
			}
			b.StartTimer()
		}
		start := time.Now()
		for _, s := range streams {
			s.ReadSamples(buffer)
		}
		elapsed := time.Since(start).Nanoseconds()
		durations[i] = elapsed
		if elapsed > int64(time.Second)*64/SampleRate {
			missed++
		}
	}
	b.StopTimer()
	sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })
	b.ReportMetric(float64(durations[(len(durations)-1)*99/100])/1000, "p99-us")
	b.ReportMetric(float64(durations[len(durations)-1])/1000, "max-us")
	b.ReportMetric(float64(missed), "missed-blocks")
}

func TestStreamMemoryDoesNotRetainOutput(t *testing.T) {
	d := streamDefinition(t)
	d.Output["duration_ms"] = json.RawMessage(`10000`)
	s, err := d.Stream(nil)
	if err != nil {
		t.Fatal(err)
	}
	buffer := make([]float32, 256)
	for s.position < SampleRate {
		s.ReadSamples(buffer)
	}
	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	for s.position < s.total {
		if _, err := s.ReadSamples(buffer); err != nil && err != io.EOF {
			t.Fatal(err)
		}
	}
	runtime.GC()
	runtime.ReadMemStats(&after)
	runtime.KeepAlive(s)
	// Nine more seconds would retain megabytes if filters cached sample history.
	if after.HeapAlloc > before.HeapAlloc+128*1024 {
		t.Fatalf("retained heap grew %d bytes", after.HeapAlloc-before.HeapAlloc)
	}
}

func TestStreamBoundsInstanceExpansion(t *testing.T) {
	d := streamDefinition(t)
	previous := "filter.audio"
	add := func(id, kind string, inputs map[string]any) {
		encoded, _ := json.Marshal(inputs)
		var raw map[string]json.RawMessage
		json.Unmarshal(encoded, &raw)
		d.Graph[id] = node{Module: kind, Inputs: raw}
	}
	for i := 0; i < 10; i++ {
		a, b, sum := fmt.Sprintf("a%d", i), fmt.Sprintf("b%d", i), fmt.Sprintf("sum%d", i)
		for _, id := range []string{a, b} {
			add(id, "place", map[string]any{"audio": map[string]string{"ref": previous}, "start_ms": 0, "duration_ms": 1000})
		}
		add(sum, "mix", map[string]any{"audio_a": map[string]string{"ref": a + ".audio"}, "audio_b": map[string]string{"ref": b + ".audio"}, "duration_a_ms": 1000, "duration_b_ms": 1000})
		previous = sum + ".audio"
	}
	if _, err := d.Stream(nil); err == nil || !strings.Contains(err.Error(), "instance expansion") {
		t.Fatalf("expansion error: %v", err)
	}
}
