package synth

import "math"

func audioGain(m *module) ports {
	audio := m.signal("audio", "audio")
	duration := m.number("duration_ms", "duration", 0, maxDuration)
	gain := m.number("gain", "number", 0, 1)
	end := duration
	if gain == 0 {
		end = 0
	}
	return ports{"duration_ms": {"duration", end}, "audio": {"audio", signal(func(t float64) float64 {
		if gain == 0 || t < 0 || t >= duration/1000 {
			return 0
		}
		return audio(t) * gain
	})}}
}

func reverb(m *module) ports {
	audio := m.signal("audio", "audio")
	sourceDuration := m.number("duration_ms", "duration", 0, maxDuration)
	preDelay := m.number("pre_delay_ms", "duration", 0, 250)
	size := m.number("room_size", "number", 0, 1)
	decay := m.number("decay_ms", "duration", 100, 4000)
	damping := m.number("damping", "number", 0, 1) * .9
	wet := m.number("wet", "number", 0, 1)
	if m.err != nil {
		return nil
	}
	if wet == 0 || sourceDuration == 0 {
		return ports{"audio": {"audio", signal(func(float64) float64 { return 0 })}, "duration_ms": {"duration", float64(0)}}
	}
	duration := sourceDuration + preDelay + 2*decay + 200
	if duration > maxDuration {
		m.fail("duration_ms", "reverb tail exceeds 10000 ms; shorten the sound or decay")
		return nil
	}
	length := func(tuning int) int {
		return max(1, int(math.Round(float64(tuning)*SampleRate/44100*(.5+size))))
	}
	type comb struct {
		buffer        []float64
		index         int
		low, feedback float64
	}
	combs := make([]comb, 8)
	for i, tuning := range []int{1116, 1188, 1277, 1356, 1422, 1491, 1557, 1617} {
		count := length(tuning)
		combs[i] = comb{buffer: make([]float64, count), feedback: math.Pow(10, -3*float64(count)/SampleRate/(decay/1000))}
	}
	type diffuser struct {
		buffer []float64
		index  int
	}
	diffusers := make([]diffuser, 4)
	for i, tuning := range []int{556, 441, 341, 225} {
		diffusers[i].buffer = make([]float64, length(tuning))
	}
	preSamples := int(math.Round(preDelay * SampleRate / 1000))
	count := int(math.Round(duration * SampleRate / 1000))
	samples := []float64{}
	output := signal(func(t float64) float64 {
		if t < 0 || t >= duration/1000 {
			return 0
		}
		index := int(math.Round(t * SampleRate))
		for len(samples) <= index {
			n := len(samples)
			inputTime := float64(n-preSamples) / SampleRate
			input := 0.0
			if inputTime >= 0 && inputTime < sourceDuration/1000 {
				input = audio(inputTime)
			}
			value := 0.0
			for i := range combs {
				c := &combs[i]
				delayed := c.buffer[c.index]
				c.low = (1-damping)*delayed + damping*c.low
				c.buffer[c.index] = input + c.feedback*c.low
				c.index = (c.index + 1) % len(c.buffer)
				value += delayed / 8
			}
			for i := range diffusers {
				d := &diffusers[i]
				delayed := d.buffer[d.index]
				out := delayed - .5*value
				d.buffer[d.index] = value + .5*out
				d.index = (d.index + 1) % len(d.buffer)
				value = out
			}
			fade := math.Max(0, math.Min(1, float64(count-1-n)/(SampleRate*.02)))
			samples = append(samples, value*wet*fade)
		}
		return samples[index]
	})
	return ports{"audio": {"audio", output}, "duration_ms": {"duration", duration}}
}
