package celestrak

import (
	"strings"
	"testing"
)

func TestParseTLE(t *testing.T) {
	input := "ISS (ZARYA)             \r\n" +
		"1 25544U 98067A   26239.14379413  .00008717  00000+0  16243-3 0  9995\r\n" +
		"2 25544  51.6326 308.8886 0007697  87.8986 272.2885 15.49649978582754\r\n" +
		"NOAA 19                 \r\n" +
		"1 33591U 09005A   26239.50000000  .00000100  00000-0  10000-3 0  9990\r\n" +
		"2 33591  99.1234 100.0000 0013000  90.0000 270.0000 14.12345678123456\r\n"

	entries, err := parseTLE(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parseTLE returned error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2", len(entries))
	}

	if entries[0].Name != "ISS (ZARYA)" {
		t.Errorf("entries[0].Name = %q, want %q", entries[0].Name, "ISS (ZARYA)")
	}
	if entries[0].NoradID != 25544 {
		t.Errorf("entries[0].NoradID = %d, want 25544", entries[0].NoradID)
	}
	if entries[1].NoradID != 33591 {
		t.Errorf("entries[1].NoradID = %d, want 33591", entries[1].NoradID)
	}
}

func TestParseTLEIgnoresMalformedTriplets(t *testing.T) {
	input := "GARBAGE ENTRY\r\nnot a line1\r\nnot a line2\r\n"
	entries, err := parseTLE(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parseTLE returned error: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("got %d entries, want 0 for malformed input", len(entries))
	}
}

func TestParseTLEEmptyInput(t *testing.T) {
	entries, err := parseTLE(strings.NewReader(""))
	if err != nil {
		t.Fatalf("parseTLE returned error: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("got %d entries, want 0 for empty input", len(entries))
	}
}
