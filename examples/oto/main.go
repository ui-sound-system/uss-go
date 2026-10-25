// Run from this directory with go run . [sound.json]. This is a separate module
// so using synth alone does not add Oto or its platform dependencies.
package main

import (
	_ "embed"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/ebitengine/oto/v3"
	"github.com/ui-sound-system/uss-go/synth"
)

//go:embed sound.json
var defaultSound []byte

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	volume := flag.Float64("volume", 0.5, "playback volume, 0 to 1")
	flag.Parse()
	if flag.NArg() > 1 || !(*volume >= 0 && *volume <= 1) {
		return fmt.Errorf("usage: uss-oto [-volume 0..1] [sound.json]")
	}
	data := defaultSound
	if flag.NArg() == 1 {
		var err error
		data, err = os.ReadFile(flag.Arg(0))
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
	stream, err := prepared.Stream(nil)
	if err != nil {
		return err
	}
	reader := stream.Float32LEReader()
	defer reader.Close()
	context, ready, err := oto.NewContext(&oto.NewContextOptions{
		SampleRate: synth.SampleRate, ChannelCount: 1, Format: oto.FormatFloat32LE,
		BufferSize: 20 * time.Millisecond,
	})
	if err != nil {
		return err
	}
	<-ready
	if err := context.Err(); err != nil {
		return err
	}
	player := context.NewPlayer(reader)
	// Oto reads the source asynchronously. Stop those reads before closing it.
	defer player.PauseAndStopReading()
	player.SetBufferSize(synth.SampleRate * 4 / 100) // 10 ms of source PCM.
	player.SetVolume(*volume)
	player.Play()
	fmt.Printf("Streaming %s with Oto\n", definition.Name)
	for player.IsPlaying() {
		if err := context.Err(); err != nil {
			return err
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err := player.Err(); err != nil {
		return err
	}
	if err := context.Err(); err != nil {
		return err
	}
	// IsPlaying ends when the source and player buffer have drained, while the
	// device can still hold the last packet. Let it reach the speakers before exit.
	time.Sleep(100 * time.Millisecond)
	return nil
}
