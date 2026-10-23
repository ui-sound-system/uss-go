//go:build !windows

package playback

import "fmt"

func playWAV([]byte) error {
	return fmt.Errorf("device playback requires Windows; use --wav FILE to render instead")
}
