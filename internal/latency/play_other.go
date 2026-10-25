//go:build !windows || !amd64

package latency

import (
	"fmt"
	"io"
)

func play(options, *source, *pressure, io.Writer) error {
	return fmt.Errorf("audio playback requires Windows; --bench works on this platform")
}
