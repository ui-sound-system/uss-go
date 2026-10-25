package latency

import (
	_ "embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/ui-sound-system/uss-go/synth"
)

//go:embed sound.json
var builtIn []byte

type settings synth.Inputs

func (s settings) String() string { return "NAME=VALUE" }
func (s settings) Set(text string) error {
	name, value, ok := strings.Cut(text, "=")
	if !ok || name == "" || value == "" {
		return fmt.Errorf("expected NAME=VALUE")
	}
	var scalar any
	if json.Unmarshal([]byte(value), &scalar) != nil {
		scalar = value
	}
	s[name] = scalar
	return nil
}

type options struct {
	block, period, voices, iterations int
	seconds                           float64
	interval                          time.Duration
	mode                              mode
	stress, bench                     bool
	inputs                            settings
}

type source struct {
	prepared *synth.Prepared
	cached   []float32
	inputs   synth.Inputs
	name     string
}

// Run provides the benchmark on all platforms and interactive playback on Windows.
func Run(args []string, out, diagnostics io.Writer) error {
	o := options{inputs: settings{}}
	flags := flag.NewFlagSet("uss-latency.exe", flag.ContinueOnError)
	flags.SetOutput(diagnostics)
	flags.IntVar(&o.block, "block", 64, "synthesis block size in samples (1-4096)")
	flags.IntVar(&o.period, "period", 0, "requested device period in frames; 0 selects the supported minimum")
	flags.IntVar(&o.voices, "voices", 1, "simultaneous voices per trigger (1-32)")
	flags.IntVar(&o.iterations, "iterations", 10000, "processing blocks per mode in --bench")
	flags.Float64Var(&o.seconds, "seconds", 0, "automatic playback duration; 0 uses immediate keyboard controls")
	flags.DurationVar(&o.interval, "interval", 500*time.Millisecond, "automatic trigger interval with --seconds")
	modeName := flags.String("mode", "stream", "initial playback mode: stream or cached")
	flags.BoolVar(&o.stress, "stress", false, "add unrelated Go allocation pressure")
	flags.BoolVar(&o.bench, "bench", false, "measure streaming and cached processing without an audio device")
	flags.Var(o.inputs, "set", "named input override NAME=VALUE (repeatable)")
	flags.Usage = func() {
		fmt.Fprintln(diagnostics, "Usage: uss-latency.exe [options] [sound.json]")
		fmt.Fprintln(diagnostics, "No Go installation or sound file needed. Space plays; M changes mode; P reports; Q quits.")
		flags.PrintDefaults()
	}
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if flags.NArg() > 1 {
		return fmt.Errorf("expected at most one sound.json file")
	}
	if o.block < 1 || o.block > 4096 || o.voices < 1 || o.voices > 32 || o.period < 0 || o.period > 48000 || o.iterations < 1 || o.iterations > 1000000 || math.IsNaN(o.seconds) || math.IsInf(o.seconds, 0) || o.seconds < 0 || o.seconds > 86400 || o.interval < 10*time.Millisecond {
		return fmt.Errorf("invalid block, voices, period, iterations, seconds or interval; use --help")
	}
	switch *modeName {
	case "stream":
		o.mode = streaming
	case "cached":
		o.mode = cached
	default:
		return fmt.Errorf("mode must be stream or cached")
	}
	data := builtIn
	if flags.NArg() == 1 {
		var err error
		data, err = os.ReadFile(flags.Arg(0))
		if err != nil {
			return err
		}
	}
	definition, err := synth.Parse(data)
	if err != nil {
		return err
	}
	prepared, err := definition.Prepare()
	if err != nil {
		return err
	}
	sound, err := prepared.Render(synth.Inputs(o.inputs))
	if err != nil {
		return err
	}
	src := &source{prepared: prepared, cached: sound.Samples, inputs: synth.Inputs(o.inputs), name: definition.Name}
	fmt.Fprintf(out, "USS latency tester | %s | 48000 Hz mono synthesis\n", src.name)
	fmt.Fprintf(out, "Runtime: %s/%s %s; GOMAXPROCS=%d; stress=%t\n", runtime.GOOS, runtime.GOARCH, runtime.Version(), runtime.GOMAXPROCS(0), o.stress)
	fmt.Fprintf(out, "Sound %.2f ms; %d voice(s)/trigger; synthesis block %d = %.3f ms\n", sound.DurationMS(), o.voices, o.block, float64(o.block)/48)
	pressure := newPressure(o.stress)
	defer pressure.close()
	if o.bench {
		return benchmark(o, src, pressure, out)
	}
	return play(o, src, pressure, out)
}
