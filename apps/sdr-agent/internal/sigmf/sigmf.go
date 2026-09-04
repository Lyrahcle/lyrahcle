package sigmf

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

const (
	Version       = "1.0.0"
	MetaExtension = ".sigmf-meta"
	DataExtension = ".sigmf-data"

	DatatypeCF32LE = "cf32_le"
	DatatypeCI16LE = "ci16_le"
	DatatypeCI8    = "ci8"
	DatatypeCU8    = "cu8"

	RecorderName = "Lyrahcle sdr-agent"

	maxCaptures     = 1024
	maxAnnotations  = 8192
	maxSampleRateHz = 1e10
	maxFrequencyHz  = 1e12
)

type Descriptor struct {
	Global      Global       `json:"global"`
	Captures    []Capture    `json:"captures"`
	Annotations []Annotation `json:"annotations,omitempty"`
}

type Global struct {
	Datatype     string   `json:"core:datatype"`
	SampleRate   float64  `json:"core:sample_rate,omitempty"`
	Version      string   `json:"core:version"`
	Hardware     string   `json:"core:hw,omitempty"`
	Author       string   `json:"core:author,omitempty"`
	Description  string   `json:"core:description,omitempty"`
	Recorder     string   `json:"core:recorder,omitempty"`
	SHA512       string   `json:"core:sha512,omitempty"`
	Dataset      string   `json:"core:dataset,omitempty"`
	Extensions   []string `json:"core:extensions,omitempty"`
	AgentID      string   `json:"lyrahcle:agent_id,omitempty"`
	CaptureID    string   `json:"lyrahcle:capture_id,omitempty"`
	EvidenceKind string   `json:"lyrahcle:evidence_kind,omitempty"`
}

type Capture struct {
	SampleStart uint64  `json:"core:sample_start"`
	Frequency   float64 `json:"core:frequency,omitempty"`
	Datetime    string  `json:"core:datetime,omitempty"`
	GlobalIndex uint64  `json:"core:global_index,omitempty"`
}

type Annotation struct {
	SampleStart   uint64  `json:"core:sample_start"`
	SampleCount   uint64  `json:"core:sample_count,omitempty"`
	FreqLowerEdge float64 `json:"core:freq_lower_edge,omitempty"`
	FreqUpperEdge float64 `json:"core:freq_upper_edge,omitempty"`
	Label         string  `json:"core:label,omitempty"`
	Description   string  `json:"core:description,omitempty"`
	Generator     string  `json:"core:generator,omitempty"`
}

func Parse(raw []byte) (Descriptor, error) {
	var descriptor Descriptor
	if err := json.Unmarshal(raw, &descriptor); err != nil {
		return Descriptor{}, fmt.Errorf("sigmf: parse metadata: %w", err)
	}
	if err := descriptor.Validate(); err != nil {
		return Descriptor{}, err
	}
	return descriptor, nil
}

func Marshal(descriptor Descriptor) ([]byte, error) {
	if err := descriptor.Validate(); err != nil {
		return nil, err
	}
	out, err := json.MarshalIndent(descriptor, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("sigmf: marshal metadata: %w", err)
	}
	return out, nil
}

func (d Descriptor) Validate() error {
	if normalizeVersion(d.Global.Version) != Version {
		return fmt.Errorf("sigmf: unsupported metadata version %q, expected %s", d.Global.Version, Version)
	}
	if _, err := BytesPerSample(d.Global.Datatype); err != nil {
		return err
	}
	if !isFinite(d.Global.SampleRate) || d.Global.SampleRate <= 0 || d.Global.SampleRate > maxSampleRateHz {
		return fmt.Errorf("sigmf: sample rate %v is missing or out of range", d.Global.SampleRate)
	}
	if strings.TrimSpace(d.Global.Dataset) != "" {
		return fmt.Errorf("sigmf: non-conforming datasets referenced by core:dataset are not supported")
	}
	if len(d.Captures) == 0 {
		return fmt.Errorf("sigmf: metadata declares no capture segment")
	}
	if len(d.Captures) > maxCaptures {
		return fmt.Errorf("sigmf: metadata declares %d capture segments, above the %d supported", len(d.Captures), maxCaptures)
	}
	if len(d.Annotations) > maxAnnotations {
		return fmt.Errorf("sigmf: metadata declares %d annotations, above the %d supported", len(d.Annotations), maxAnnotations)
	}
	if d.Captures[0].SampleStart != 0 {
		return fmt.Errorf("sigmf: first capture segment must start at sample 0")
	}
	for index, capture := range d.Captures {
		if index > 0 && capture.SampleStart <= d.Captures[index-1].SampleStart {
			return fmt.Errorf("sigmf: capture segments must be ordered by increasing sample_start")
		}
		if !isFinite(capture.Frequency) || capture.Frequency < 0 || capture.Frequency > maxFrequencyHz {
			return fmt.Errorf("sigmf: capture segment %d declares an out-of-range centre frequency", index)
		}
		if strings.TrimSpace(capture.Datetime) != "" {
			if _, err := time.Parse(time.RFC3339, capture.Datetime); err != nil {
				return fmt.Errorf("sigmf: capture segment %d has an invalid RFC3339 datetime", index)
			}
		}
	}
	for index, annotation := range d.Annotations {
		if !isFinite(annotation.FreqLowerEdge) || !isFinite(annotation.FreqUpperEdge) {
			return fmt.Errorf("sigmf: annotation %d declares non-finite frequency edges", index)
		}
		if annotation.FreqUpperEdge != 0 && annotation.FreqLowerEdge > annotation.FreqUpperEdge {
			return fmt.Errorf("sigmf: annotation %d has an inverted frequency range", index)
		}
	}
	return nil
}

