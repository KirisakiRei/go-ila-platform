#!/usr/bin/env bash
# Menerbitkan sertifikat mTLS untuk Engine ILA di VPS 1.
#
# Dijalankan DI VPS 2, di folder deploy/certs, karena hanya di sinilah ca.key
# berada. ca.key TIDAK ikut disalin ke VPS 1: bila VPS 1 jebol, penyerang tidak
# boleh bisa menerbitkan sertifikat palsu yang dipercaya platform.
#
# WAJIB lewat sudo: ca.key sengaja dimiliki root dengan mode 640, sehingga user
# biasa tidak bisa membacanya. Kegagalan membacanya muncul sebagai sertifikat
# 0 byte, bukan sebagai pesan izin yang jelas.
#
#   sudo bash scripts/siapkan-sertifikat-engine.sh
#   sudo chown ubuntu:1001 deploy/certs/engine-*
#
# Dua sertifikat yang dibuat:
#   engine-server.*  identitas Engine  (dipasang di VPS 1)
#   engine-client.*  identitas Runner  (dipasang di VPS 2, dipakai Runner)
#
# Engine memverifikasi Runner lewat engine-client (mTLS), dan Runner
# memverifikasi Engine lewat engine-server. Keduanya ditandatangani CA internal.
#
# Aman dijalankan berulang: berkas yang sudah ada dan SAH tidak ditimpa
# kecuali FORCE=1. Sertifikat kosong/rusak selalu dibuat ulang.

set -euo pipefail

CA_CRT="ca.crt"
CA_KEY="ca.key"
HARI=3650

if [ ! -f "$CA_CRT" ] || [ ! -f "$CA_KEY" ]; then
	echo "Jalankan skrip ini di folder deploy/certs yang berisi ca.crt dan ca.key." >&2
	exit 1
fi

# Berkas ekstensi OPENSSL_CONF ditulis ke berkas biasa, bukan process
# substitution: openssl menolak /dev/fd pada beberapa lingkungan non-interaktif.
EKSTENSI="$(mktemp)"
trap 'rm -f "$EKSTENSI"' EXIT

sah() {
	# Sertifikat dianggap sah bila berkasnya ada dan bisa diurai openssl.
	[ -s "${1}.crt" ] && openssl x509 -in "${1}.crt" -noout 2>/dev/null
}

buat() {
	local nama="$1" cn="$2" san="$3"

	if sah "$nama" && [ "${FORCE:-0}" != "1" ]; then
		echo "Lewati ${nama}: sudah sah (pakai FORCE=1 untuk menerbitkan ulang)."
		return
	fi

	printf 'subjectAltName=%s\nextendedKeyUsage=serverAuth,clientAuth\n' "$san" > "$EKSTENSI"

	openssl req -new -newkey rsa:2048 -nodes \
		-keyout "${nama}.key" -out "${nama}.csr" \
		-subj "/CN=${cn}" 2>/dev/null

	openssl x509 -req -in "${nama}.csr" \
		-CA "$CA_CRT" -CAkey "$CA_KEY" -CAcreateserial \
		-days "$HARI" -sha256 \
		-extfile "$EKSTENSI" \
		-out "${nama}.crt" 2>/dev/null

	rm -f "${nama}.csr"
	echo "Dibuat: ${nama}.crt / ${nama}.key  (SAN: ${san})"
}

# SAN engine-server mencakup IP lokal AWS VPS 1 dan IP publiknya, supaya
# Runner tetap bisa terhubung baik lewat jaringan lokal maupun bila lokal
# sedang bermasalah.
buat engine-server "ila-engine" "IP:172.26.1.188,IP:13.54.63.73,DNS:engine,DNS:localhost,IP:127.0.0.1"
buat engine-client "ila-runner" "DNS:runner,DNS:localhost,IP:127.0.0.1"

echo
echo "Verifikasi rantai:"
for f in engine-server.crt engine-client.crt; do
	openssl verify -CAfile "$CA_CRT" "$f"
done

# Kunci hanya boleh dibaca pemilik dan grup. Mode 600 membuat container UID
# 1001 gagal membacanya, jadi grup dipakai seperti pola sertifikat log.
chmod 640 *.key
echo
echo "Selesai. Yang disalin ke VPS 1 hanya:"
echo "  engine-server.crt  engine-server.key  ca.crt  engine-client.crt  engine-client.key"
echo "ca.key TIDAK pernah keluar dari VPS 2."
