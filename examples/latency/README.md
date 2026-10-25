# USS Windows latency tester

A self-contained Windows 10/11 x64 console application. No Go installation,
extra DLLs, or sound files are needed. It embeds a short tone with an envelope,
filter and reverb. You can also supply exported USS playback JSON.

## Start

Open PowerShell in the executable's folder:

```powershell
.\uss-latency.exe
```

- Space: trigger a sound immediately, without Enter.
- M: switch new triggers between streaming and cached playback.
- V: cycle 1, 4 and 16 simultaneous voices per trigger.
- G: toggle unrelated Go allocation pressure.
- P: print measurements while keeping the audio device open.
- Q or Escape: quit and print the final measurements. Ctrl+C also exits cleanly.

The device stays open and is continuously supplied with audio or silence.
Each trigger creates fresh voices on the control goroutine; the audio thread
consumes them in reusable blocks. Triggers may overlap, up to 128 active voices.
Per-trigger voices are averaged to keep level comparable. Overlapping triggers
can clip; the clipping counter distinguishes that from possible dropouts.

The cache is deliberately rendered once at startup for this application's A/B
comparison. The synth library has no automatic cache. Changing mode affects the
next trigger; old tails continue, so overlapping different modes appear as mixed.

## Automatic tests and saved reports

These run without keyboard interaction. They use the default Windows multimedia
output device and print a report after 30 seconds:

```powershell
.\uss-latency.exe --seconds 30 --voices 16 --mode stream > stream.txt
.\uss-latency.exe --seconds 30 --voices 16 --mode cached > cached.txt
.\uss-latency.exe --seconds 30 --voices 16 --mode stream --stress > stress.txt
```

Try the same tests while using your normal applications. Listen for crackles,
gaps and sluggish response. Begin with a modest speaker/headphone volume.

For longer sounds, set `--interval 2s` to avoid overlapping triggers. The default
is 500ms; the embedded sound is 490ms. `--seconds` stops at the requested time,
so a final tail may be interrupted. Run `--help` for every option.

## Block size and Windows buffering

```powershell
.\uss-latency.exe --block 64 --period 64
.\uss-latency.exe --block 256 --period 256
.\uss-latency.exe --set note=G5 --set level=0.15
.\uss-latency.exe --voices 4 .\my-sound.json
```

`--block` controls synthesis calls; it does not dictate device latency. `--period`
requests a Windows shared-mode engine period in 48 kHz frames. The default, 0,
requests the supported minimum. The tester rounds to a valid supported multiple
and prints the selected period and actual buffer capacity. Requested 64 frames
may become a larger period on your device. The app requests MMCSS Pro Audio
thread scheduling and reports whether Windows accepted it.

The preferred path is event-driven IAudioClient3 with 48 kHz stereo Float32.
If that format/period cannot be opened, the tester reports a fallback to standard
shared-mode WASAPI with Windows format conversion enabled. The period request
is not applied on the fallback path. The actual period, capacity and Windows'
reported maximum stream latency are always printed. Choosing 48 kHz in Windows
Sound settings may help devices whose engine currently uses another format.
Changing default devices while running can end the test; restart afterwards.

## Read the measurements

- Processing: generation/mixing/conversion of an active synthesis block, excluding
  native device calls. It includes any Go pauses that happen inside that work.
  Compare with 1.333ms for 64 samples, or 5.333ms for 256 samples. Actual packet
  remainders are compared with their own shorter sample budget.
- Over block budget: processing blocks slower than the audio they generated.
- Voice setup: preparation of a triggered batch outside the audio thread. This
  includes instantiating all requested streaming voices, or cached cursors.
- Trigger received -> first buffer submitted: time from reading the console key
  (or servicing the automatic timer) to successful submission of the packet
  containing that trigger's first sample. Includes setup and waiting for an
  audio event. It excludes keyboard hardware/OS delivery before the key is read.
  The first synthesized sample may be silent because of phase or the envelope.
- Queued audio before refill: Windows' padding observation before generating the
  packet. It describes already queued samples ahead of the new trigger; some
  may play during generation. It is not a physical output timestamp.
- Wake gap: elapsed time between serviced device events, including idle periods.
- Empty queue while active: possible starvation observations. An empty buffer
  near its normal refill boundary does not prove an audible underrun. This is
  not an exact driver glitch counter; listen and compare cached mode too.

Percentiles are histogram bucket upper bounds (1 microsecond for processing and
setup, 10 microseconds for trigger/queue/wake timing); overflow uses the maximum.
Reports while playing are approximate snapshots. Final reports follow audio
shutdown. Counters accumulate across mode, voice-count and stress changes; use
separate automatic runs for comparisons with fixed conditions.

Windows' reported maximum stream latency, padding and submission timings overlap
in what they describe. Do not simply add them and call the result end-to-end
latency. Physical trigger-to-speaker measurements require a synchronized trigger
reference and analog recording. Bluetooth/headphone DSP adds delay outside this
processor and may not be fully described by the API.

## CPU-only benchmark

```powershell
.\uss-latency.exe --bench --voices 16 --block 64
.\uss-latency.exe --bench --voices 16 --block 256 --stress
```

This measures 10000 blocks in each mode, without an audio device or real callback
schedule. Voices are replenished outside the timed processing region. It tests
processing headroom and interruptions during processing, not audible latency or
late callback arrivals. Stress allocates 64KiB on approximately 1ms timer ticks;
actual allocated bytes are reported because timer behavior varies by system.

## Build and validation

From `uss-go`, `op build-latency-windows` cross-builds `dist/uss-latency.exe` and
copies these instructions alongside it. `op latency --bench` runs the portable
benchmark on a development machine. There are no added dependencies.

Automated tests cover streamed/cached numerical parity, irregular device packet
sizes, processing allocations, input validation, queue/voice limits and timing
accounting. Windows ABI layouts are checked at compilation. Cross-building does
not verify native playback: real Windows device testing is still required.

WASAPI references: [engine periods](https://learn.microsoft.com/en-us/windows/win32/api/audioclient/nf-audioclient-iaudioclient3-getsharedmodeengineperiod),
[shared stream initialization](https://learn.microsoft.com/en-us/windows/win32/api/audioclient/nf-audioclient-iaudioclient3-initializesharedaudiostream),
[reported stream latency](https://learn.microsoft.com/en-us/windows/win32/api/audioclient/nf-audioclient-iaudioclient-getstreamlatency).
