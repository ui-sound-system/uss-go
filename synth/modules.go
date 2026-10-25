package synth

type moduleSpec struct {
	inputs []string
	build  func(*module) ports
}

var modules map[string]moduleSpec

func init() {
	modules = map[string]moduleSpec{
		"note_frequency":    {[]string{"note"}, noteFrequency},
		"transpose":         {[]string{"note", "semitones"}, transpose},
		"pitch_sweep":       {[]string{"frequency_hz", "semitones", "duration_ms"}, pitchSweep},
		"fm_oscillator":     {[]string{"frequency_hz", "ratio", "index", "envelope"}, fmOscillator},
		"unison_oscillator": {[]string{"waveform", "frequency_hz", "phase_cycles", "voices", "spread_cents"}, unisonOscillator},
		"oscillator":        {[]string{"waveform", "frequency_hz", "phase", "phase_cycles"}, oscillator},
		"noise":             {[]string{"seed"}, noise},
		"envelope_ahd":      {[]string{"attack_ms", "hold_ms", "hold_until_ms", "decay_ms", "attack_curve", "decay_curve"}, envelopeAHD},
		"envelope_segment":  {[]string{"previous", "duration_ms", "level", "curve"}, envelopeSegmentNode},
		"envelope_segments": {[]string{"segments"}, envelopeSegments},
		"envelope_ad":       {[]string{"attack_ms", "decay_ms", "curve"}, envelopeAD},
		"lowpass":           {[]string{"audio", "envelope", "cutoff_hz", "resonance", "amount"}, lowpass},
		"audio_gain":        {[]string{"audio", "duration_ms", "gain"}, audioGain},
		"reverb":            {[]string{"audio", "duration_ms", "pre_delay_ms", "room_size", "decay_ms", "damping", "wet"}, reverb},
		"amplifier":         {[]string{"audio", "envelope", "gain"}, amplifier},
		"place":             {[]string{"audio", "start_ms", "duration_ms"}, place},
		"number_lerp":       {[]string{"from", "to", "position"}, numberLerp},
		"duration_lerp":     {[]string{"from_ms", "to_ms", "position", "multiplier", "offset_ms"}, durationLerp},
		"mix":               {[]string{"audio_a", "audio_b", "duration_a_ms", "duration_b_ms"}, mix},
	}
	for _, spec := range []struct{ name, left, right, output, op string }{
		{"number_add", "number", "number", "number", "add"},
		{"number_subtract", "number", "number", "number", "subtract"},
		{"number_multiply", "number", "number", "number", "multiply"},
		{"number_divide", "number", "number", "number", "divide"},
		{"duration_add", "duration", "duration", "duration", "add"},
		{"duration_subtract", "duration", "duration", "duration", "subtract"},
		{"duration_multiply", "duration", "number", "duration", "multiply"},
		{"duration_divide", "duration", "number", "duration", "divide"},
		{"duration_ratio", "duration", "duration", "number", "ratio"},
	} {
		modules[spec.name] = moduleSpec{[]string{"a", "b"}, arithmetic(spec.op, spec.left, spec.right, spec.output)}
	}
	for _, kind := range []string{"number", "duration", "note", "text", "audio"} {
		modules[kind+"_select"] = moduleSpec{[]string{"selector", "match", "if_true", "if_false"}, selection(kind)}
	}
}
