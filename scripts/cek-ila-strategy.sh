#!/usr/bin/env bash
#
# Membandingkan isi ila-strategy/ dengan sumber aslinya di go-trade/pkg/.
#
# Latar belakang: aturan ILA 2.0 ada di DUA tempat. Engine live di VPS 1 memakai
# salinan di go-trade/pkg, sedangkan backtest di platform memakai salinan di
# ila-strategy/. Bila keduanya berbeda, hasil backtest tidak lagi mencerminkan
# perilaku engine — dan tidak ada yang akan menyadarinya sampai angkanya
# meleset, yang bisa berbulan-bulan kemudian.
#
# Skrip ini membandingkan seluruh berkas sumber dan uji, karena keduanya
# disalin. Berkas yang hanya ada di satu sisi (mis. ila/version.go) dilaporkan
# sebagai informasi.
#
# Pemakaian:
#   scripts/cek-ila-strategy.sh                      # pakai path bawaan
#   ILA_SUMBER=/path/go-trade scripts/cek-ila-strategy.sh
#
# Bila memang ingin menyalin ulang setelah aturan berubah:
#   scripts/cek-ila-strategy.sh --salin

set -euo pipefail

AKAR_PLATFORM="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SUMBER="${ILA_SUMBER:-${AKAR_PLATFORM}/../go-trade}"
TUJUAN="${AKAR_PLATFORM}/ila-strategy"

MODE_SALIN=0
if [[ "${1:-}" == "--salin" ]]; then
  MODE_SALIN=1
fi

if [[ ! -d "${SUMBER}/pkg/ila" ]]; then
  echo "GAGAL: sumber tidak ditemukan di ${SUMBER}/pkg/ila" >&2
  echo "        Setel ILA_SUMBER ke folder go-trade." >&2
  exit 2
fi

# Ganti penanda import agar berkas yang dibandingkan sebanding: salinan di
# platform memakai "ila-strategy/indicators", sumbernya memakai
# "ila-engine/pkg/indicators". Tanpa penyeragaman ini setiap berkas akan
# terlihat berbeda padahal hanya nama module-nya yang beda.
seragamkan() {
  sed -E 's#"ila-(engine/pkg|strategy)/indicators"#"MODULE/indicators"#g' "$1"
}

if [[ ${MODE_SALIN} -eq 1 ]]; then
  cp -r "${SUMBER}/pkg/ila/." "${TUJUAN}/ila/"
  cp -r "${SUMBER}/pkg/indicators/." "${TUJUAN}/indicators/"
  perl -pi -e 's#"ila-engine/pkg/indicators"#"ila-strategy/indicators"#g' \
    "${TUJUAN}"/ila/*.go "${TUJUAN}"/indicators/*.go

  SHA="$(git -C "${SUMBER}" rev-parse HEAD 2>/dev/null || echo "tidak-diketahui")"
  echo "Disalin ulang dari ${SUMBER} (commit ${SHA})."
  echo "Jangan lupa: perbarui commit SHA di ila-strategy/ASAL.md dan naikkan"
  echo "Version di ila-strategy/ila/version.go bila aturan trading berubah."
  exit 0
fi

BEDA=0
DIBANDINGKAN=0
for PAKET in ila indicators; do
  for BERKAS in "${SUMBER}/pkg/${PAKET}"/*.go; do
    NAMA="$(basename "${BERKAS}")"

    if [[ ! -f "${TUJUAN}/${PAKET}/${NAMA}" ]]; then
      echo "HILANG   ${PAKET}/${NAMA} ada di sumber, tidak ada di ila-strategy"
      BEDA=1
      continue
    fi

    # Bandingkan isi. Bila sed gagal, diff akan melaporkan perbedaan — bukan
    # diam-diam dianggap sama. Kesalahan diam seperti itu justru yang paling
    # berbahaya pada pemeriksaan keselarasan.
    if ! diff -q <(seragamkan "${BERKAS}") <(seragamkan "${TUJUAN}/${PAKET}/${NAMA}") >/dev/null; then
      echo "BERBEDA  ${PAKET}/${NAMA}"
      BEDA=1
    fi
    DIBANDINGKAN=$((DIBANDINGKAN + 1))
  done
done

if [[ ${DIBANDINGKAN} -eq 0 ]]; then
  echo "GAGAL: tidak ada berkas yang dibandingkan. Periksa ILA_SUMBER="${SUMBER}"" >&2
  exit 2
fi

# Berkas yang hanya ada di sisi platform (mis. version.go) dilaporkan sebagai
# informasi, bukan kegagalan.
for PAKET in ila indicators; do
  for BERKAS in "${TUJUAN}/${PAKET}"/*.go; do
    NAMA="$(basename "${BERKAS}")"
    if [[ ! -f "${SUMBER}/pkg/${PAKET}/${NAMA}" ]]; then
      echo "TAMBAHAN ${PAKET}/${NAMA} hanya ada di ila-strategy"
    fi
  done
done

if [[ ${BEDA} -eq 1 ]]; then
  echo "" >&2
  echo "Aturan ILA 2.0 di platform BERBEDA dari engine live." >&2
  echo "Hasil backtest tidak akan mencerminkan perilaku sebenarnya." >&2
  echo "Jalankan ulang dengan --salin bila memang ingin menyelaraskan." >&2
  exit 1
fi

echo "Sepakat: ${DIBANDINGKAN} berkas dibandingkan, ila-strategy identik dengan ${SUMBER}/pkg."
