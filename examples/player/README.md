# Go playback example

A standalone Windows console player, with no external Go dependencies. The executable
embeds the **Two note** example and plays it once when started without arguments.
No browser, YAML parser, or recipe generator is needed at playback time.

From the repository root:

```sh
op check
op build-windows
```

The build produces `dist/go.exe` for Windows amd64. Go checks use the local acceptance fixtures; Node is not required.

On Windows, in PowerShell:

```powershell
.\go.exe
.\go.exe --set note=G5
.\go.exe --set note=D5 --set spacing=100
.\go.exe --set note=G5 .\notification.json
.\go.exe --wav preview.wav .\notification.json
```

Export `notification.json` from USS with **Ctrl Shift E**. Flags must precede the
optional JSON filename. Repeat `--set NAME=VALUE` to override any exposed input,
including pitch, duration, interval, level, or waveform (`--set waveform=square`
with a Waveform tone export). Unspecified inputs use the values
chosen when exporting. Ranges and steps still apply; notes clamp to their range,
numeric values outside their range produce an error. Unknown input names are errors.
Older `.resolved.json` exports still play, but contain no adjustable inputs.

## Calling from Go

The example keeps rendering separate from device playback. Its application-facing
function accepts the same named inputs:

```go
// Import github.com/ui-sound-system/uss-go/synth.
// playback is internal to this example.
definition, err := synth.Parse(exportedJSON)
if err != nil {
    return err
}
return playback.Play(definition, synth.Inputs{"note": "G5"})
```

Parse once, then play with different values as the application state changes.
The definition is unchanged by overrides. A nil map uses all exported defaults.
To render without an audio device, use `definition.Render(inputs)`, then
`sound.WAV()` for bytes or `sound.Samples` for mono Float32 samples at 48 kHz.
Rendering and WAV export work on Linux too.

Windows playback uses synchronous
[PlaySoundW](https://learn.microsoft.com/en-us/previous-versions/dd743680(v=vs.85))
with an in-memory WAV. Calls block until the sound finishes; this example does not
mix concurrent calls or provide cancellation. Run it away from an application's
UI thread. Independent synthesis voices within a sound are mixed by the graph.

The renderer is the public `github.com/ui-sound-system/uss-go/synth` package.
Device playback remains internal to this example; asynchronous playback is not provided.

## Sound-specific function example

[standalone/two_note.go](../standalone/two_note.go) demonstrates what a
Go code export of our largest current recipe could look like. It is hand-written;
there is no automatic code exporter yet. The entire implementation is one file,
with only standard-library imports and no dependency on the generic USS renderer.

```go
inputs := standalone.DefaultTwoNoteInputs()
inputs.Note = "G5"
samples, err := standalone.TwoNote(inputs)
if err != nil {
    return err
}
// Send samples to your audio output: mono Float32, 48000 Hz.
```

All six controls remain configurable. Defaults, range checks, note-name parsing,
and step quantization are included in the file. It can be copied into another
application with its package declaration adjusted. WAV encoding and device playback
are separate; they are not included in this file's size.

Tests compare it with the recorded JavaScript acceptance suite for defaults, runtime pitch overrides,
overlapping voices, range limits, alternate note spellings, and non-grid inputs.

## Verification

Go tests compare the renderer with the recorded JavaScript acceptance suite, including pitch overrides,
step quantization, overlapping notes, and old resolved files. Float32 sample error
must be at most `1e-6`; PCM16 difference at most one quantization step. Exact byte
identity is not promised across math libraries and CPU architectures.
This extraction is checked numerically against the reference suite. Audible
playback still requires running the built player on Windows.
