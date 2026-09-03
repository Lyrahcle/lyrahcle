package dsp

import (
	"math"
	"math/cmplx"
	"math/rand"
	"testing"
)

func naiveDFT(samples []complex128) []complex128 {
	n := len(samples)
	out := make([]complex128, n)
	for k := 0; k < n; k++ {
		var sum complex128
		for j := 0; j < n; j++ {
			angle := -2 * math.Pi * float64(k) * float64(j) / float64(n)
			sum += samples[j] * cmplx.Exp(complex(0, angle))
		}
		out[k] = sum
	}
	return out
}

func TestFFTMatchesNaiveDFT(t *testing.T) {
	source := rand.New(rand.NewSource(7))
	for _, size := range []int{2, 8, 64, 256} {
		samples := make([]complex128, size)
		for i := range samples {
			samples[i] = complex(source.NormFloat64(), source.NormFloat64())
		}
		got, err := FFT(samples)
		if err != nil {
			t.Fatalf("FFT(size=%d): %v", size, err)
		}
		want := naiveDFT(samples)
		for i := range want {
			if cmplx.Abs(got[i]-want[i]) > 1e-9*float64(size) {
				t.Fatalf("size=%d bin=%d: got %v, want %v", size, i, got[i], want[i])
			}
		}
	}
}

func TestFFTRejectsNonPowerOfTwo(t *testing.T) {
	if _, err := FFT(make([]complex128, 12)); err == nil {
		t.Fatal("expected an error for a non power-of-two length")
	}
}

func TestIFFTRoundTrip(t *testing.T) {
	source := rand.New(rand.NewSource(11))
	samples := make([]complex128, 512)
	for i := range samples {
		samples[i] = complex(source.NormFloat64(), source.NormFloat64())
	}
	spectrum, err := FFT(samples)
	if err != nil {
		t.Fatal(err)
	}
	restored, err := IFFT(spectrum)
	if err != nil {
		t.Fatal(err)
	}
	for i := range samples {
		if cmplx.Abs(restored[i]-samples[i]) > 1e-9 {
			t.Fatalf("sample %d: got %v, want %v", i, restored[i], samples[i])
		}
	}
}

func TestFFTSatisfiesParseval(t *testing.T) {
	source := rand.New(rand.NewSource(13))
	n := 1024
	samples := make([]complex128, n)
	timeEnergy := 0.0
	for i := range samples {
		samples[i] = complex(source.NormFloat64(), source.NormFloat64())
		timeEnergy += real(samples[i])*real(samples[i]) + imag(samples[i])*imag(samples[i])
	}
	spectrum, err := FFT(samples)
	if err != nil {
		t.Fatal(err)
	}
	frequencyEnergy := 0.0
	for _, bin := range spectrum {
		frequencyEnergy += real(bin)*real(bin) + imag(bin)*imag(bin)
	}
	frequencyEnergy /= float64(n)
	if math.Abs(timeEnergy-frequencyEnergy) > 1e-6*timeEnergy {
		t.Fatalf("Parseval mismatch: time=%v frequency=%v", timeEnergy, frequencyEnergy)
	}
}

func TestFFTPlacesToneInExpectedBin(t *testing.T) {
	n := 256
	targetBin := 37
	samples := make([]complex128, n)
	for i := range samples {
		angle := 2 * math.Pi * float64(targetBin) * float64(i) / float64(n)
		samples[i] = cmplx.Exp(complex(0, angle))
	}
	spectrum, err := FFT(samples)
	if err != nil {
		t.Fatal(err)
	}
	peak := 0
	for i, bin := range spectrum {
		if cmplx.Abs(bin) > cmplx.Abs(spectrum[peak]) {
			peak = i
		}
	}
	if peak != targetBin {
		t.Fatalf("peak bin = %d, want %d", peak, targetBin)
	}
	if magnitude := cmplx.Abs(spectrum[targetBin]); math.Abs(magnitude-float64(n)) > 1e-6 {
		t.Fatalf("peak magnitude = %v, want %v", magnitude, n)
	}
}

func TestNextPowerOfTwo(t *testing.T) {
	cases := map[int]int{0: 1, 1: 1, 2: 2, 3: 4, 1000: 1024, 4096: 4096}
	for input, want := range cases {
		if got := NextPowerOfTwo(input); got != want {
			t.Fatalf("NextPowerOfTwo(%d) = %d, want %d", input, got, want)
		}
	}
}

func TestShiftMovesDCToCenter(t *testing.T) {
	values := []float64{0, 1, 2, 3, 4, 5, 6, 7}
	shifted := ShiftFloat(values)
	want := []float64{4, 5, 6, 7, 0, 1, 2, 3}
	for i := range want {
		if shifted[i] != want[i] {
			t.Fatalf("index %d: got %v, want %v", i, shifted[i], want[i])
		}
	}
}
