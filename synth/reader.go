package synth

import (
	"encoding/binary"
	"io"
	"math"
)

// Float32Reader exposes raw 48000 Hz mono Float32 little-endian PCM, without a
// WAV header. It owns its stream cursor and bounded scratch storage. Use one
// goroutine, and do not read the underlying Stream separately or create another
// reader for it. Stop any asynchronous consumer before calling Close.
type Float32Reader struct {
	stream      *Stream
	samples     [BlockSize]float32
	bytes       [BlockSize * 4]byte
	offset, end int
	terminal    error
}

var _ io.ReadCloser = (*Float32Reader)(nil)

// Float32LEReader wraps this voice without generating samples. The caller owns
// the reader and must close it when finished. ReadSamples remains available for
// consumers that work directly with floats; both interfaces share one cursor.
func (s *Stream) Float32LEReader() *Float32Reader { return &Float32Reader{stream: s} }

// Read synthesizes on demand. Arbitrary byte lengths, including partial samples,
// are supported. The final bytes may arrive together with io.EOF or a synthesis
// error; callers must consume n bytes before handling that error.
func (r *Float32Reader) Read(p []byte) (written int, err error) {
	if len(p) == 0 {
		return 0, nil
	}
	for written < len(p) {
		if r.offset == r.end {
			if r.terminal != nil {
				return written, r.terminal
			}
			count := min(BlockSize, (len(p)-written-1)/4+1)
			n, readErr := r.stream.ReadSamples(r.samples[:count])
			for i, sample := range r.samples[:n] {
				binary.LittleEndian.PutUint32(r.bytes[i*4:], math.Float32bits(sample))
			}
			r.offset, r.end, r.terminal = 0, n*4, readErr
			if n == 0 {
				return written, readErr
			}
		}
		n := copy(p[written:], r.bytes[r.offset:r.end])
		written += n
		r.offset += n
	}
	if r.offset == r.end {
		return written, r.terminal
	}
	return written, nil
}

// Close abandons unread bytes and releases the stream's DSP state. It is
// idempotent and must not race with Read or other use of the underlying stream.
func (r *Float32Reader) Close() error {
	if r.stream != nil {
		r.stream.Close()
		r.stream = nil
	}
	r.offset, r.end, r.terminal = 0, 0, io.EOF
	return nil
}
