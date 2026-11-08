#!/usr/bin/env bash
# Pemulihan backup ke database KOSONG.
#
# Aturan pakai:
#   1. Buat database baru (jangan memulihkan di atas database hidup).
#   2. MASTER_KEY pada environment HARUS kunci yang sama dengan saat backup.
#      Bila berbeda, akun broker akan terbaca tetapi kredensialnya tidak.
set -euo pipefail

: "${TARGET_DSN:?TARGET_DSN wajib diisi (database kosong)}"
: "${DUMP_FILE:?DUMP_FILE wajib diisi (berkas .dump)}"

if [[ ! -f "$DUMP_FILE" ]]; then
  echo "[restore] berkas tidak ditemukan: $DUMP_FILE" >&2
  exit 1
fi

echo "[restore] memeriksa arsip"
pg_restore --list "$DUMP_FILE" > /dev/null

echo "[restore] memulihkan ke $TARGET_DSN"
pg_restore --dbname="$TARGET_DSN" --no-owner --no-privileges --clean --if-exists "$DUMP_FILE"

echo "[restore] selesai. Langkah lanjutan:"
echo "  1) jalankan migrasi: go run ./cmd/migrate"
echo "  2) jalankan backend-api satu kali untuk memastikan tabel terbaca"
echo "  3) periksa jumlah baris tabel: user, broker_account, position, invoice"
echo "  4) uji login satu akun uji dan buka satu akun broker (kredensial harus terbaca)"
