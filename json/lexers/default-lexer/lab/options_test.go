package lab

import (
	"bytes"
	"testing"
)

// TestBufferSizeAlignment pins the tiny-buffer guard rail: WithBufferSize rounds the
// requested size up to a multiple of bufferSizeAlignment (32 bytes, one AVX2 stride),
// so the streaming window is never narrower than a single vector/SWAR step. This
// floors out the pathological tiny windows that stressed the byte-by-byte refill seam.
func TestBufferSizeAlignment(t *testing.T) {
	cases := []struct {
		in, want int
	}{
		{1, 32}, {2, 32}, {8, 32}, {31, 32}, {32, 32},
		{33, 64}, {63, 64}, {64, 64}, {65, 96},
		{4095, 4096}, {4096, 4096},
	}
	for _, c := range cases {
		if got := alignBufferSize(c.in); got != c.want {
			t.Errorf("alignBufferSize(%d) = %d, want %d", c.in, got, c.want)
		}
	}

	// non-positive sizes are ignored: the option keeps the (aligned) default.
	for _, bad := range []int{0, -1, -32} {
		var o options
		o.applyWithDefaults([]Option{WithBufferSize(bad)})
		if o.bufferSize != defaultBufferBytes {
			t.Errorf("WithBufferSize(%d): bufferSize = %d, want default %d", bad, o.bufferSize, defaultBufferBytes)
		}
	}

	// the aligned size is observable as the allocated window capacity.
	l := New(bytes.NewReader([]byte(`"x"`)), WithBufferSize(10))
	if cap(l.buffer) != 32 {
		t.Errorf("WithBufferSize(10): window cap = %d, want 32", cap(l.buffer))
	}
}
