#!/usr/bin/env bash
# Memeriksa bahwa log tidak memuat rahasia.
#
# Dijalankan sebelum promosi ke produksi atau saat menambah log baru.
# Keluaran bukan nol berarti ada nilai yang harus disensor lebih dulu.
set -euo pipefail

: "${LOG_FILE:?LOG_FILE wajib diisi}"

patterns=(
  "password"
  "secret"
  "access_token"
  "Authorization"
  "MASTER_KEY"
  "Bearer "
)

found=0
for pattern in "${patterns[@]}"; do
  # Kata kunci boleh muncul sebagai nama kolom, tetapi nilainya tidak.
  if grep -Eic "$pattern[=\": ]+[A-Za-z0-9._-]{12,}" "$LOG_FILE" > /dev/null; then
    echo "[audit-log] nilai rahasia terdeteksi untuk pola: $pattern" >&2
    found=1
  fi
done

if [[ "$found" -eq 1 ]]; then
  exit 1
fi
echo "[audit-log] bersih"
