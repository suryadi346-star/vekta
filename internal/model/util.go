package model

import (
	"bytes"
	"io"
)

// jsonReader membungkus []byte sebagai io.Reader untuk HTTP body.
// Dipisah agar ollama.go tetap bersih tanpa bytes.NewReader inline.
func jsonReader(b []byte) io.Reader {
	return bytes.NewReader(b)
}
