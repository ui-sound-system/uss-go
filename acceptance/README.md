# USS acceptance suite, version 1

This directory is owned by `uss-editor` and copied into each consumer. Tests
read their local copy without Node, another checkout, or network access.

`manifest.json` contains:

- `format_version`: the manifest and reference encoding version (currently 1).
- `source.repository`, `source.revision`, `source.sha256`: generator identity,
  Git HEAD at generation, and SHA-256 of the actual synthesis/compiler/preset
  sources and reference-case definitions, with CRLF normalized to LF. The digest includes uncommitted edits;
  the revision alone does not describe those edits.
- `tolerance.float32_absolute`: maximum absolute difference per Float32 sample.
- `tolerance.pcm16_absolute`: maximum integer difference per PCM16 sample.
- `cases`: uniquely named cases with a playback JSON `definition` path, named
  `inputs` overrides, and `expected` audio dimensions and reference paths.

All paths are relative to this directory. Each `expected` object specifies
`sample_rate`, `channels`, `sample_count`, `samples`, and `wav`. `samples` is raw
IEEE 754 Float32, little endian, with no header; its byte length is sample count
times channels times four. Version 1 audio is 48000 Hz mono. Samples must be finite.
`wav` is the full canonical 44-byte-header PCM16 WAV produced by the generator.
Runners require matching lengths and identical headers, then compare sample
values using the declared tolerances. Do not compare whole-file hashes across
implementations: floating-point math may differ slightly across platforms.

From `uss-editor`:

```sh
op acceptance-generate   # explicitly replace reference data from current JS
op acceptance-sync       # copy the reviewed suite into sibling consumers
op acceptance-check      # require identical copies and run native checks in order
```

Generation reuses all cases in `tests/render-reference.js`. Review reference
changes before syncing; numerical agreement alone does not establish that a
changed sound is desirable. Ordinary tests and checks never regenerate references.
Stale source digests and mismatched consumer copies fail checks.

Consumer repositories commit their copies for reproducible independent tests.
Add future consumers to `scripts/acceptance.js` when their native runner exists.
