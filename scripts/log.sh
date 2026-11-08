#!/usr/bin/env bash
# Membaca log harian ILA (Engine, Runner, backend-api, Caddy).
#
# Semua layanan menulis JSONL dengan kunci sama: t, lvl, svc, evt, msg, trace.
# Skrip ini hanya pembungkus jq agar path folder tidak perlu dihafal.
#
# Contoh:
#   ./scripts/log.sh                        # hari ini, semua layanan
#   ./scripts/log.sh 2026/11/07             # hari tertentu
#   ./scripts/log.sh 2026/11/07 engine      # satu layanan
#   LOG_LEVEL=ERROR ./scripts/log.sh        # hanya error
#   LOG_CARI=a1b2c3 ./scripts/log.sh        # lacak satu trace

set -euo pipefail

AKAR="${LOG_DIR:-/var/log/ila}"
HARI="${1:-$(date -u +%Y/%m/%d)}"
LAYANAN="${2:-}"
FOLDER="$AKAR/$HARI"

if [ ! -d "$FOLDER" ]; then
	echo "Tidak ada log untuk $HARI di $FOLDER" >&2
	exit 1
fi

if ! command -v jq >/dev/null 2>&1; then
	echo "jq belum terpasang. Pasang dulu: apt install jq" >&2
	exit 1
fi

if [ -n "$LAYANAN" ]; then
	POLA=("$FOLDER/$LAYANAN"-*.log)
else
	POLA=("$FOLDER"/*.log)
fi

shopt -s nullglob
BERKAS=("${POLA[@]}")
shopt -u nullglob

if [ ${#BERKAS[@]} -eq 0 ]; then
	echo "Tidak ada berkas log di $FOLDER" >&2
	exit 1
fi

# Satu filter jq per kebutuhan, digabung dengan "and".
FILTER='true'
if [ -n "${LOG_LEVEL:-}" ]; then
	FILTER="$FILTER and (.lvl == \"$LOG_LEVEL\")"
fi
if [ -n "${LOG_CARI:-}" ]; then
	FILTER="$FILTER and ((tostring) | test(\"$LOG_CARI\"; \"i\"))"
fi

# fromjson? membuang baris yang tidak terbaca, bukan menghentikan seluruh berkas.
jq -c "select($FILTER)" <(cat "${BERKAS[@]}" | jq -R 'fromjson? // empty')
