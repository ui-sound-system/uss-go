//go:build windows && amd64

package latency

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"time"
)

func play(o options, src *source, p *pressure, out io.Writer) (err error) {
	stats := newStatistics()
	m := newMixer(o.block, stats)
	stop, done, ready := make(chan struct{}), make(chan struct{}), make(chan deviceInfo, 1)
	var audioErr error
	go func() { audioErr = audioLoop(o, m, stop, ready); close(done) }()
	defer func() {
		close(stop)
		<-done
		stats.report(out, p)
		if err == nil {
			err = audioErr
		}
	}()
	var info deviceInfo
	select {
	case info = <-ready:
	case <-done:
		return audioErr
	}
	fmt.Fprintf(out, "%s; device period %d frames = %.3f ms; capacity %d frames = %.3f ms\n", info.backend, info.period, float64(info.period)/48, info.capacity, float64(info.capacity)/48)
	fmt.Fprintf(out, "Windows-reported maximum stream latency: %.3f ms; MMCSS Pro Audio: %t\n", float64(info.latency)/float64(time.Millisecond), info.mmcss)
	if info.fallback != "" {
		fmt.Fprintf(out, "Fallback: %s. Requested period is not applied on this path.\n", info.fallback)
	}
	fmt.Fprintln(out, "Default Windows multimedia output; stereo device feed. No physical latency measurement.")
	fmt.Fprintln(out, "Space=play  M=stream/cached  V=1/4/16 voices  G=stress  P=report  Q/Esc=quit")
	fmt.Fprintf(out, "Mode=%s voices=%d stress=%t\n", o.mode, o.voices, p.enabled.Load())
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	defer signal.Stop(interrupt)
	if o.seconds > 0 {
		return automatic(o, src, m, done, &audioErr, interrupt, out)
	}
	console, err := openConsole()
	if err != nil {
		return err
	}
	defer console.close()
	for {
		select {
		case <-done:
			return audioErr
		case <-interrupt:
			return nil
		default:
		}
		key, err := console.key()
		if err != nil {
			return err
		}
		requested := time.Now()
		switch key {
		case 0:
			continue
		case 0x20:
			t, err := src.trigger(o.mode, o.voices, requested)
			if err != nil {
				return err
			}
			if !m.enqueue(t) {
				fmt.Fprintln(out, "Trigger queue full; wait for active sounds to finish.")
			}
		case 'M':
			if o.mode == streaming {
				o.mode = cached
			} else {
				o.mode = streaming
			}
			fmt.Fprintf(out, "Mode=%s\n", o.mode)
		case 'V':
			switch o.voices {
			case 1:
				o.voices = 4
			case 4:
				o.voices = 16
			default:
				o.voices = 1
			}
			fmt.Fprintf(out, "Voices=%d\n", o.voices)
		case 'G':
			enabled := !p.enabled.Load()
			p.enabled.Store(enabled)
			fmt.Fprintf(out, "Stress=%t\n", enabled)
		case 'P':
			stats.report(out, p)
		case 'Q', 0x1b:
			return nil
		}
	}
}
func automatic(o options, src *source, m *mixer, done <-chan struct{}, audioErr *error, interrupt <-chan os.Signal, out io.Writer) error {
	fmt.Fprintf(out, "Automatic %s test for %.1f seconds, every %s. Ctrl+C stops.\n", o.mode, o.seconds, o.interval)
	end := time.NewTimer(time.Duration(o.seconds * float64(time.Second)))
	defer end.Stop()
	ticker := time.NewTicker(o.interval)
	defer ticker.Stop()
	trigger := func() error {
		t, err := src.trigger(o.mode, o.voices, time.Now())
		if err != nil {
			return err
		}
		m.enqueue(t)
		return nil
	}
	if err := trigger(); err != nil {
		return err
	}
	for {
		select {
		case <-done:
			return *audioErr
		case <-interrupt:
			return nil
		case <-end.C:
			return nil
		case <-ticker.C:
			if err := trigger(); err != nil {
				return err
			}
		}
	}
}
