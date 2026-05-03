package api

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"vekta/internal/cache"
	"vekta/internal/chat"
	"vekta/internal/model"
)

// zeroTime dipakai untuk disable write deadline pada SSE connections.
var zeroTime time.Time

type Handler struct {
	Model        *model.OllamaClient
	Cache        *cache.DB
	Sessions     *chat.Manager
	DefaultModel string
}

func writeSSE(w http.ResponseWriter, event, data string) {
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, data)
}

func (h *Handler) ChatStream(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Guard: pastikan transport support SSE
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported by this transport", http.StatusInternalServerError)
		slog.Error("SSE flusher not available — check reverse proxy config")
		return
	}

	// Disable write deadline per-request untuk SSE long-lived connection
	rc := http.NewResponseController(w)
	if err := rc.SetWriteDeadline(zeroTime); err != nil {
		slog.Warn("could not disable write deadline for SSE", "err", err)
	}

	// Limit body 64KB — cegah OOM via giant payload
	r.Body = http.MaxBytesReader(w, r.Body, 64*1024)
	var chatReq model.ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&chatReq); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if chatReq.Model == "" {
		chatReq.Model = h.DefaultModel
		if chatReq.Model == "" {
			chatReq.Model = "qwen2.5:0.5b"
		}
	}

	// Session context: ambil session ID dari header, append messages ke history
	sessionID := r.Header.Get("X-Session-Id")
	if sessionID != "" && h.Sessions != nil {
		sess := h.Sessions.Get(sessionID)
		// Merge history session ke request messages
		if len(chatReq.Messages) > 0 {
			for _, m := range chatReq.Messages {
				sess.Add(m.Role, m.Content)
			}
			chatReq.Messages = sess.History()
		}
	}

	// SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	stream, errCh := h.Model.Stream(ctx, chatReq)

	var assistantReply string
	for {
		select {
		case <-ctx.Done():
			slog.Info("client disconnected", "reason", ctx.Err())
			return

		case err, open := <-errCh:
			if !open {
				return
			}
			if err != nil {
				slog.Error("model stream error", "err", err)
				writeSSE(w, "error", fmt.Sprintf(`{"message":%q}`, err.Error()))
				flusher.Flush()
				return
			}

		case chunk, ok := <-stream:
			if !ok {
				// Simpan reply assistant ke session
				if sessionID != "" && h.Sessions != nil && assistantReply != "" {
					h.Sessions.Get(sessionID).Add("assistant", assistantReply)
				}
				writeSSE(w, "done", "{}")
				flusher.Flush()
				return
			}
			if chunk.Done {
				if sessionID != "" && h.Sessions != nil && assistantReply != "" {
					h.Sessions.Get(sessionID).Add("assistant", assistantReply)
				}
				writeSSE(w, "done", "{}")
				flusher.Flush()
				return
			}
			assistantReply += chunk.Content
			encoded, err := json.Marshal(map[string]string{"content": chunk.Content})
			if err != nil {
				slog.Warn("failed to marshal chunk", "err", err)
				continue
			}
			writeSSE(w, "chunk", string(encoded))
			flusher.Flush()
		}
	}
}
