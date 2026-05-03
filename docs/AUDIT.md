# Shadow Audit: vekta Vulnerability Report

## Kerentanan yang Ditemukan & Status Fix

| # | File | Masalah | Severity | Status |
|---|------|---------|----------|--------|
| 1 | `handler.go` | `flusher` assertion tanpa guard → nil panic | HIGH | ✅ Fixed |
| 2 | `handler.go` | `errCh` tidak di-handle di select loop → error tenggelam | HIGH | ✅ Fixed |
| 3 | `ollama.go` | Goroutine leak saat ctx cancel di tengah `dec.More()` | HIGH | ✅ Fixed |
| 4 | `ollama.go` | Error dari goroutine dikirim ke `errCh` tapi tidak dibaca | MEDIUM | ✅ Fixed |
| 5 | `app.yaml` | `memory_hard_limit_mb` deklaratif tapi tidak pernah di-enforce | MEDIUM | ✅ Fixed |
| 6 | `cache` | SQLite tanpa `SetMaxOpenConns` → SQLITE_BUSY di concurrent load | MEDIUM | ✅ Fixed |
| 7 | `handler.go` | SSE chunk tidak di-JSON-encode → karakter newline/quote bisa break protokol | LOW | ✅ Fixed |
| 8 | `config` | Tidak ada env override untuk deployment di Termux | LOW | ✅ Fixed |

---

## Fix Summary Per File

### `internal/model/ollama.go`
- Setiap send ke channel sekarang punya `select + ctx.Done()` — goroutine exit bersih
- Error dikirim ke `errCh` dengan pattern non-blocking (`select default`) 
- `doStream` mewariskan ctx dari caller — tidak buat timeout duplikat

### `internal/api/handler.go`
- Guard `flusher` di awal fungsi — return 500 jika transport tidak support SSE
- `select` loop sekarang handle `errCh` channel secara eksplisit
- Chunk di-marshal via `json.Marshal` sebelum dikirim sebagai SSE data

### `internal/cache/sqlite.go`
- `SetMaxOpenConns(1)` — SQLite WAL paling stabil dengan single writer
- Pragma diinline ke DSN — lebih reliable dari PRAGMA post-open
- Lazy delete untuk expired entries — tidak block caller

### `cmd/server/main.go`
- `debug.SetMemoryLimit()` dipanggil dengan nilai dari config — limit sekarang aktif
- `debug.SetGCPercent()` dipanggil via `cfg.System.GOGC`
- Background goroutine untuk cache eviction berkala

### `internal/config/config.go`
- Struct mapped ke YAML field dengan env override
- Validasi minimal sebelum server start

---

## Model Recommendation (Gratis, Ringan)

| Model | RAM Est. | Speed | Cocok Untuk |
|-------|----------|-------|-------------|
| `qwen2.5:0.5b` | ~400MB | Cepat | Default — paling ringan |
| `qwen2.5:1.5b` | ~900MB | Medium | Kalau RAM cukup |
| `phi3:mini` | ~2.3GB | Medium | Reasoning lebih baik |
| `gemma2:2b` | ~1.6GB | Cepat | Balance quality/speed |

**Untuk Termux/J2 Pro: stick ke `qwen2.5:0.5b`.**

---

## Risk/Reward Note

**Kalau skip fix ini:**
- Goroutine leak akan accumulate seiring traffic → OOM crash tanpa warning
- Nil flusher panic akan kill satu goroutine per request yang masuk lewat proxy
- Memory limit tidak aktif → sistem bisa ambil RAM sesukanya sampai Android OOM killer eksekusi prosesnya

**Setelah fix ini:**
- Server bisa jalan stabil di 512MB RAM (target J2 Pro Termux)
- Graceful disconnect tanpa resource leak
- SQLite tidak deadlock di concurrent request
