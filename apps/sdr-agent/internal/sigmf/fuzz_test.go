package sigmf

import "testing"

func FuzzParse(f *testing.F) {
	f.Add([]byte(`{"global":{"core:datatype":"cf32_le","core:version":"1.0.0","core:sample_rate":240000},"captures":[{"core:sample_start":0,"core:frequency":437500000}]}`))
	f.Add([]byte(`{"global":{},"captures":[]}`))
	f.Add([]byte(`{"global":{"core:datatype":"cu8","core:version":"1.0.0","core:sample_rate":-1},"captures":[{"core:sample_start":7}]}`))
	f.Add([]byte(`not json`))

	f.Fuzz(func(t *testing.T, data []byte) {
		descriptor, err := Parse(data)
		if err != nil {
			return
		}
		if err := descriptor.Validate(); err != nil {
			t.Fatalf("Parse accepted metadata that Validate rejects: %v", err)
		}
		if _, err := BytesPerSample(descriptor.Global.Datatype); err != nil {
			t.Fatalf("Parse accepted an unsupported datatype: %v", err)
		}
		if descriptor.Global.SampleRate <= 0 {
			t.Fatalf("Parse accepted a non-positive sample rate: %v", descriptor.Global.SampleRate)
		}
		if len(descriptor.Captures) == 0 || descriptor.Captures[0].SampleStart != 0 {
			t.Fatal("Parse accepted metadata without a valid first capture segment")
		}
	})
}
