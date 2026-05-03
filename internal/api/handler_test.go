package api

import (
    "bytes"
    "net/http"
    "net/http/httptest"
    "strings"
    "testing"

    "vekta/internal/model"
)

func TestChatStream_NoFlusher(t *testing.T) {
    h := &Handler{Model: nil, Cache: nil}
    w := &nonFlusherWriter{header: make(http.Header)}
    r := httptest.NewRequest("POST", "/v1/chat/stream", strings.NewReader(`{}`))

    h.ChatStream(w, r)

    if w.statusCode != http.StatusInternalServerError {
        t.Errorf("expected 500 for non-flusher transport, got %d", w.statusCode)
    }
}

func TestChatStream_InvalidBody(t *testing.T) {
    h := &Handler{Model: &model.OllamaClient{}, Cache: nil}
    w := httptest.NewRecorder()
    r := httptest.NewRequest("POST", "/v1/chat/stream",
        bytes.NewReader([]byte(`not valid json`)))

    h.ChatStream(w, r)

    if w.Code != http.StatusBadRequest {
        t.Errorf("expected 400 for invalid body, got %d", w.Code)
    }
}

// nonFlusherWriter adalah writer yang TIDAK implement http.Flusher
// dipakai untuk test guard flusher di handler
type nonFlusherWriter struct {
    header     http.Header
    body       bytes.Buffer
    statusCode int
}

func (w *nonFlusherWriter) Header() http.Header         { return w.header }
func (w *nonFlusherWriter) Write(b []byte) (int, error) { return w.body.Write(b) }
func (w *nonFlusherWriter) WriteHeader(code int)        { w.statusCode = code }
