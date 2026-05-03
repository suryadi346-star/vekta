#!/usr/bin/env bash
# scripts/ollama-setup.sh
# Setup Ollama dan pull model default untuk Vekta.
# Kompatibel dengan Linux x86_64 dan Termux (Android arm64).

set -euo pipefail

MODEL="${1:-qwen2.5:0.5b}"
OLLAMA_HOST="${OLLAMA_HOST:-127.0.0.1:11434}"

log() { echo "[vekta/setup] $*"; }
err() { echo "[vekta/setup] ERROR: $*" >&2; exit 1; }

# ── Deteksi environment ──────────────────────────────────────────────
if command -v termux-info &>/dev/null || [ -d "/data/data/com.termux" ]; then
  ENV="termux"
else
  ENV="linux"
fi
log "Environment: $ENV"

# ── Install Ollama ───────────────────────────────────────────────────
if command -v ollama &>/dev/null; then
  log "Ollama sudah terinstall: $(ollama --version)"
else
  if [ "$ENV" = "termux" ]; then
    log "Install Ollama di Termux..."
    pkg install -y ollama 2>/dev/null || {
      log "pkg install gagal, coba manual download..."
      curl -fsSL https://ollama.com/install.sh | sh
    }
  else
    log "Install Ollama via official installer..."
    curl -fsSL https://ollama.com/install.sh | sh
  fi
fi

# ── Start Ollama server ──────────────────────────────────────────────
if ! curl -sf "http://$OLLAMA_HOST/" &>/dev/null; then
  log "Starting Ollama server..."
  OLLAMA_HOST="$OLLAMA_HOST" ollama serve &
  OLLAMA_PID=$!
  sleep 3
  if ! kill -0 "$OLLAMA_PID" 2>/dev/null; then
    err "Ollama server gagal start"
  fi
  log "Ollama berjalan di $OLLAMA_HOST (PID: $OLLAMA_PID)"
else
  log "Ollama sudah berjalan di $OLLAMA_HOST"
fi

# ── Pull model ───────────────────────────────────────────────────────
log "Pull model: $MODEL"
ollama pull "$MODEL"
log "Model $MODEL siap."

# ── Verifikasi ───────────────────────────────────────────────────────
log "Model list:"
ollama list

log ""
log "Setup selesai. Jalankan Vekta dengan:"
log "  make run"
log "  atau: OLLAMA_HOST=$OLLAMA_HOST ./vekta"