func (d Descriptor) CenterFrequencyHz() float64 {
	if len(d.Captures) == 0 {
		return 0
	}
	return d.Captures[0].Frequency
}

func (d Descriptor) StartTime() (time.Time, bool) {
	for _, capture := range d.Captures {
		if strings.TrimSpace(capture.Datetime) == "" {
			continue
		}
		parsed, err := time.Parse(time.RFC3339, capture.Datetime)
		if err != nil {
			continue
		}
		return parsed.UTC(), true
	}
	return time.Time{}, false
}

func (d Descriptor) SegmentBoundaries(sampleCount int) error {
	if sampleCount < 0 {
		return fmt.Errorf("sigmf: negative sample count")
	}
	for index, capture := range d.Captures {
		if capture.SampleStart > uint64(sampleCount) {
			return fmt.Errorf("sigmf: capture segment %d starts at sample %d beyond the %d samples present", index, capture.SampleStart, sampleCount)
		}
	}
	return nil
}

func BytesPerSample(datatype string) (int, error) {
	switch strings.ToLower(strings.TrimSpace(datatype)) {
	case DatatypeCF32LE:
		return 8, nil
	case DatatypeCI16LE:
		return 4, nil
	case DatatypeCI8, DatatypeCU8:
		return 2, nil
	case "":
		return 0, fmt.Errorf("sigmf: metadata declares no core:datatype")
	default:
		return 0, fmt.Errorf("sigmf: unsupported datatype %q, expected one of cf32_le, ci16_le, ci8, cu8", datatype)
	}
}

func DecodeSamples(datatype string, raw []byte) ([]complex64, error) {
	width, err := BytesPerSample(datatype)
	if err != nil {
		return nil, err
	}
	if len(raw)%width != 0 {
		return nil, fmt.Errorf("sigmf: data file size %d is not a multiple of %d bytes for datatype %s", len(raw), width, datatype)
	}
	count := len(raw) / width
	samples := make([]complex64, count)
	switch strings.ToLower(strings.TrimSpace(datatype)) {
	case DatatypeCF32LE:
		for i := range samples {
			offset := i * 8
			re := math.Float32frombits(binary.LittleEndian.Uint32(raw[offset:]))
			im := math.Float32frombits(binary.LittleEndian.Uint32(raw[offset+4:]))
			if isNonFinite32(re) || isNonFinite32(im) {
				return nil, fmt.Errorf("sigmf: sample %d is not finite", i)
			}
			samples[i] = complex(re, im)
		}
	case DatatypeCI16LE:
		for i := range samples {
			offset := i * 4
			re := int16(binary.LittleEndian.Uint16(raw[offset:]))
			im := int16(binary.LittleEndian.Uint16(raw[offset+2:]))
			samples[i] = complex(float32(re)/32768, float32(im)/32768)
		}
	case DatatypeCI8:
		for i := range samples {
			offset := i * 2
			samples[i] = complex(float32(int8(raw[offset]))/128, float32(int8(raw[offset+1]))/128)
		}
	case DatatypeCU8:
		for i := range samples {
			offset := i * 2
			samples[i] = complex((float32(raw[offset])-127.5)/127.5, (float32(raw[offset+1])-127.5)/127.5)
		}
	}
	return samples, nil
}

func EncodeSamples(datatype string, samples []complex64) ([]byte, error) {
	if strings.ToLower(strings.TrimSpace(datatype)) != DatatypeCF32LE {
		return nil, fmt.Errorf("sigmf: recordings are written as %s only, not %q", DatatypeCF32LE, datatype)
	}
	out := make([]byte, 0, len(samples)*8)
	for index, sample := range samples {
		re, im := real(sample), imag(sample)
		if isNonFinite32(re) || isNonFinite32(im) {
			return nil, fmt.Errorf("sigmf: sample %d is not finite", index)
		}
		out = binary.LittleEndian.AppendUint32(out, math.Float32bits(re))
		out = binary.LittleEndian.AppendUint32(out, math.Float32bits(im))
	}
	return out, nil
}

func SortAnnotations(annotations []Annotation) []Annotation {
	out := make([]Annotation, len(annotations))
	copy(out, annotations)
	sort.SliceStable(out, func(i, j int) bool { return out[i].SampleStart < out[j].SampleStart })
	return out
}

func normalizeVersion(version string) string {
	return strings.TrimPrefix(strings.TrimSpace(version), "v")
}

func isFinite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

func isNonFinite32(value float32) bool {
	converted := float64(value)
	return math.IsNaN(converted) || math.IsInf(converted, 0)
}
