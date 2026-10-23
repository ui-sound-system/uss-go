// Package playback connects the portable renderer to the Windows audio device.
package playback

import "github.com/ui-sound-system/uss-go/synth"

// Play renders with named overrides, then blocks until playback finishes.
// A nil inputs map uses the exported defaults.
func Play(definition *synth.Definition, inputs synth.Inputs) error {
	sound, err := definition.Render(inputs)
	if err != nil {
		return err
	}
	return PlaySound(sound)
}

func PlaySound(sound *synth.Sound) error { return playWAV(sound.WAV()) }
