#!/usr/bin/env bash
# scripts/memory-tune.sh
# Tuning environment variable untuk Vekta di perangkat low-spec.
# Source file ini sebelum jalankan server: source scripts/memory-tune.sh

# ── Go runtime ──────────────────────────────────────────────────────
# GOGC=50: GC lebih agresif — buang garbage lebih cepat, tukar dengan CPU
export GOGC=50

# GOMEMLIMIT: enforce batas memori di level runtime Go (Go 1.19+)
# Sesuaikan dengan RAM tersedia. Default: 1GB untuk Termux J2 Pro
export GOMEMLIMIT="${GOMEMLIMIT:-1073741824}"  # 1GB dalam bytes

# GOMAXPROCS=1: single core — cocok untuk Termux/low-spec
# Hilangkan line ini untuk device dengan >2 core
export GOMAXPROCS=1

# ── Ollama ──────────────────────────────────────────────────────────
# Batasi thread Ollama agar tidak rebutan CPU dengan Go runtime
export OLLAMA_NUM_PARALLEL=1
export OLLAMA_MAX_LOADED_MODELS=1

# ── SQLite ──────────────────────────────────────────────────────────
# Cache SQLite path — pastikan ada foldernya
export SQLITE_PATH="${SQLITE_PATH:-./cache/chat.db}"
mkdir -p "$(dirname "$SQLITE_PATH")"

# ── Summary ─────────────────────────────────────────────────────────
echo "[vekta/tune] Memory settings applied:"
echo "  GOGC=$GOGC"
echo "  GOMEMLIMIT=$(( GOMEMLIMIT / 1024 / 1024 ))MB"
echo "  GOMAXPROCS=$GOMAXPROCS"
echo "  OLLAMA_NUM_PARALLEL=$OLLAMA_NUM_PARALLEL"
echo ""
echo "[vekta/tune] Jalankan server:"
echo "  ./vekta"
