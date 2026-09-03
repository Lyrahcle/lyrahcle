package dsp

import (
	"math"
	"math/cmplx"
	"math/rand"
	"testing"
)

func tone(n int, sampleRateHz, toneHz, amplitude float64, noiseSigma float64, seed int64) []complex128 {
	source := rand.New(rand.NewSource(seed))
	out := make([]complex128, n)
	for i := range out {
		phase := 2 * math.Pi * toneHz * float64(i) / sampleRateHz
		signal := complex(amplitude, 0) * cmplx.Exp(complex(0, phase))
		if noiseSigma > 0 {
			signal += complex(source.NormFloat64()*noiseSigma, source.NormFloat64()*noiseSigma)
		}
		out[i] = signal
	}
	return out
}

func TestWelchLocatesToneFrequency(t *testing.T) {
	const sampleRate = 2_048_000
	const centerFreq = 437_500_000
	const toneOffset = 120_000

	samples := tone(32768, sampleRate, toneOffset, 1, 0.01, 3)
	psd, err := Welch(samples, PSDOptions{FFTSize: 4096, Window: WindowHann, SampleRateHz: sampleRate, CenterFreqHz: centerFreq})
	if err != nil {
		t.Fatal(err)
	}
	peak := psd.PeakBin()
	if math.Abs(peak.OffsetHz-toneOffset) > psd.BinWidthHz {
		t.Fatalf("peak offset = %.1f Hz, want %.1f Hz within one bin (%.1f Hz)", peak.OffsetHz, float64(toneOffset), psd.BinWidthHz)
	}
	if math.Abs(peak.FrequencyHz-(centerFreq+toneOffset)) > psd.BinWidthHz {
		t.Fatalf("peak frequency = %.1f Hz, want %.1f Hz", peak.FrequencyHz, float64(centerFreq+toneOffset))
	}
	if peak.SNRDb < 20 {
		t.Fatalf("SNR = %.1f dB, expected a clearly detectable tone", peak.SNRDb)
	}
}

func TestWelchNegativeOffsetTone(t *testing.T) {
	const sampleRate = 2_048_000
	samples := tone(32768, sampleRate, -300_000, 1, 0.01, 5)
	psd, err := Welch(samples, PSDOptions{FFTSize: 4096, Window: WindowHann, SampleRateHz: sampleRate, CenterFreqHz: 100_000_000})
	if err != nil {
		t.Fatal(err)
	}
	peak := psd.PeakBin()
	if math.Abs(peak.OffsetHz+300_000) > psd.BinWidthHz {
		t.Fatalf("peak offset = %.1f Hz, want -300000 Hz", peak.OffsetHz)
	}
}

func TestWelchToneAmplitudeScaling(t *testing.T) {
	const sampleRate = 1_024_000
	strong := tone(16384, sampleRate, 50_000, 1.0, 0, 1)
	weak := tone(16384, sampleRate, 50_000, 0.1, 0, 1)

	options := PSDOptions{FFTSize: 4096, Window: WindowRectangular, SampleRateHz: sampleRate}
	strongPSD, err := Welch(strong, options)
	if err != nil {
		t.Fatal(err)
	}
	weakPSD, err := Welch(weak, options)
	if err != nil {
		t.Fatal(err)
	}
	delta := strongPSD.PeakBin().PowerDbFS - weakPSD.PeakBin().PowerDbFS
	if math.Abs(delta-20) > 0.5 {
		t.Fatalf("amplitude ratio of 10 should be 20 dB, got %.2f dB", delta)
	}
}

func TestWelchRejectsShortInput(t *testing.T) {
	if _, err := Welch(make([]complex128, 100), PSDOptions{FFTSize: 1024, SampleRateHz: 1000}); err == nil {
		t.Fatal("expected an error when there are fewer samples than the FFT size")
	}
}

