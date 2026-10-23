package playback

import (
	"fmt"
	"runtime"
	"syscall"
	"unsafe"
)

var playSound = syscall.NewLazyDLL("winmm.dll").NewProc("PlaySoundW")

func playWAV(wav []byte) error {
	if err := playSound.Find(); err != nil {
		return err
	}
	const sndMemory = 0x4
	const sndNoDefault = 0x2
	// Synchronous playback keeps the WAV buffer alive until Windows is finished.
	result, _, _ := playSound.Call(uintptr(unsafe.Pointer(&wav[0])), 0, sndMemory|sndNoDefault)
	runtime.KeepAlive(wav)
	if result == 0 {
		return fmt.Errorf("Windows could not play the WAV")
	}
	return nil
}
