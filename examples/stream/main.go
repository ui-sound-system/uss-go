// Stream demonstrates device-independent, caller-buffered synthesis.
package main

import (
	"fmt"
	"io"
	"os"

	"github.com/ui-sound-system/uss-go/synth"
)

func run() error {
	if len(os.Args) != 2 {
		return fmt.Errorf("usage: stream playback.json")
	}
	data, err := os.ReadFile(os.Args[1])
	if err != nil {
		return err
	}
	definition, err := synth.Parse(data)
	if err != nil {
		return err
	}
	prepared, err := definition.Prepare()
	if err != nil {
		return err
	}
	voice, err := prepared.Stream(nil)
	if err != nil {
		return err
	}
	defer voice.Close()
	buffer := make([]float32, 64)
	total, blocks := 0, 0
	for {
		n, err := voice.ReadSamples(buffer)
		// Supply buffer[:n] to the caller's mixer or output adapter here. Consume it
		// before reusing buffer. No complete waveform is generated or retained.
		total += n
		if n > 0 {
			blocks++
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}
	fmt.Printf("%d samples, %d blocks, %d Hz mono, peak %.6f\n", total, blocks, synth.SampleRate, voice.Peak())
	return nil
}
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