func TestWelchRejectsInvalidOptions(t *testing.T) {
	samples := make([]complex128, 8192)
	if _, err := Welch(samples, PSDOptions{FFTSize: 1000, SampleRateHz: 1000}); err == nil {
		t.Fatal("expected an error for a non power-of-two FFT size")
	}
	if _, err := Welch(samples, PSDOptions{FFTSize: 1024, SampleRateHz: 0}); err == nil {
		t.Fatal("expected an error for a zero sample rate")
	}
	if _, err := Welch(samples, PSDOptions{FFTSize: 1024, SampleRateHz: 1000, OverlapRatio: 1.5}); err == nil {
		t.Fatal("expected an error for an out-of-range overlap ratio")
	}
	if _, err := Welch(samples, PSDOptions{FFTSize: 1024, SampleRateHz: 1000, Window: "triangular"}); err == nil {
		t.Fatal("expected an error for an unsupported window")
	}
}

func TestNoiseFloorTracksNoisePower(t *testing.T) {
	const sampleRate = 1_024_000
	quiet := tone(32768, sampleRate, 0, 0, 0.01, 21)
	loud := tone(32768, sampleRate, 0, 0, 0.1, 21)

	options := PSDOptions{FFTSize: 2048, Window: WindowHann, SampleRateHz: sampleRate}
	quietPSD, err := Welch(quiet, options)
	if err != nil {
		t.Fatal(err)
	}
	loudPSD, err := Welch(loud, options)
	if err != nil {
		t.Fatal(err)
	}
	delta := loudPSD.NoiseFloorDbFS() - quietPSD.NoiseFloorDbFS()
	if math.Abs(delta-20) > 1 {
		t.Fatalf("a 10x noise amplitude increase should raise the floor by 20 dB, got %.2f dB", delta)
	}
}

func TestOccupiedBandwidthIsNarrowForATone(t *testing.T) {
	const sampleRate = 2_048_000
	samples := tone(32768, sampleRate, 0, 1, 0.001, 33)
	psd, err := Welch(samples, PSDOptions{FFTSize: 4096, Window: WindowBlackmanHarris, SampleRateHz: sampleRate})
	if err != nil {
		t.Fatal(err)
	}
	bandwidth := psd.OccupiedBandwidthHz(0.99)
	if bandwidth <= 0 || bandwidth > sampleRate/50 {
		t.Fatalf("occupied bandwidth = %.1f Hz, expected a narrow value for a pure tone", bandwidth)
	}
}

func TestBandPowerIgnoresOutOfBandEnergy(t *testing.T) {
	const sampleRate = 2_048_000
	const center = 400_000_000
	samples := tone(32768, sampleRate, 500_000, 1, 0.001, 44)
	psd, err := Welch(samples, PSDOptions{FFTSize: 4096, Window: WindowHann, SampleRateHz: sampleRate, CenterFreqHz: center})
	if err != nil {
		t.Fatal(err)
	}
	inBand := psd.BandPowerDbFS(center+450_000, center+550_000)
	outOfBand := psd.BandPowerDbFS(center-550_000, center-450_000)
	if inBand-outOfBand < 30 {
		t.Fatalf("in-band power (%.1f dB) should dominate out-of-band power (%.1f dB)", inBand, outOfBand)
	}
}

func TestWindowGains(t *testing.T) {
	for _, kind := range []WindowKind{WindowRectangular, WindowHann, WindowHamming, WindowBlackmanHarris} {
		window, err := NewWindow(kind, 1024)
		if err != nil {
			t.Fatalf("%s: %v", kind, err)
		}
		if window.CoherentGain <= 0 || window.CoherentGain > 1.0001 {
			t.Fatalf("%s: coherent gain = %v", kind, window.CoherentGain)
		}
		if enbw := window.EquivalentNoiseBandwidthBins(); enbw < 0.99 || enbw > 2.5 {
			t.Fatalf("%s: ENBW = %v bins, outside the expected range", kind, enbw)
		}
	}
	if window, err := NewWindow(WindowRectangular, 512); err != nil {
		t.Fatal(err)
	} else if math.Abs(window.EquivalentNoiseBandwidthBins()-1) > 1e-9 {
		t.Fatalf("a rectangular window must have an ENBW of exactly 1 bin, got %v", window.EquivalentNoiseBandwidthBins())
	}
}
