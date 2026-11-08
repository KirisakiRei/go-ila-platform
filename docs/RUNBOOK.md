# Runbook Operasional — Imaginary Liquidity

Panduan singkat saat sesuatu berjalan salah. Setiap bagian: **gejala → periksa →
tindakan → yang TIDAK boleh dilakukan**.

Prinsip yang berlaku di seluruh runbook:

| Prinsip | Arti |
|---|---|
| **Posisi berjalan tidak pernah ditutup sistem** | Broker yang menutup posisi lewat TP/SL. Platform hanya melaporkan. |
| **Menahan entry lebih aman daripada mengeksekusi buta** | Bila ragu, set mode `read-only`. |
| **Satu perubahan satu langkah** | Jangan mengubah flag dan mode bersamaan; jejak audit jadi kabur. |

---

## 0. Di Mana Log Berada

Semua layanan menulis JSONL harian ke folder yang sama di **disk VPS**, bukan di
dalam container. Log tetap ada walau container dibuat ulang atau dijalankan
`docker compose down -v`.

| VPS | Lokasi | Masa simpan |
|---|---|---|
| 2 (platform) | `/var/log/ila/YYYY/MM/DD/` | 30 hari |
| 1 (engine) | `/opt/data/ila-engine/logs/YYYY/MM/DD/` | 7 hari / 150 MB |

Isi folder di VPS 2:

```
/var/log/ila/2026/11/08/
  backend-api-2026-11-08.log
  runner-2026-11-08.log
  engine-2026-11-08.log     ← kiriman dari VPS 1
  caddy.log                 ← dirotasi Caddy sendiri
```

**Membaca log.** Tidak perlu `docker exec`:

```bash
./scripts/log.sh                          # hari ini, semua layanan
./scripts/log.sh 2026/11/07 runner        # satu layanan
LOG_LEVEL=ERROR ./scripts/log.sh          # hanya error
LOG_CARI=a1b2c3 ./scripts/log.sh          # lacak satu trace
```

**Satu insiden, dua VPS.** Cari `trace` yang sama di kedua tempat:

```bash
grep a1b2c3 /opt/data/ila-engine/logs/2026/11/08/engine-*.log   # VPS 1
grep a1b2c3 /var/log/ila/2026/11/08/runner-*.log                # VPS 2
```

**Level yang dipakai.** `DEBUG` detail internal (butuh `LOG_LEVEL=debug`),
`INFO` alur normal, `WARN` bisa pulih sendiri (reconnect, feed retry, login
gagal), `ERROR` butuh manusia. Penolakan input pengguna adalah `WARN`, bukan
`ERROR`: bila semua hal ditandai ERROR, tidak ada lagi yang menonjol.

**Deploy pertama wajib** menjalankan `sudo ./scripts/siapkan-log.sh` sekali,
agar container (UID 1001) bisa menulis ke folder tersebut.

---

## 1. Runner Mati

**Gejala.** Dashboard admin menampilkan Runner tidak tersambung. Peringatan
Telegram `Engine terputus` masuk (≤2 menit).

**Periksa.**

1. `docker ps` — apakah container `runner` berjalan?
2. `docker logs --tail 200 runner` — cari `error` pertama, bukan yang terakhir.
3. Cek `DB_DSN` dan `MASTER_KEY` masih terbaca.

**Tindakan.**

1. Nyalakan ulang container: `docker compose restart runner`.
2. Bila gagal terus karena konfigurasi, perbaiki `.env` lalu nyalakan lagi.
3. Setelah hidup, Runner menjalankan rekonsiliasi posisi otomatis.

**Jangan.** Menutup posisi manual karena Runner mati — broker tetap menjaga
TP/SL yang sudah terpasang.

---

## 2. Engine Terputus

**Gejala.** Sinyal baru tidak masuk, tetapi posisi berjalan tetap terpantau.

**Periksa.**

1. `/admin/kesehatan` → kolom "Terakhir terlihat" untuk setiap engine.
2. Dashboard user menampilkan banner engine terputus setelah 10 menit.
3. Cek log engine di VPS 1.

**Tindakan.**

