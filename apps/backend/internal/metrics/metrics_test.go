package metrics

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMiddlewarePreservesFlusher(t *testing.T) {
	handler := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("middleware removed http.Flusher support")
		}
		w.WriteHeader(http.StatusOK)
		flusher.Flush()
	}))

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/stream", nil))

	if !recorder.Flushed {
		t.Fatal("middleware did not forward Flush")
	}
}

type basicResponseWriter struct {
	header http.Header
	status int
}

func (w *basicResponseWriter) Header() http.Header {
	return w.header
}

func (w *basicResponseWriter) Write(body []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return len(body), nil
}

func (w *basicResponseWriter) WriteHeader(status int) {
	w.status = status
}

func TestMiddlewareDoesNotInventFlusherSupport(t *testing.T) {
	handler := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := w.(http.Flusher); ok {
			t.Fatal("middleware exposed http.Flusher without underlying support")
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	writer := &basicResponseWriter{header: make(http.Header)}
	handler.ServeHTTP(writer, httptest.NewRequest(http.MethodGet, "/health", nil))

	if writer.status != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", writer.status, http.StatusNoContent)
	}
}
