# USS Go

Synthesize [UI Sound System](https://github.com/ui-sound-system/uss-editor) playback
JSON in Go. The library has no external dependencies and needs no browser, Node,
YAML parser, or audio device. Streaming and full rendering produce mono Float32 samples at 48 kHz.

Requires Go 1.26+.

```sh
go get github.com/ui-sound-system/uss-go/synth
```

```go
import "github.com/ui-sound-system/uss-go/synth"

definition, err := synth.Parse(exportedJSON)
if err != nil {
    return err
}
sound, err := definition.Render(synth.Inputs{"note": "G5"})
if err != nil {
    return err
}
wav := sound.WAV() // PCM16 WAV bytes
_ = wav
```

Export playback JSON from USS Editor. Parse once and
render repeatedly with named overrides; nil uses exported defaults. Overrides
preserve the definition. Unknown inputs and unsupported modules return errors.
Feed `sound.Samples` into your application's audio output, or save `sound.WAV()`.

For incremental synthesis, prepare once and create independent voices:

```go
prepared, err := definition.Prepare()
if err != nil { return err }
voice, err := prepared.Stream(synth.Inputs{"note": "G5"})
if err != nil { return err }
defer voice.Close()
buffer := make([]float32, 64)
for {
    n, err := voice.ReadSamples(buffer)
    // Consume buffer[:n] before reusing it, including when err == io.EOF.
    if err == io.EOF { break }
    if err != nil { return err }
}
```

Import `io` for EOF handling. Preparation and voice creation validate and allocate
state without generating audio; perform them outside the audio callback.
`Prepared` owns an immutable snapshot. Each voice binds its own input values and
DSP state. Process a voice from one goroutine; independent voices may run
concurrently. `definition.Stream(inputs)` is the convenient direct equivalent.

`ReadSamples` accepts any buffer size, returns the final samples with `io.EOF`,
and leaves the suffix after `n` untouched. Later reads return `(0, io.EOF)`;
empty reads return `(0, nil)`. Non-finite output returns a valid prefix and a
sticky error. `Peak()` measures delivered samples; `DurationMS()` includes
explicit effect tails. `Close()` abandons the voice and releases DSP references.
No background workers, audio device, or automatic waveform cache are involved.
Use `prepared.Render(inputs)` or `definition.Render(inputs)` to collect and cache
a complete sound. Existing Float32 samples and PCM16 WAV export remain available.

For byte-oriented playback, wrap a fresh voice instead of calling `ReadSamples`:

```go
reader := voice.Float32LEReader()
defer reader.Close()
// Pass reader to an audio player accepting io.Reader.
```

The reader produces raw 48 kHz mono Float32 little-endian PCM without a WAV header,
synthesizing on demand. It supports arbitrary byte-buffer sizes and implements
`io.ReadCloser`. The reader and voice share one cursor; do not read them separately.
Stop an asynchronous player's reads before closing the reader.

Connections advance on one sample timeline. Shared nodes compute once per frame;
`place` creates an independent instance of its upstream graph, with onset rounded
to the nearest sample. DSP scratch storage is bounded independently of elapsed
playback. Graphs allow 256 authored modules and at most 4096 expanded module
instances, including validation. Automatic trimming has been removed: reverb
retains its explicit finite tail. Live parameter mutation, seeking, arbitrary
feedback, and device scheduling are outside this API.

The [stream example](examples/stream/) processes a JSON file in reusable 64-sample
buffers without an audio device:

```sh
go run ./examples/stream synth/testdata/delayed-bleep.resolved.json
```

The [player example](examples/player/) embeds a Two note sound and supports
`--set NAME=VALUE`, a JSON filename, and `--wav FILE`. Device playback uses Windows
PlaySoundW and blocks until completion. WAV rendering also works on other systems.
The [standalone example](examples/standalone/) retains the hand-written Two note
function for comparison; it is not an automatic Go exporter.

The [Oto example](examples/oto/) streams through `Float32LEReader()` to a
cross-platform audio player, with Cgo-free desktop builds. It has its own Go module
so the synthesis library keeps no external dependencies. The example embeds a sound
and accepts an optional JSON filename and `-volume` (0 to 1).
`op build-oto-windows` builds `dist/uss-oto.exe`, ready to run on Windows without Go
or additional sound files. See the example for player setup and safe reader shutdown.

```sh
op check
op player --set note=G5 --wav /tmp/preview.wav
op build-windows
```

Tests use the checked-in [acceptance suite](acceptance/README.md), independently
of sibling repositories and network access. Float32 absolute error is at most
`1e-6`; PCM16 samples may differ by one quantization step. Each suite records its
format version, generator revision, and source digest.

During development, keep `uss-editor` and `uss-go` as siblings. From the
editor run `op acceptance-generate`, review the reference changes, then run
`op acceptance-sync` and `op acceptance-check`. Each repository versions its own
suite copy. Git commits and releases are managed separately.

Exported mapped choices retain their labels: pass `synth.Inputs{"mode": "close"}`
to select an event pattern, or `synth.Inputs{"direction": "down"}` for a mapped
scalar control. Referenced note and timing controls remain adjustable. The
renderer uses the exported graph and does not need YAML or event-list parsing.

`op check` also vets and tests the separate Oto example with Cgo disabled.
GitHub Actions checks formatting, vet, unit and acceptance tests, the player build,
and the Cgo-free Oto example build on Linux and Windows for pushes to main/master
and pull requests.

For device testing, the [Windows latency tester](examples/latency/) keeps WASAPI
open and compares streamed/cached playback, overlapping voices and optional Go
allocation pressure. `op build-latency-windows` produces `dist/uss-latency.exe`;
Windows needs no Go installation or additional sound files. `--bench` also works
on other platforms. Its software timing and queue observations do not measure
physical key-to-speaker latency.
