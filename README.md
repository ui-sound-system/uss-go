# USS Go

Render [UI Sound System](https://github.com/ui-sound-system/uss-editor) playback
JSON in Go. The library has no external dependencies and needs no browser, Node,
YAML parser, or audio device. Rendering produces mono Float32 samples at 48 kHz.

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

Export playback JSON from USS Editor with **Ctrl Shift E**. Parse once and
render repeatedly with named overrides; nil uses exported defaults. Overrides
preserve the definition. Unknown inputs and unsupported modules return errors.
Feed `sound.Samples` into your application's audio output, or save `sound.WAV()`.

The [player example](examples/player/) embeds a Two note sound and supports
`--set NAME=VALUE`, a JSON filename, and `--wav FILE`. Device playback uses Windows
PlaySoundW and blocks until completion. WAV rendering also works on other systems.
The [standalone example](examples/standalone/) retains the hand-written Two note
function for comparison; it is not an automatic Go exporter.

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

GitHub Actions checks formatting, vet, unit and acceptance tests, and the player
build on Linux and Windows for pushes to main/master and pull requests.
