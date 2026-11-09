#!/usr/bin/env bash
# Memasang public key laptop ke VPS 1 (Engine) agar bisa masuk tanpa kata sandi.
#
# Jalankan skrip ini DI DALAM terminal web AWS Lightsail VPS 1, bukan di laptop.
# Skrip ini sengaja berdiri sendiri: tidak butuh repo, tidak butuh Docker, dan
# aman dijalankan berulang kali.
#
# Cara pakai:
#   bash pasang-key-vps1.sh
# atau salin-tempel seluruh isi berkas ini ke terminal VPS 1.

set -euo pipefail

PUBKEY="ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIPoCHOFP05/GLj04UKJyPVkZBbqWYdFM+LYw/P5Gv7g9 rynv@laptop-vps1-engine"
AUTH="$HOME/.ssh/authorized_keys"

mkdir -p "$HOME/.ssh"
chmod 700 "$HOME/.ssh"
touch "$AUTH"

# Idempoten: jangan menumpuk baris yang sama setiap kali dijalankan.
if grep -qxF "$PUBKEY" "$AUTH" 2>/dev/null; then
	echo "Key sudah terpasang, tidak ada yang diubah."
else
	printf '%s\n' "$PUBKEY" >> "$AUTH"
	echo "Key ditambahkan."
fi

# Izin yang salah membuat sshd menolak key tanpa pesan yang jelas.
chmod 600 "$AUTH"

# Pemilik harus user yang login: berkas milik root dengan HOME ubuntu membuat
# StrictModes menolak authorized_keys.
chown -R "$(id -un):$(id -gn)" "$HOME/.ssh"

echo
echo "Selesai. Isi authorized_keys sekarang:"
echo "--------------------------------------------------------------"
cat "$AUTH"
echo "--------------------------------------------------------------"
echo
echo "Sekarang coba dari laptop:"
echo "  ssh -i ~/.ssh/id_ed25519_vps1 ubuntu@13.54.63.73"
