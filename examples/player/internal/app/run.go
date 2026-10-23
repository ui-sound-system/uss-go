package app

import (
	_ "embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ui-sound-system/uss-go/examples/player/internal/playback"
	"github.com/ui-sound-system/uss-go/synth"
)

//go:embed two-note.json
var defaultSound []byte

type settings synth.Inputs

func (s settings) String() string { return "NAME=VALUE" }
func (s settings) Set(text string) error {
	name, value, ok := strings.Cut(text, "=")
	if !ok || name == "" || value == "" {
		return fmt.Errorf("expected NAME=VALUE")
	}
	var scalar any
	if err := json.Unmarshal([]byte(value), &scalar); err != nil {
		scalar = value
	}
	s[name] = scalar
	return nil
}

func Run(args []string, out, diagnostics io.Writer) error {
	options := flag.NewFlagSet("go.exe", flag.ContinueOnError)
	options.SetOutput(diagnostics)
	wavPath := options.String("wav", "", "render to a WAV file instead of playing")
	overrides := settings{}
	options.Var(overrides, "set", "override an exposed input: NAME=VALUE (repeatable)")
	options.Usage = func() {
		fmt.Fprintln(diagnostics, "Usage: go.exe [--set note=G5] [--wav FILE] [sound.json]")
		fmt.Fprintln(diagnostics, "Without a file, plays the embedded Two note example. Flags precede the file.")
		options.PrintDefaults()
	}
	if err := options.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if options.NArg() > 1 {
		return fmt.Errorf("expected at most one sound.json file")
	}
	data := defaultSound
	if options.NArg() == 1 {
		var err error
		data, err = os.ReadFile(options.Arg(0))
		if err != nil {
			return err
		}
	}
	definition, err := synth.Parse(data)
	if err != nil {
		return err
	}
	sound, err := definition.Render(synth.Inputs(overrides))
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "%s: %g ms, 48000 Hz mono, peak %.3f\n", sound.Name, sound.DurationMS(), sound.Peak)
	if *wavPath != "" {
		return os.WriteFile(*wavPath, sound.WAV(), 0644)
	}
	return playback.PlaySound(sound)
}