1. Hidupkan ulang engine.
2. Bila engine akan lama mati, pengumuman di `/admin/cms` agar user tahu.

**Jangan.** Memalsukan sinyal secara manual untuk menutupi kekosongan.

---

## 3. Broker Error

**Gejala.** Peringatan `Banyak akun broker bermasalah` (≥3 akun tidak tersambung).

**Periksa.**

1. `/admin/kesehatan` → jumlah akun error.
2. Bila hanya satu akun: kemungkinan token broker kedaluwarsa milik user itu.

**Tindakan.**

1. Satu akun: minta user menghubungkan ulang akunnya (token broker).
2. Banyak akun sekaligus: kemungkinan gangguan sisi broker. Set mode
   `read-only` sampai pulih, lalu kembalikan ke `normal`.

**Jangan.** Membekukan akun user secara massal — pembekuan berarti keputusan
terhadap user, bukan terhadap gangguan teknis.

---

## 4. Disk Hampir Penuh

**Gejala.** Peringatan `Disk hampir penuh` (≥85 persen).

**Periksa.**

1. `df -h` dan `docker system df`.
2. Ukuran log: `du -sh /var/log/ila /var/lib/docker/containers/*/*-json.log`.

**Tindakan.**

1. Log aplikasi sudah merawat diri sendiri (30 hari / 1 GB di VPS 2, 7 hari /
   150 MB di VPS 1). Bila masih penuh, periksa `du -sh /var/log/ila/*`, lalu
   hapus hari terlama manual: `rm -rf /var/log/ila/2026/11/01`.
2. Stdout container tetap dibatasi `max-size 10m, max-file 2`; bila perlu
   diputar: `docker compose up -d --force-recreate`.
3. Hapus backup lama di luar retensi: `ls -1 /var/backups/ila`.
4. Bila masih penuh, lihat tabel terbesar di PostgreSQL dan arsipkan.

**Jangan.** Menghapus tabel audit untuk menghemat ruang.

---

## 5. Database Tidak Merespons

**Gejala.** Endpoint mengembalikan galat 500; `/health` tetap hidup tetapi
`/ready` gagal.

**Periksa.**

1. `docker logs postgres` — cari `out of memory` atau `no space left`.
2. `SELECT count(*) FROM pg_stat_activity;` — jumlah koneksi.

**Tindakan.**

1. Bila koneksi penuh: naikkan `DB_MAX_OPEN_CONNS` atau turunkan jumlah replika.
2. Bila disk penuh: ikuti bagian 4.
3. Pulihkan dari backup hanya bila database rusak, dan ke **database kosong**
   lebih dulu (lihat `scripts/restore.sh`).

**Jangan.** Menjalankan `pg_restore --clean` pada database yang sedang melayani
permintaan.

---

## 6. Pemulihan Data

1. Siapkan database kosong.
2. Pastikan `MASTER_KEY` sama dengan saat backup dibuat.
3. Jalankan `scripts/restore.sh`.
4. Jalankan `go run ./cmd/migrate` untuk memastikan skema mutakhir.
5. Uji: login satu akun, buka satu akun broker, pastikan saldo terbaca.

**Ingat.** Backup tanpa `MASTER_KEY` hanya memulihkan data, bukan kemampuan
membuka kredensial broker. Simpan `MASTER_KEY` terpisah dari backup.

---

## 7. Mode Pemeliharaan

| Tingkat | Login | Entry baru | Posisi berjalan |
|---|:--:|:--:|:--:|
| `normal` | ✅ | ✅ | ✅ |
| `read-only` | ✅ | ❌ (dilewati) | ✅ |
| `maintenance` | Banner pemeliharaan | ❌ (dilewati) | ✅ |

**Cara mengaktifkan.** `/admin/pemeliharaan`, ketik `MAINTENANCE`, isi alasan
minimal 10 karakter. Setiap perubahan tercatat atas nama pelaku.

**Catatan.** Sinyal saat `read-only` **dilewati**, bukan ditunda: harga sudah
berubah, sehingga mengeksekusi belakangan lebih berbahaya daripada melewatkan.
