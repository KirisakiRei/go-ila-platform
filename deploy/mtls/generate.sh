#!/bin/sh
# Membuat CA, sertifikat server engine, dan sertifikat klien runner.
#
# Hasil ditulis ke deploy/certs/ yang sudah di-.gitignore.
# Jalankan sekali; simpan ca.key terpisah dari server (plans/09 §2.2).
set -eu

OUT_DIR="${1:-$(dirname "$0")/../certs}"
ENGINE_CN="${ENGINE_CN:-engine.internal}"
RUNNER_CN="${RUNNER_CN:-runner.internal}"
DAYS="${DAYS:-90}"

mkdir -p "$OUT_DIR"
cd "$OUT_DIR"

if [ -f ca.key ]; then
  echo "CA sudah ada; lewati pembuatan." >&2
else
  echo "Membuat CA..." >&2
  openssl req -x509 -newkey rsa:4096 -sha256 -days 3650 -nodes \
    -keyout ca.key -out ca.crt -subj "/CN=ILA Platform CA"
fi

gen_pair() {
  name="$1"
  cn="$2"
  ext="$3"

  echo "Membuat sertifikat $name..." >&2
  openssl req -newkey rsa:2048 -sha256 -nodes \
    -keyout "$name.key" -out "$name.csr" -subj "/CN=$cn"

  openssl x509 -req -in "$name.csr" -CA ca.crt -CAkey ca.key -CAcreateserial \
    -out "$name.crt" -days "$DAYS" -sha256 -extfile "$ext"

  rm -f "$name.csr"
}

cat > engine.ext <<EOF
basicConstraints=CA:FALSE
keyUsage=digitalSignature,keyEncipherment
extendedKeyUsage=serverAuth
subjectAltName=DNS:$ENGINE_CN
EOF

cat > runner.ext <<EOF
basicConstraints=CA:FALSE
keyUsage=digitalSignature,keyEncipherment
extendedKeyUsage=clientAuth
subjectAltName=DNS:$RUNNER_CN
EOF

gen_pair engine-server "$ENGINE_CN" engine.ext
gen_pair runner-client "$RUNNER_CN" runner.ext

rm -f engine.ext runner.ext ca.srl

chmod 600 ./*.key 2>/dev/null || true

echo "Selesai. File di $OUT_DIR:" >&2
ls -1 ./*.crt ./*.key
