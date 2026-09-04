package sigmf

import (
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	MaxMetadataBytes = 8 << 20
	MaxDataBytes     = 512 << 20
)

type Recording struct {
	Descriptor     Descriptor
	Samples        []complex64
	MetadataPath   string
	DataPath       string
	IntegrityState string
}

func DataPathFor(metadataPath string) (string, error) {
	if !strings.HasSuffix(metadataPath, MetaExtension) {
		return "", fmt.Errorf("sigmf: metadata path must end in %s", MetaExtension)
	}
	return strings.TrimSuffix(metadataPath, MetaExtension) + DataExtension, nil
}

func IsMetadataPath(path string) bool {
	return strings.HasSuffix(path, MetaExtension)
}

func Read(metadataPath string) (Recording, error) {
	dataPath, err := DataPathFor(metadataPath)
	if err != nil {
		return Recording{}, err
	}
	metaBytes, err := readRegularFile(metadataPath, MaxMetadataBytes)
	if err != nil {
		return Recording{}, fmt.Errorf("sigmf: read metadata: %w", err)
	}
	descriptor, err := Parse(metaBytes)
	if err != nil {
		return Recording{}, err
	}
	rawData, err := readRegularFile(dataPath, MaxDataBytes)
	if err != nil {
		return Recording{}, fmt.Errorf("sigmf: read data: %w", err)
	}
	integrity := "not_declared"
	if declared := strings.TrimSpace(descriptor.Global.SHA512); declared != "" {
		sum := sha512.Sum512(rawData)
		if !strings.EqualFold(declared, hex.EncodeToString(sum[:])) {
			return Recording{}, fmt.Errorf("sigmf: data file does not match the declared core:sha512 checksum")
		}
		integrity = "verified"
	}
	samples, err := DecodeSamples(descriptor.Global.Datatype, rawData)
	if err != nil {
		return Recording{}, err
	}
	if err := descriptor.SegmentBoundaries(len(samples)); err != nil {
		return Recording{}, err
	}
	return Recording{
		Descriptor:     descriptor,
		Samples:        samples,
		MetadataPath:   metadataPath,
		DataPath:       dataPath,
		IntegrityState: integrity,
	}, nil
}

func Write(dir, name string, descriptor Descriptor, samples []complex64) (dataPath, metadataPath string, err error) {
	if len(samples) == 0 {
		return "", "", fmt.Errorf("sigmf: recording has no samples")
	}
	cleanName := strings.TrimSpace(name)
	if cleanName == "" || cleanName == "." || cleanName == ".." || strings.ContainsAny(cleanName, `/\`) || strings.Contains(cleanName, "..") {
		return "", "", fmt.Errorf("sigmf: invalid recording name")
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", "", fmt.Errorf("sigmf: create dir: %w", err)
	}
	descriptor.Global.Version = Version
	descriptor.Global.Datatype = DatatypeCF32LE
	if strings.TrimSpace(descriptor.Global.Recorder) == "" {
		descriptor.Global.Recorder = RecorderName
	}
	descriptor.Annotations = SortAnnotations(descriptor.Annotations)

	rawData, err := EncodeSamples(DatatypeCF32LE, samples)
	if err != nil {
		return "", "", err
	}
	sum := sha512.Sum512(rawData)
	descriptor.Global.SHA512 = hex.EncodeToString(sum[:])

	metaBytes, err := Marshal(descriptor)
	if err != nil {
		return "", "", err
	}

	dataPath = filepath.Join(dir, cleanName+DataExtension)
	metadataPath = filepath.Join(dir, cleanName+MetaExtension)
	if err := os.WriteFile(dataPath, rawData, 0o600); err != nil {
		return "", "", fmt.Errorf("sigmf: write data: %w", err)
	}
	if err := os.WriteFile(metadataPath, metaBytes, 0o600); err != nil {
		os.Remove(dataPath)
		return "", "", fmt.Errorf("sigmf: write metadata: %w", err)
	}
	return dataPath, metadataPath, nil
}

func NewDescriptor(agentID, captureID string, frequencyHz, sampleRateHz uint64, capturedAt time.Time, hardware string) Descriptor {
	return Descriptor{
		Global: Global{
			Datatype:     DatatypeCF32LE,
			SampleRate:   float64(sampleRateHz),
			Version:      Version,
			Hardware:     hardware,
			Recorder:     RecorderName,
			AgentID:      agentID,
			CaptureID:    captureID,
			EvidenceKind: "agent_measured",
		},
		Captures: []Capture{{
			SampleStart: 0,
			Frequency:   float64(frequencyHz),
			Datetime:    capturedAt.UTC().Format(time.RFC3339),
		}},
	}
}

func readRegularFile(path string, limit int64) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("symlink files are not allowed")
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("path is not a regular file")
	}
	if info.Size() > limit {
		return nil, fmt.Errorf("file of %d bytes is above the %d byte limit", info.Size(), limit)
	}
	return os.ReadFile(path)
}
