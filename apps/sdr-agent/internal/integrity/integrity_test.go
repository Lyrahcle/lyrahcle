package integrity

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMeasureDetectsCriticalFileChange(t *testing.T) {
	d := t.TempDir()
	p := filepath.Join(d, "agent.conf")
	if err := os.WriteFile(p, []byte("a=1\n"), 0600); err != nil {
		t.Fatal(err)
	}
	a, err := Measure([]string{p})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("a=2\n"), 0600); err != nil {
		t.Fatal(err)
	}
	b, err := Measure([]string{p})
	if err != nil {
		t.Fatal(err)
	}
	if a.ManifestSHA256 == b.ManifestSHA256 {
		t.Fatal("manifest did not change")
	}
	if a.FileCount < 1 {
		t.Fatal("expected measured file")
	}
}
