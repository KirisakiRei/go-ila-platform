#!/usr/bin/env bash
# Backup harian PostgreSQL ILA.
#
# Data kredensial broker terenkripsi dengan MASTER_KEY, jadi backup database
# SAJA tidak cukup untuk memulihkan. MASTER_KEY sengaja TIDAK disalin di sini:
# simpan terpisah (password manager atau media offline).
set -euo pipefail

: "${DB_DSN:?DB_DSN wajib diisi}"
BACKUP_DIR="${BACKUP_DIR:-/var/backups/ila}"
RETENTION_DAYS="${RETENTION_DAYS:-14}"

mkdir -p "$BACKUP_DIR"
stamp="$(date -u +%Y%m%dT%H%M%SZ)"
target="$BACKUP_DIR/ila-$stamp.dump"

echo "[backup] menulis $target"
pg_dump --dbname="$DB_DSN" --format=custom --no-owner --no-privileges --file="$target"

# Verifikasi isi arsip: backup yang tidak bisa dibaca dianggap tidak ada.
echo "[backup] memverifikasi arsip"
pg_restore --list "$target" > /dev/null

checksum="$(sha256sum "$target" | awk '{print $1}')"
echo "$checksum  $(basename "$target")" >> "$BACKUP_DIR/SHA256SUMS"

# Hapus arsip lama agar disk tidak penuh tanpa disadari.
find "$BACKUP_DIR" -name 'ila-*.dump' -type f -mtime "+$RETENTION_DAYS" -delete

echo "[backup] selesai: $target ($checksum)"
