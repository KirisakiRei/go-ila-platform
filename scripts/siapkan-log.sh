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

# Sertifikat mTLS ikut disiapkan di sini karena masalahnya sama: berkas dibuat
# oleh user sudo (UID 1000) dengan mode 600, lalu container UID 1001 tidak bisa
# membacanya dan penerima log berputar-putar restart.
CERTS="$(dirname "$0")/../deploy/certs"
if [ -d "$CERTS" ]; then
	chown root:$GID_APP "$CERTS"/*.key
	chmod 640 "$CERTS"/*.key
	echo "Kunci sertifikat siap: $CERTS (pemilik root:$GID_APP, mode 640)"
else
	echo "Lewati sertifikat: $CERTS belum ada" >&2
fi

echo "Isi:"
ls -la "$FOLDER" | head -5
