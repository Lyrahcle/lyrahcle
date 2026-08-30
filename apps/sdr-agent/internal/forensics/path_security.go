package forensics

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ResolveWithin returns an existing regular file confined to baseDir.
// candidate must be a relative path. Symlinks are rejected so a capture tree
// cannot redirect replay/list operations outside its configured root.
func ResolveWithin(baseDir, candidate string) (string, error) {
	if strings.TrimSpace(baseDir) == "" {
		return "", fmt.Errorf("forensics: base directory is required")
	}
	if strings.TrimSpace(candidate) == "" {
		return "", fmt.Errorf("forensics: capture path is required")
	}
	if filepath.IsAbs(candidate) {
		return "", fmt.Errorf("forensics: absolute capture paths are not allowed")
	}

	cleanCandidate := filepath.Clean(candidate)
	if cleanCandidate == "." || cleanCandidate == ".." || strings.HasPrefix(cleanCandidate, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("forensics: capture path escapes base directory")
	}

	baseAbs, err := filepath.Abs(baseDir)
	if err != nil {
		return "", fmt.Errorf("forensics: resolve base directory: %w", err)
	}
	baseResolved, err := filepath.EvalSymlinks(baseAbs)
	if err != nil {
		return "", fmt.Errorf("forensics: resolve base directory symlinks: %w", err)
	}

	candidateAbs := filepath.Join(baseResolved, cleanCandidate)
	candidateResolved, err := filepath.EvalSymlinks(candidateAbs)
	if err != nil {
		return "", fmt.Errorf("forensics: resolve capture path: %w", err)
	}

	rel, err := filepath.Rel(baseResolved, candidateResolved)
	if err != nil {
		return "", fmt.Errorf("forensics: compare capture path: %w", err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return "", fmt.Errorf("forensics: capture path escapes base directory")
	}

	info, err := os.Lstat(candidateAbs)
	if err != nil {
		return "", fmt.Errorf("forensics: inspect capture path: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("forensics: capture path must not be a symlink")
	}

	resolvedInfo, err := os.Stat(candidateResolved)
	if err != nil {
		return "", fmt.Errorf("forensics: inspect resolved capture path: %w", err)
	}
	if !resolvedInfo.Mode().IsRegular() {
		return "", fmt.Errorf("forensics: capture path is not a regular file")
	}

	return candidateResolved, nil
}

func readRegularFile(path string) ([]byte, error) {
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
	return os.ReadFile(path)
}
