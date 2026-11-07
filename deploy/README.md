# Panduan Deploy — VPS 2

> Dokumen ini khusus untuk menjalankan ILA Platform di VPS 2.
> Engine tetap berjalan terpisah di VPS 1 (dipegang Hermes).

---

## 1. Bentuk Akhir

Enam container berjalan di VPS 2. **Hanya Caddy** yang terbuka ke internet.

```
                    Internet (443)
                           │
                    ┌──────▼──────┐
                    │    Caddy    │  TLS otomatis
                    └──────┬──────┘
                           │
          ┌────────────────┼────────────────┐
          │ /api/*         │ /*             │
    ┌─────▼─────┐    ┌─────▼─────┐         │
    │ backend   │    │  frontend │         │
    │  :8080    │    │  nginx    │         │
    └─────┬─────┘    └───────────┘         │
          │                                │
    ┌─────▼─────┐   ┌──────────┐           │
    │  runner   │──▶│ postgres │           │
    │  :8081    │   └──────────┘           │
    └───────────┘                          │
```

| Container | Port terbuka | Peran |
|---|---|---|
| `caddy` | **80, 443** | satu-satunya pintu masuk, TLS otomatis |
| `backend-api` | tidak | API untuk frontend |
| `runner` | tidak | eksekusi order per user |
| `frontend` | tidak | aset statis SPA |
| `postgres` | tidak | basis data |
| `migrate` | tidak | membuat tabel, lalu berhenti |

**Mengapa satu domain:** cookie sesi memakai `SameSite=Strict`. Kalau frontend
dan API dipisah domain, deep-link konfirmasi dari Telegram kehilangan sesi.

---

## 2. Persiapan Awal di VPS

```bash
sudo apt update && sudo apt install -y git make
curl -fsSL https://get.docker.com | sudo sh
sudo usermod -aG docker "$USER"
```

Keluar lalu masuk lagi agar keanggotaan grup berlaku.

```bash
git clone https://github.com/KirisakiRei/go-ila-platform.git
cd go-ila-platform
```

---

## 3. Membuat `.env`

```bash
cp .env.example .env
```

Isi nilai berikut. **Jangan** memakai nilai contoh di produksi.

| Variabel | Cara mengisi |
|---|---|
| `APP_ENV` | `production` |
| `DB_PASSWORD` | `openssl rand -base64 24` |
| `JWT_SECRET` | `openssl rand -hex 32` |
| `MASTER_KEY` | `openssl rand -base64 32` |
| `INTERNAL_SECRET` | `openssl rand -hex 32` |
| `SITE_ADDRESS` | domain Anda, mis. `ila.id` |
| `ACME_EMAIL` | email aktif untuk Let's Encrypt |
| `PUBLIC_BASE_URL` | `https://ila.id` |
| `ALLOWED_ORIGINS` | `https://ila.id` |
| `COOKIE_SECURE` | **`true`** |

`DB_DSN` **tidak perlu** diisi — compose menyusunnya dari `DB_USER`,
`DB_PASSWORD`, dan `DB_NAME` sehingga password hanya hidup di satu tempat.

> Server **menolak start** bila `APP_ENV=production` tetapi `COOKIE_SECURE`
> bukan `true`. Ini disengaja.

`MASTER_KEY` harus **base64 dari 32 byte**. Nilai acak biasa ditolak.

---

## 4. Arahkan Domain

Buat catatan DNS tipe **A**:

```
ila.id        →  <IP publik VPS 2>
www.ila.id    →  <IP publik VPS 2>   (opsional)
```

Pastikan port **80 dan 443** terbuka di firewall AWS (Security Group).
Caddy butuh port 80 untuk verifikasi Let's Encrypt.

---

## 5. Menjalankan

```bash
make up
```

Yang terjadi berurutan:

1. Image dibangun.
2. `postgres` menyala dan sehat.
3. `migrate` membuat 16 tabel lalu berhenti.
4. `backend-api` dan `runner` mulai.
5. `frontend` sehat.
6. `caddy` mengambil sertifikat TLS lalu mulai menerima permintaan.

Cek status:

```bash
make ps
```

Semua harus `Up (healthy)`, kecuali `caddy` (tanpa healthcheck) dan
`migrate` yang memang berhenti setelah selesai.

---

## 6. Memeriksa Hasil

```bash
curl -I https://ila.id                       # 200, HTML
curl -o /dev/null -w "%{http_code}\n" https://ila.id/api/v1/me   # 401 (benar)
curl https://ila.id/health                   # ok
```

`401` pada `/api/v1/me` adalah tanda **benar**: endpoint hidup dan menolak
permintaan tanpa sesi.

---

## 7. Memperbarui Versi

```bash
git pull
make up
```

Bila ada perubahan skema, `migrate` berjalan ulang lebih dulu karena
`backend-api` menunggunya selesai.

---

## 8. Perintah Sehari-hari

| Perintah | Kegunaan |
|---|---|
| `make ps` | status container |
| `make logs` | log semua container |
| `make down` | hentikan tanpa menghapus data |
| `make up-cold` | **hapus data** lalu mulai dari nol |
| `make config-check` | validasi compose |
| `make caddy-check` | validasi Caddyfile |
| `make nginx-check` | validasi nginx.conf |

---

## 9. Backup

Data ada di volume `pgdata`.

```bash
docker compose --env-file .env -f deploy/docker-compose.yml \
  exec -T postgres pg_dump -U ila ila | gzip > backup-$(date +%F).sql.gz
```

Simpan di luar VPS. **`MASTER_KEY` tidak boleh ikut backup** dan tidak boleh
hilang: tanpa kunci itu, token broker milik user tidak bisa dibuka lagi.

---

## 10. Bila Ada Masalah

| Gejala | Sebab biasanya |
|---|---|
| `caddy` restart terus | `SITE_ADDRESS` salah, port 80/443 tertutup, atau DNS belum mengarah |
| `backend-api` mati saat start | `.env` kurang, atau `COOKIE_SECURE` bukan `true` di produksi |
| `master key tidak sah` | `MASTER_KEY` bukan base64 dari 32 byte |
| Login berhasil tapi `/me` 401 | cookie ditolak browser: pastikan HTTPS aktif dan domain sama |
| Halaman 404 saat refresh | `nginx.conf` tidak terpakai; jalankan `make nginx-check` |
| `DB_PASSWORD wajib diisi` | `.env` belum ada atau `make` dijalankan dari luar folder repo |

Lihat log satu container:

```bash
docker compose --env-file .env -f deploy/docker-compose.yml logs -f caddy
```

---

## 11. Catatan Kapasitas

VPS 2 dengan RAM 1 GB sudah diuji cukup untuk enam container ini.

| Container | Batas memori |
|---|---|
| postgres | 160 MB |
| backend-api | 128 MB |
| runner | 128 MB |
| frontend | 96 MB |
| caddy | 64 MB |

Sisanya untuk sistem. Bila RAM menipis, aktifkan swap 1 GB:

```bash
sudo fallocate -l 1G /swapfile && sudo chmod 600 /swapfile
sudo mkswap /swapfile && sudo swapon /swapfile
echo '/swapfile none swap sw 0 0' | sudo tee -a /etc/fstab
```
