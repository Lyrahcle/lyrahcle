package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestStatusCapturingResponseWriterV144(t *testing.T) {
	recorder := httptest.NewRecorder()
	captured := &statusCapturingResponseWriter{ResponseWriter: recorder}
	http.Error(captured, "boom", http.StatusInternalServerError)
	if captured.status != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", captured.status, http.StatusInternalServerError)
	}
	if captured.bytes == 0 {
		t.Fatal("expected response byte count")
	}
}
