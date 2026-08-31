package integrity

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Report struct {
	ManifestSHA256 string
	MachineIDHash  string
	BootID         string
	FileCount      int
}

func Measure(paths []string) (Report, error) {
	exe, err := os.Executable()
	if err == nil && exe != "" {
		paths = append(paths, exe)
	}
	uniq := map[string]struct{}{}
	for _, raw := range paths {
		p := strings.TrimSpace(raw)
		if p == "" {
			continue
		}
		abs, err := filepath.Abs(p)
		if err != nil {
			return Report{}, err
		}
		uniq[filepath.Clean(abs)] = struct{}{}
	}
	ordered := make([]string, 0, len(uniq))
	for p := range uniq {
		ordered = append(ordered, p)
	}
	sort.Strings(ordered)
	manifest := sha256.New()
	count := 0
	for _, p := range ordered {
		f, err := os.Open(p)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return Report{}, fmt.Errorf("integrity: open %s: %w", p, err)
		}
		h := sha256.New()
		if _, err := io.Copy(h, f); err != nil {
			f.Close()
			return Report{}, err
		}
		f.Close()
		fmt.Fprintf(manifest, "%s\x00%s\n", p, hex.EncodeToString(h.Sum(nil)))
		count++
	}
	return Report{ManifestSHA256: hex.EncodeToString(manifest.Sum(nil)), MachineIDHash: hashOptionalFile("/etc/machine-id"), BootID: readLine("/proc/sys/kernel/random/boot_id"), FileCount: count}, nil
}

func PathsFromCSV(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return []string{"/etc/os-release", "/proc/cmdline"}
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if v := strings.TrimSpace(p); v != "" {
			out = append(out, v)
		}
	}
	return out
}
func hashOptionalFile(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	s := sha256.Sum256(bytesTrimSpace(b))
	return hex.EncodeToString(s[:])
}
func readLine(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	if sc.Scan() {
		return strings.TrimSpace(sc.Text())
	}
	return ""
}
func bytesTrimSpace(b []byte) []byte { return []byte(strings.TrimSpace(string(b))) }
