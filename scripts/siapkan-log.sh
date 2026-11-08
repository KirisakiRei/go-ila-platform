#!/usr/bin/env bash
# Menyiapkan folder log di host /var/log/ila.
#
# Kenapa perlu skrip ini: backend-api dan runner berjalan sebagai user `app`
# (UID 1001) di dalam container, sementara folder yang di-mount dari host
# dimiliki root. Tanpa izin yang benar, container gagal membuat berkas log dan
# kegagalan itu hanya muncul sebagai error tulis saat runtime.
#
# Aman dijalankan berulang. Jalankan sekali sebelum `docker compose up`:
#   sudo ./scripts/siapkan-log.sh

set -euo pipefail

FOLDER="${LOG_DIR_HOST:-/var/log/ila}"
UID_APP=1001
GID_APP=1001

if [ "$(id -u)" -ne 0 ]; then
	echo "Skrip ini perlu root: sudo $0" >&2
	exit 1
fi

mkdir -p "$FOLDER"

# Folder dipakai bersama backend-api, runner, dan Caddy:
# - setgid agar berkas baru mewarisi grup yang sama
# - 2775 agar grup bisa menulis, orang lain hanya membaca
chown -R "$UID_APP:$GID_APP" "$FOLDER"
chmod 2775 "$FOLDER"

echo "Folder log siap: $FOLDER (pemilik $UID_APP:$GID_APP, mode 2775)"
echo "Isi:"
ls -la "$FOLDER" | head -5
