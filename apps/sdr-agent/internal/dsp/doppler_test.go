package dsp

import (
	"math"
	"testing"
)

func TestEstimateCarrierRecoversKnownOffset(t *testing.T) {
	const sampleRate = 2_048_000
	const center = 437_500_000

	for _, offset := range []float64{0, 1_337, -4_200, 51_234.7, -180_000} {
		samples := tone(65536, sampleRate, offset, 1, 0.02, 99)
		estimate, err := EstimateCarrier(samples, PSDOptions{FFTSize: 4096, Window: WindowHann, SampleRateHz: sampleRate, CenterFreqHz: center})
		if err != nil {
			t.Fatalf("offset %.1f: %v", offset, err)
		}
		if math.Abs(estimate.OffsetHz-offset) > 20 {
			t.Fatalf("offset %.1f Hz: estimated %.1f Hz (method=%s, bin=%.1f Hz)", offset, estimate.OffsetHz, estimate.Method, estimate.BinWidthHz)
		}
		if !estimate.Converged {
			t.Fatalf("offset %.1f Hz: fine estimation did not converge (SNR=%.1f dB)", offset, estimate.SNRDb)
		}
	}
}

func TestEstimateCarrierBeatsBinResolution(t *testing.T) {
	const sampleRate = 2_048_000
	binWidth := float64(sampleRate) / 4096
	offset := 3.5 * binWidth

	samples := tone(65536, sampleRate, offset, 1, 0.01, 123)
	estimate, err := EstimateCarrier(samples, PSDOptions{FFTSize: 4096, Window: WindowHann, SampleRateHz: sampleRate})
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(estimate.OffsetHz-offset) > binWidth/10 {
		t.Fatalf("estimated %.2f Hz, want %.2f Hz within a tenth of a bin (%.1f Hz)", estimate.OffsetHz, offset, binWidth/10)
	}
}

func TestEstimateCarrierFallsBackOnLowSNR(t *testing.T) {
	const sampleRate = 1_024_000
	samples := tone(32768, sampleRate, 10_000, 0.005, 1.0, 7)
	estimate, err := EstimateCarrier(samples, PSDOptions{FFTSize: 2048, Window: WindowHann, SampleRateHz: sampleRate})
	if err != nil {
		t.Fatal(err)
	}
	if estimate.Converged {
		t.Fatalf("fine estimation should not claim convergence at %.1f dB SNR", estimate.SNRDb)
	}
	if estimate.Method == "psd-peak+kay" {
		t.Fatal("low-SNR captures must not use the Kay estimator")
	}
}

func TestKayEstimatorOnCleanTone(t *testing.T) {
	const sampleRate = 1_000_000
	const offset = 1_234.5
	samples := tone(8192, sampleRate, offset, 1, 0, 1)
	estimated, ok := KayFrequencyOffset(samples, sampleRate)
	if !ok {
		t.Fatal("Kay estimator reported failure on a noise-free tone")
	}
	if math.Abs(estimated-offset) > 0.5 {
		t.Fatalf("Kay estimate = %.3f Hz, want %.3f Hz", estimated, offset)
	}
}

func TestKayEstimatorRejectsShortInput(t *testing.T) {
	if _, ok := KayFrequencyOffset(make([]complex128, 2), 1000); ok {
		t.Fatal("expected failure for fewer than three samples")
	}
	if _, ok := KayFrequencyOffset(make([]complex128, 100), 0); ok {
		t.Fatal("expected failure for a zero sample rate")
	}
}

func TestMixDownCentersTheCarrier(t *testing.T) {
	const sampleRate = 1_024_000
	const offset = 77_000
	samples := tone(16384, sampleRate, offset, 1, 0, 2)
	baseband := MixDown(samples, offset, sampleRate)
	residual, ok := KayFrequencyOffset(baseband, sampleRate)
	if !ok {
		t.Fatal("Kay estimator failed on the mixed-down signal")
	}
	if math.Abs(residual) > 0.5 {
		t.Fatalf("residual after mixing = %.3f Hz, expected close to zero", residual)
	}
}

