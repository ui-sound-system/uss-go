# Pre-streaming reference

These eight cases retain audio recorded by the previous renderer, before the
streaming migration. They cover oscillators, FM, unison, envelopes, filters,
noise and reverb. The source revision and digest are the original suite's.

For the reverb case only, its removed trim node is bypassed in the definition.
The reference retains the original audio prefix, stopping 480 samples before
the old trimmed ending to exclude its terminal fade. No audio was regenerated.
Other cases retain their original full reference. WAV headers reflect the
retained sample counts; PCM payloads are copied from the original recordings.

Tests compare streaming prefixes directly to these recordings. Do not regenerate
this directory with the new engine. The ordinary acceptance generator owns only
`../data` and the main manifest; sync distributes both sets to consumers.
