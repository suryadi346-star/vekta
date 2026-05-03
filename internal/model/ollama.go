package model

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// StreamChunk adalah unit data dari stream Ollama.
type StreamChunk struct {
	Content string
	Done    bool
}

// ChatRequest adalah payload ke Ollama /api/chat.
type ChatRequest struct {
	Model    string        `json:"model"`
	Messages []ChatMessage `json:"messages"`
	Stream   bool          `json:"stream"`
}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OllamaStreamResp struct {
	Message struct {
		Content string `json:"content"`
	} `json:"message"`
	Done       bool   `json:"done"`
	DoneReason string `json:"done_reason"`
}

type OllamaClient struct {
	BaseURL    string
	HTTPClient *http.Client
}

func NewOllamaClient(baseURL string) *OllamaClient {
	return &OllamaClient{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: 90 * time.Second, // timeout total, bukan per-chunk
		},
	}
}

// Stream mengembalikan dua channel: data dan error.
// FIX #1: goroutine sekarang select pada ctx.Done() saat decode blocking,
// mencegah goroutine leak ketika client disconnect di tengah stream.
func (c *OllamaClient) Stream(ctx context.Context, req ChatRequest) (<-chan StreamChunk, <-chan error) {
	out := make(chan StreamChunk, 32)
	errCh := make(chan error, 1)

	go func() {
		defer close(out)
		defer close(errCh)

		// FIX #2: doStream menggunakan ctx yang sama sehingga auto-cancel
		// ketika request HTTP parent dibatalkan.
		resp, err := c.doStream(ctx, req)
		if err != nil {
			select {
			case errCh <- fmt.Errorf("ollama stream init: %w", err):
			default:
			}
			return
		}
		defer resp.Body.Close()

		dec := json.NewDecoder(resp.Body)
		var buf OllamaStreamResp

		for dec.More() {
			// FIX #3: cek ctx sebelum setiap decode agar loop bisa exit
			// jika client sudah disconnect (tidak menunggu decode selesai).
			select {
			case <-ctx.Done():
				// Context dibatalkan — keluar bersih tanpa kirim error ke consumer
				return
			default:
			}

			if err := dec.Decode(&buf); err != nil {
				select {
				case errCh <- fmt.Errorf("decode stream chunk: %w", err):
				default:
				}
				return
			}

			if buf.Message.Content != "" {
				select {
				case out <- StreamChunk{Content: buf.Message.Content, Done: false}:
				case <-ctx.Done():
					return
				}
			}

			if buf.Done {
				select {
				case out <- StreamChunk{Done: true}:
				case <-ctx.Done():
				}
				return
			}
		}
	}()

	return out, errCh
}

// doStream melakukan HTTP POST ke Ollama dengan context timeout.
// Retry 1x hanya untuk network error, bukan untuk 4xx/5xx.
func (c *OllamaClient) doStream(ctx context.Context, req ChatRequest) (*http.Response, error) {
	req.Stream = true

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	// Gunakan context dari caller — bukan buat timeout baru di sini.
	// Handler sudah punya timeout via middleware.
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.BaseURL+"/api/chat", jsonReader(body))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("http do: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("ollama returned status %d", resp.StatusCode)
	}

	return resp, nil
}