func TestRangeRateConversionIsSelfConsistent(t *testing.T) {
	const nominal = 437_500_000.0
	for _, rangeRate := range []float64{-7500, -100, 0, 250, 7500} {
		offset, err := OffsetFromRangeRateHz(rangeRate, nominal)
		if err != nil {
			t.Fatal(err)
		}
		restored, err := RangeRateFromOffsetMps(offset, nominal)
		if err != nil {
			t.Fatal(err)
		}
		if math.Abs(restored-rangeRate) > 1e-6 {
			t.Fatalf("round trip for %.1f m/s produced %.6f m/s", rangeRate, restored)
		}
	}
}

func TestApproachingSatelliteRaisesObservedFrequency(t *testing.T) {
	offset, err := OffsetFromRangeRateHz(-7000, 437_500_000)
	if err != nil {
		t.Fatal(err)
	}
	if offset <= 0 {
		t.Fatalf("an approaching satellite must produce a positive Doppler offset, got %.1f Hz", offset)
	}
	if math.Abs(offset-10214) > 50 {
		t.Fatalf("offset = %.1f Hz, expected roughly 10.2 kHz for 7 km/s at 437.5 MHz", offset)
	}
}

func TestRangeRateRejectsInvalidNominalFrequency(t *testing.T) {
	if _, err := RangeRateFromOffsetMps(100, 0); err == nil {
		t.Fatal("expected an error for a zero nominal frequency")
	}
	if _, err := OffsetFromRangeRateHz(100, -1); err == nil {
		t.Fatal("expected an error for a negative nominal frequency")
	}
}

func TestCompareWithPredictionFlagsResidual(t *testing.T) {
	estimate := CarrierEstimate{OffsetHz: 10_500, BinWidthHz: 500, SNRDb: 22}

	within := CompareWithPrediction(estimate, 10_300, 500)
	if !within.WithinTolerance {
		t.Fatalf("a 200 Hz residual should be within a 500 Hz tolerance, got %+v", within)
	}
	if math.Abs(within.ResidualBins-0.4) > 1e-9 {
		t.Fatalf("residual bins = %v, want 0.4", within.ResidualBins)
	}

	outside := CompareWithPrediction(estimate, 8_000, 500)
	if outside.WithinTolerance {
		t.Fatal("a 2500 Hz residual must be reported as out of tolerance")
	}
}

func TestCompareWithPredictionDefaultsToleranceToBinWidth(t *testing.T) {
	estimate := CarrierEstimate{OffsetHz: 1000, BinWidthHz: 250}
	result := CompareWithPrediction(estimate, 900, 0)
	if result.ToleranceHz != 250 {
		t.Fatalf("tolerance = %v, want the bin width 250", result.ToleranceHz)
	}
}

func FuzzEstimateCarrier(f *testing.F) {
	f.Add(int64(4096), 1_024_000.0, 12_000.0, 0.5, 0.05)
	f.Add(int64(8192), 2_048_000.0, -250_000.0, 1.0, 0.5)

	f.Fuzz(func(t *testing.T, count int64, sampleRate, offset, amplitude, noise float64) {
		if count < 4096 || count > 1<<17 {
			t.Skip()
		}
		if sampleRate <= 0 || sampleRate > 1e9 || math.IsNaN(sampleRate) || math.IsInf(sampleRate, 0) {
			t.Skip()
		}
		if math.IsNaN(offset) || math.IsInf(offset, 0) || math.IsNaN(amplitude) || math.IsInf(amplitude, 0) {
			t.Skip()
		}
		if math.IsNaN(noise) || math.IsInf(noise, 0) || noise < 0 || noise > 100 {
			t.Skip()
		}

		samples := tone(int(count), sampleRate, offset, amplitude, noise, 1)
		estimate, err := EstimateCarrier(samples, PSDOptions{FFTSize: 2048, Window: WindowHann, SampleRateHz: sampleRate})
		if err != nil {
			return
		}
		if math.IsNaN(estimate.OffsetHz) || math.IsInf(estimate.OffsetHz, 0) {
			t.Fatalf("estimator produced a non-finite offset for sampleRate=%v offset=%v", sampleRate, offset)
		}
		if math.Abs(estimate.OffsetHz) > sampleRate {
			t.Fatalf("estimated offset %v exceeds the sample rate %v", estimate.OffsetHz, sampleRate)
		}
	})
}
