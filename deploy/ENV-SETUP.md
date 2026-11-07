# Menyiapkan `.env` Produksi

> `.env` berisi seluruh rahasia platform. File ini **tidak pernah** di-commit
> (sudah masuk `.gitignore`).

---

## 1. Membuat Rahasia

Jalankan di VPS. Semua nilai dihasilkan secara acak kriptografis.

```bash
# Password database (24 byte)
openssl rand -base64 24

# Kunci tanda tangan sesi (32 byte hex)
openssl rand -hex 32

# Master key kredensial broker (32 byte base64) — WAJIB format ini
openssl rand -base64 32

# Secret antar-container
openssl rand -hex 32
```

> `MASTER_KEY` **harus** base64 dari tepat 32 byte. Nilai acak biasa ditolak
> dan server tidak mau start.

---

## 2. Isi yang Wajib Diganti

Mulai dari `cp .env.example .env`, lalu ubah:

| Variabel | Nilai produksi |
|---|---|
| `APP_ENV` | `production` |
| `PUBLIC_BASE_URL` | `https://domain-anda` |
| `ALLOWED_ORIGINS` | `https://domain-anda` |
| `SITE_ADDRESS` | `domain-anda` |
| `ACME_EMAIL` | email aktif Anda |
| `DB_PASSWORD` | hasil `openssl rand -base64 24` |
| `JWT_SECRET` | hasil `openssl rand -hex 32` |
| `MASTER_KEY` | hasil `openssl rand -base64 32` |
| `INTERNAL_SECRET` | hasil `openssl rand -hex 32` |
| `COOKIE_SECURE` | `true` |

`DB_DSN` **dibiarkan kosong** — compose menyusunnya dari
`DB_USER`, `DB_PASSWORD`, dan `DB_NAME`.

---

## 3. Verifikasi Sebelum Nyalakan

```bash
make config-check
```

Perintah ini menolak konfigurasi yang salah tanpa menjalankan container.

---

## 4. Memastikan Rahasia Tidak Bocor

Cek bahwa frontend dan Caddy **tidak** menerima kredensial:

```bash
docker compose --env-file .env -f deploy/docker-compose.yml config \
  | grep -A20 "  caddy:" | grep -E "MASTER_KEY|DB_PASSWORD" \
  && echo "BOCOR!" || echo "aman"
```

Hasil yang benar: `aman`.

---

## 5. Menyimpan Master Key

`MASTER_KEY` adalah satu-satunya benda yang membuat isi database bisa dibaca.

| ✅ | ❌ |
|---|---|
| Simpan di password manager | Menyimpan di database |
| Catat di dua tempat aman | Ikut dalam backup database |
| Salin manual saat pindah server | Dikirim lewat Telegram atau chat |

**Bila kunci ini hilang, semua token broker user tidak bisa dibuka lagi.
Permanen. Tidak ada pemulihan.**

---

## 6. Rotasi Master Key

Dipakai bila kunci diduga bocor. **Data lama tetap terbaca** karena kunci
lama didaftarkan sebagai kunci warisan.

**Langkah:**

1. Buat kunci baru:
   ```bash
   openssl rand -base64 32
   ```

2. Simpan kunci lama dulu. Misalnya kunci lama adalah versi `1`:
   ```
   MASTER_KEYS_LEGACY=1:<kunci-lama>
   ```

3. Set kunci baru sebagai aktif:
   ```
   MASTER_KEY=<kunci-baru>
   MASTER_KEY_VERSION=2
   ```

4. Nyalakan ulang:
   ```bash
   make up
   ```

5. Akun yang kredensialnya ditulis ulang otomatis memakai versi 2. Setelah
   semua akun berpindah, hapus `MASTER_KEYS_LEGACY`.

> Selama `MASTER_KEYS_LEGACY` masih ada, kunci lama masih bisa membuka data
> lama. Hapus hanya setelah tidak ada blob versi lama.

---

## 7. Memisahkan `.env` untuk VPS

`.env` yang dipakai saat pengujian **harus diganti** saat pindah ke VPS
sungguhan, terutama:

- `SITE_ADDRESS` dan `ACME_EMAIL` → domain nyata
- `MASTER_KEY`, `JWT_SECRET`, `DB_PASSWORD`, `INTERNAL_SECRET` → hasil
  generate baru di VPS

**Jangan menyalin `MASTER_KEY` dari pengujian ke produksi.** Kunci yang pernah
tertulis di chat atau berkas sementara dianggap sudah bocor.

---

## 8. Telegram (Opsional)

Notifikasi Telegram tidak aktif bila `TELEGRAM_BOT_TOKEN` kosong; fitur lain
tetap berjalan normal.

| Variabel | Isi |
|---|---|
| `TELEGRAM_BOT_TOKEN` | dari `@BotFather` |
| `TELEGRAM_BOT_USERNAME` | tanpa `@`, mis. `ILABot` |
| `TELEGRAM_WEBHOOK_SECRET` | `openssl rand -hex 32` |

Daftarkan webhook setelah domain aktif:

```bash
curl -X POST "https://api.telegram.org/bot<TOKEN>/setWebhook" \
  -d "url=https://domain-anda/api/v1/telegram/webhook" \
  -d "secret_token=<TELEGRAM_WEBHOOK_SECRET>"
```
