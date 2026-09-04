package forensics

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/mlko01/lyrahcle/apps/sdr-agent/internal/sigmf"
	"github.com/mlko01/lyrahcle/apps/sdr-agent/internal/streaming"
)

func SaveSigMF(dir, agentID string, data *streaming.IQData) (dataPath, metaPath string, err error) {
	if data == nil || len(data.Samples) == 0 {
		return "", "", fmt.Errorf("forensics: capture has no samples")
	}
	id := fmt.Sprintf("%s_%d", sanitize(agentID), data.Timestamp.UnixNano())
	descriptor := sigmf.NewDescriptor(agentID, id, data.Frequency, data.SampleRate, data.Timestamp, data.Metadata["device"])
	descriptor.Global.Description = "Lyrahcle agent IQ capture"
	dataPath, metaPath, err = sigmf.Write(dir, id, descriptor, data.Samples)
	if err != nil {
		return "", "", err
	}
	return dataPath, metaPath, nil
}

func LoadSigMFWithin(baseDir, name string) (Metadata, []complex64, error) {
	if !sigmf.IsMetadataPath(name) {
		return Metadata{}, nil, fmt.Errorf("forensics: SigMF captures are referenced by their %s file", sigmf.MetaExtension)
	}
	metaPath, err := ResolveWithin(baseDir, name)
	if err != nil {
		return Metadata{}, nil, err
	}
	dataPath, err := sigmf.DataPathFor(metaPath)
	if err != nil {
		return Metadata{}, nil, err
	}
	if _, err := ResolveWithin(baseDir, strings.TrimSuffix(name, sigmf.MetaExtension)+sigmf.DataExtension); err != nil {
		return Metadata{}, nil, err
	}
	recording, err := sigmf.Read(metaPath)
	if err != nil {
		return Metadata{}, nil, err
	}
	meta := metadataFromDescriptor(recording, dataPath)
	return meta, recording.Samples, nil
}

func LoadAnyWithin(baseDir, name string) (Metadata, []complex64, error) {
	if sigmf.IsMetadataPath(name) {
		return LoadSigMFWithin(baseDir, name)
	}
	return LoadWithin(baseDir, name)
}

func metadataFromDescriptor(recording sigmf.Recording, dataPath string) Metadata {
	descriptor := recording.Descriptor
	frequency := descriptor.CenterFrequencyHz()
	if frequency < 0 {
		frequency = 0
	}
	extra := map[string]string{
		"source":         "sigmf",
		"datatype":       descriptor.Global.Datatype,
		"integrity":      recording.IntegrityState,
		"timestamp_from": "metadata",
	}
	if strings.TrimSpace(descriptor.Global.Hardware) != "" {
		extra["hardware"] = descriptor.Global.Hardware
	}
	if strings.TrimSpace(descriptor.Global.Author) != "" {
		extra["author"] = descriptor.Global.Author
	}
	if strings.TrimSpace(descriptor.Global.EvidenceKind) != "" {
		extra["evidence_kind"] = descriptor.Global.EvidenceKind
	} else {
		extra["evidence_kind"] = "external_recording"
	}

	capturedAt, ok := descriptor.StartTime()
	if !ok {
		capturedAt = fileModificationTime(dataPath)
		extra["timestamp_from"] = "file_modification_time"
	}

	agentID := descriptor.Global.AgentID
	if strings.TrimSpace(agentID) == "" {
		agentID = "sigmf-import"
	}

	return Metadata{
		ID:           descriptor.Global.CaptureID,
		AgentID:      agentID,
		FrequencyHz:  uint64(frequency),
		SampleRateHz: uint64(descriptor.Global.SampleRate),
		CapturedAt:   capturedAt,
		SampleCount:  len(recording.Samples),
		Format:       descriptor.Global.Datatype,
		Extra:        extra,
	}
}

func fileModificationTime(path string) time.Time {
	info, err := os.Stat(path)
	if err != nil {
		return time.Now().UTC()
	}
	return info.ModTime().UTC()
}
