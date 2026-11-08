#!/usr/bin/env bash
# Backup harian PostgreSQL ILA.
#
# Data kredensial broker terenkripsi dengan MASTER_KEY, jadi backup database
# SAJA tidak cukup untuk memulihkan. MASTER_KEY sengaja TIDAK disalin di sini:
# simpan terpisah (password manager atau media offline).
#
# Retensi mengikuti rencana: 7 salinan harian + 4 salinan mingguan.
# Bila REMOTE_DIR diisi (direktori hasil mount rclone/rsync ke penyimpanan
# luar), salinan harian juga dikirim ke sana. Backup yang hanya ada di VPS
# yang sama tidak melindungi dari kerusakan VPS itu sendiri.
set -euo pipefail

: "${DB_DSN:?DB_DSN wajib diisi}"
BACKUP_DIR="${BACKUP_DIR:-/var/backups/ila}"
DAILY_KEEP="${DAILY_KEEP:-7}"
WEEKLY_KEEP="${WEEKLY_KEEP:-4}"
REMOTE_DIR="${REMOTE_DIR:-}"

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

# Salinan mingguan dibuat pada hari Minggu (UTC) supaya jadwal tidak bergeser.
if [[ "$(date -u +%u)" == "7" ]]; then
  weekly="$BACKUP_DIR/weekly/ila-$stamp.dump"
  mkdir -p "$BACKUP_DIR/weekly"
  cp "$target" "$weekly"
  echo "[backup] salinan mingguan: $weekly"
fi

if [[ -n "$REMOTE_DIR" ]]; then
  mkdir -p "$REMOTE_DIR"
  echo "[backup] mengirim salinan ke $REMOTE_DIR"
  cp "$target" "$REMOTE_DIR/"
  echo "$checksum  $(basename "$target")" >> "$REMOTE_DIR/SHA256SUMS"
fi

# Hapus arsip lama agar disk tidak penuh tanpa disadari.
find "$BACKUP_DIR" -maxdepth 1 -name 'ila-*.dump' -type f -mtime "+$DAILY_KEEP" -delete
find "$BACKUP_DIR/weekly" -name 'ila-*.dump' -type f -mtime "+$((WEEKLY_KEEP * 7))" -delete 2>/dev/null || true

# --- Log layanan ---------------------------------------------------------
# Log harian ikut diarsipkan: berkas di host terlindungi dari container yang
# dibuat ulang, tapi tidak dari disk VPS yang rusak. Arsip mingguan dipakai
# saat menelusuri insiden lama setelah berkas aslinya melewati masa simpan.
LOG_SOURCE="${LOG_DIR_HOST:-/var/log/ila}"
if [[ -d "$LOG_SOURCE" ]]; then
  if [[ "$(date -u +%u)" == "7" ]]; then
    mkdir -p "$BACKUP_DIR/logs"
    arsip="$BACKUP_DIR/logs/ila-log-$stamp.tar.gz"
    tar -czf "$arsip" -C "$(dirname "$LOG_SOURCE")" "$(basename "$LOG_SOURCE")" 2>/dev/null || true
    if [[ -f "$arsip" ]]; then
      echo "[backup] arsip log mingguan: $arsip"
      if [[ -n "$REMOTE_DIR" ]]; then
        cp "$arsip" "$REMOTE_DIR/"
      fi
    fi
  fi
  find "$BACKUP_DIR/logs" -name 'ila-log-*.tar.gz' -type f -mtime "+$((WEEKLY_KEEP * 7))" -delete 2>/dev/null || true
fi

echo "[backup] selesai: $target ($checksum)"
