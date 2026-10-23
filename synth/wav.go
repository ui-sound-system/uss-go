package synth

import (
	"encoding/binary"
	"math"
)

// WAV encodes the sound as mono PCM16, clipping samples outside [-1, 1].
func (s *Sound) WAV() []byte {
	data := make([]byte, 44+2*len(s.Samples))
	copy(data, "RIFF")
	binary.LittleEndian.PutUint32(data[4:], uint32(len(data)-8))
	copy(data[8:], "WAVEfmt ")
	binary.LittleEndian.PutUint32(data[16:], 16)
	binary.LittleEndian.PutUint16(data[20:], 1)
	binary.LittleEndian.PutUint16(data[22:], 1)
	binary.LittleEndian.PutUint32(data[24:], SampleRate)
	binary.LittleEndian.PutUint32(data[28:], SampleRate*2)
	binary.LittleEndian.PutUint16(data[32:], 2)
	binary.LittleEndian.PutUint16(data[34:], 16)
	copy(data[36:], "data")
	binary.LittleEndian.PutUint32(data[40:], uint32(len(data)-44))
	for i, sample := range s.Samples {
		value := math.Max(-1, math.Min(1, float64(sample)))
		scale := 32767.0
		if value < 0 {
			scale = 32768
		}
		// JS Math.round rounds negative half ties toward +infinity too.
		pcm := int16(math.Floor(value*scale + .5))
		binary.LittleEndian.PutUint16(data[44+i*2:], uint16(pcm))
	}
	return data
}
