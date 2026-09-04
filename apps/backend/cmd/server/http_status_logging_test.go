package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestStatusCapturingResponseWriterV144(t *testing.T) {
	recorder := httptest.NewRecorder()
	captured := &statusCapturingResponseWriter{ResponseWriter: recorder, requestID: "req-test"}
	http.Error(captured, "boom", http.StatusInternalServerError)
	if captured.status != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", captured.status, http.StatusInternalServerError)
	}
	if captured.bytes == 0 {
		t.Fatal("expected response byte count")
	}
	if contentType := recorder.Header().Get("Content-Type"); contentType != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", contentType)
	}
	var envelope apiErrorEnvelope
	if err := json.Unmarshal(recorder.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode error envelope: %v", err)
	}
	if envelope.Error.Code != "internal_error" || envelope.Error.Message != "boom" || envelope.Error.RequestID != "req-test" {
		t.Fatalf("unexpected error envelope: %+v", envelope.Error)
	}
}
