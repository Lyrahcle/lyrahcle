package main

import (
	"os"
	"sort"
	"strings"
	"testing"
)

func readServerProductionSource(t *testing.T) string {
	t.Helper()

	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)

	var source strings.Builder
	for _, name := range names {
		data, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		source.WriteString("\n// --- ")
		source.WriteString(name)
		source.WriteString(" ---\n")
		source.Write(data)
		source.WriteByte('\n')
	}
	return source.String()
}
