package dsp

import (
	"fmt"
	"math"
	"math/bits"
)

func IsPowerOfTwo(n int) bool {
	return n > 0 && n&(n-1) == 0
}

func NextPowerOfTwo(n int) int {
	if n <= 1 {
		return 1
	}
	return 1 << bits.Len(uint(n-1))
}

func FFT(samples []complex128) ([]complex128, error) {
	n := len(samples)
	if !IsPowerOfTwo(n) {
		return nil, fmt.Errorf("dsp: FFT length %d is not a power of two", n)
	}
	out := make([]complex128, n)
	copy(out, samples)
	transform(out, false)
	return out, nil
}

func IFFT(spectrum []complex128) ([]complex128, error) {
	n := len(spectrum)
	if !IsPowerOfTwo(n) {
		return nil, fmt.Errorf("dsp: IFFT length %d is not a power of two", n)
	}
	out := make([]complex128, n)
	copy(out, spectrum)
	transform(out, true)
	scale := complex(1/float64(n), 0)
	for i := range out {
		out[i] *= scale
	}
	return out, nil
}

func transform(buf []complex128, inverse bool) {
	n := len(buf)
	if n < 2 {
		return
	}
	shift := uint(bits.TrailingZeros(uint(n)))
	for i := 0; i < n; i++ {
		j := int(bits.Reverse(uint(i)) >> (bits.UintSize - shift))
		if j > i {
			buf[i], buf[j] = buf[j], buf[i]
		}
	}
	sign := -1.0
	if inverse {
		sign = 1.0
	}
	for size := 2; size <= n; size <<= 1 {
		half := size / 2
		theta := sign * 2 * math.Pi / float64(size)
		wStep := complex(math.Cos(theta), math.Sin(theta))
		for start := 0; start < n; start += size {
			w := complex(1, 0)
			for k := 0; k < half; k++ {
				even := buf[start+k]
				odd := buf[start+k+half] * w
				buf[start+k] = even + odd
				buf[start+k+half] = even - odd
				w *= wStep
			}
		}
	}
}

func Shift(spectrum []complex128) []complex128 {
	n := len(spectrum)
	out := make([]complex128, n)
	half := (n + 1) / 2
	copy(out, spectrum[half:])
	copy(out[n-half:], spectrum[:half])
	return out
}

func ShiftFloat(values []float64) []float64 {
	n := len(values)
	out := make([]float64, n)
	half := (n + 1) / 2
	copy(out, values[half:])
	copy(out[n-half:], values[:half])
	return out
}

func ToComplex128(samples []complex64) []complex128 {
	out := make([]complex128, len(samples))
	for i, s := range samples {
		out[i] = complex(float64(real(s)), float64(imag(s)))
	}
	return out
}

func ToComplex64(samples []complex128) []complex64 {
	out := make([]complex64, len(samples))
	for i, s := range samples {
		out[i] = complex(float32(real(s)), float32(imag(s)))
	}
	return out
}
