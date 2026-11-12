# ANALISIS v2 — Invite Only (Register Ditutup) + Report Bug dengan Lampiran

Status: BELUM DIKERJAKAN. Analisis konsolidasi setelah jawaban human.

---

## 0. Keputusan human (final)

| # | Pertanyaan | Jawaban |
|---|---|---|
| 1 | Pembayaran | **Dinonaktifkan total.** Hanya orang yang di-invite boleh pakai |
| 2 | Isi halaman billing | Diganti copy **"Invite only"** |
| 3 | Register | **Ditutup.** Hanya invite |
| 4 | Report bug | Untuk **semua user**, ditampung di **panel admin** |
| 5 | Severity | **Tidak dipakai.** Cukup kategori + **lampiran file** |

---

## 1. Temuan yang mengubah rancangan

### 1.1 Data produksi: aman untuk menutup pembayaran

```
tier user : plus x2 (seluruh pengguna)
subscription: 4 baris — ini WATCHLIST per simbol, bukan billing
```

**Artinya:** tidak ada satu pun pelanggan berbayar yang bergantung pada halaman
billing. Menonaktifkan pembayaran **tidak merugikan siapa pun** hari ini.
Catatan: `subscription` di skema ini adalah daftar simbol yang dipantau,
bukan langganan berbayar — jangan tertukar saat membaca kode.

### 1.2 Register publik: satu fungsi, satu pintu

`POST /api/v1/auth/register` → `AuthHandler.Register` → `authService.Register`.

Sudah tersedia:
- `h.registerLimiter` (batas per IP+email) — bisa dipakai ulang sebagai pengaman.
- `adminUserService.Create` — **sudah lengkap**: membuat user dengan password,
  tier, `EmailVerifiedAt` terisi, `OnboardingDone: true`. **Admin sudah bisa
  membuat akun tanpa register publik.** Inilah pintu invite.

**Konsekuensi:** implementasi cukup menutup satu endpoint. Tidak perlu membangun
sistem undangan (token, email invite) — sesuai keputusan #3.

### 1.3 Belum ada mekanisme upload sama sekali

| Komponen | Status |
|---|---|
| Handler `multipart/form-data` | ❌ tidak ada |
| Config storage | ❌ tidak ada |
| Volume writable untuk backend | ✅ ada: `${LOG_DIR_HOST}: /var/log/ila` |
| Batas memori container | ⚠️ `mem_limit: 160m` |

**Implikasi:** lampiran bukan sekadar "tambah input file". Perlu:
- parsing multipart + validasi tipe & ukuran,
- direktori penyimpanan + pembuatan folder,
- endpoint penyajian file **dengan kontrol akses** (tidak boleh publik),
- pembersihan saat laporan dihapus.

Batas `160m` memaksa ukuran file **kecil** (rekomendasi 2 MB/file).

---

## 2. Rancangan: Invite Only

### 2.1 Flag

```
settings.KeyInviteOnly = "INVITE_ONLY"   // default true
dto.PublicStatus += InviteOnly bool
```

Dibaca lewat `GET /api/v1/status` (publik, sudah dikonsumsi `useSystemStatus()`).

**Default bila gagal baca: `true`** (fail-closed). Berbeda dengan analisis v1
yang menyarankan fail-open untuk billing: di sini **menutup register lebih aman**
daripada menerima pendaftaran tak diinginkan.

### 2.2 Backend

| Area | Perilaku saat `invite_only = true` |
|---|---|
| `POST /auth/register` | Tolak `403` dengan pesan "Pendaftaran hanya lewat undangan" |
| `GET /billing/plans` | `404` (publik) |
| Checkout / transaksi | `403` |
| Admin create user | **tetap jalan** |
| Login user lama | **tetap jalan** |
| Trial | **dinonaktifkan juga** — karena tidak ada pembayaran, trial tidak bermakna |
| Runner / sinyal / eksekusi | **tidak disentuh** |

⚠️ Trial: perlu konfirmasi. Rekomendasi: matikan `StartTrial` saat invite-only,
karena trial adalah pintu masuk jalur berbayar yang sekarang ditutup.

### 2.3 Frontend — 11 titik masuk (dari analisis v1)

Saat `invite_only`:
- 9 titik UI (menu sidebar, nav mobile, drawer, quick action, AccountStatusCard ×2,
  Settings mobile, tombol upgrade ×2) → **disembunyikan**.
- Rute `/langganan` & `/langganan/bayar` → **redirect** ke `/dashboard`
  dengan copy "Invite only", **rute tidak dihapus** agar bisa dibalik tanpa deploy.
- Halaman login/register → tombol register diganti copy **"Akses hanya lewat undangan"**.

---

## 3. Rancangan: Report Bug (dengan lampiran)

### 3.1 Model `bug_report`

```
Base
UserID    varchar(36) index
Category  varchar(24)      -- bug | ui | data | lainnya
Title     varchar(120)
Body      text
PageURL   varchar(255)     -- diisi otomatis
UserAgent varchar(255)     -- diisi otomatis
Status    varchar(16) default 'baru'   -- baru | dibaca | selesai
AdminNote text
```

### 3.2 Model `bug_report_attachment` (tabel terpisah)

```
Base
ReportID  varchar(36) index
FileName  varchar(160)     -- nama asli, sudah dibersihkan
MimeType  varchar(80)
SizeBytes bigint
StoredAs  varchar(160)     -- nama acak di disk, bukan nama asli
```

**Kenapa tabel terpisah, bukan kolom JSON:** memudahkan hitung pemakaian,
pembersihan saat hapus laporan, dan tidak membebani baris utama.

### 3.3 Penyimpanan lampiran

- Direktori: `${LOG_DIR_HOST}/bug-report/<report_id>/<stored_as>`
  (memakai volume yang **sudah ada**, tidak perlu volume baru).
- Nama file di disk **acak** (UUID), bukan nama asli → mencegah path traversal
  dan tabrakan nama.
- **Yang boleh diunggah:** `image/png`, `image/jpeg`, `image/webp`,
  `text/plain`, `application/pdf`.
- **Batas:** 2 MB per file, maksimal **3 file** per laporan.
- Validasi tipe **dari isi file** (magic bytes), bukan hanya `Content-Type`
  yang dikirim browser (bisa dipalsukan).

### 3.4 Penyajian file (penting)

`GET /api/v1/bug-reports/:id/attachments/:aid`

- Hanya **pemilik laporan** atau **admin**.
- Kirim `Content-Disposition: attachment` + `X-Content-Type-Options: nosniff`
  agar file tidak dieksekusi di browser.
- **Jangan** sajikan sebagai HTML/SVG (vektor XSS).

### 3.5 Endpoint

| Method | Path | Akses |
|---|---|---|
| `POST` | `/api/v1/bug-reports` (multipart) | user login |
| `GET` | `/api/v1/bug-reports` | user login, miliknya saja |
| `GET` | `/api/v1/bug-reports/:id/attachments/:aid` | pemilik / admin |
| `GET` | `/api/v1/admin/bug-reports` | admin, semua |
| `GET` | `/api/v1/admin/bug-reports/:id` | admin, detail + lampiran |
| `PATCH` | `/api/v1/admin/bug-reports/:id` | admin: status + catatan |

### 3.6 Batasan

| Batasan | Nilai | Alasan |
|---|---|---|
| Laporan / jam / user | 3 | Cegah spam |
| Judul | ≤120 char | Kolom DB |
| Isi | ≤4000 char | Cegah kiriman raksasa |
| Lampiran | 3 × 2 MB | `mem_limit: 160m` |
| Total laporan per user | 50 | Cegah disk penuh |

### 3.7 Frontend

- Rute `/laporkan-bug` + `/laporkan-bug/riwayat`.
- Form: kategori (dropdown), judul, deskripsi, **lampiran (opsional)**.
- `page_url` & `user_agent` diisi otomatis.
- Tautan masuk dari menu "Bantuan".
- Panel admin: daftar laporan, filter status, detail + lampiran, ubah status.

---

## 4. Urutan pengerjaan

| Fase | Isi | Risiko |
|---|---|---|
| 1 | Flag `INVITE_ONLY` + `invite_only` di `PublicStatus` | Rendah |
| 2 | Tutup register + guard API billing + matikan trial | Rendah |
| 3 | Sembunyikan 11 titik UI + redirect rute + copy invite-only | Rendah |
| 4 | Model + migrasi `bug_report` & `bug_report_attachment` | Rendah |
| 5 | Endpoint user (POST/GET + upload + penyajian aman) | **Sedang-tinggi** |
| 6 | Halaman form + riwayat laporan | Sedang |
| 7 | Endpoint + UI admin | Sedang |

Fase 1–3 lebih dulu karena itu yang menutup akses.

---

## 5. Risiko utama

| Risiko | Penanganan |
|---|---|
| Register ditutup tapi admin belum bisa buat akun | Sudah bisa (`adminUserService.Create`) — **verifikasi manual dulu** sebelum deploy |
| User lama tidak bisa login | Login **tidak** disentuh; hanya register |
| Flag gagal dibaca → register terbuka | Default `true` (fail-closed) |
| Lampiran jadi jalan XSS | `nosniff` + `Content-Disposition: attachment` + tolak SVG/HTML |
| Disk penuh | Batas 3×2 MB + maks 50 laporan/user |
| `mem_limit 160m` habis saat upload | Streaming ke disk, **jangan** baca seluruh file ke memori |
| Volume log & lampiran tercampur | Subdirektori terpisah `bug-report/` |

---

## 6. Pertanyaan tersisa (hanya 2)

1. **Trial ikut dimatikan saat invite-only?**
   Rekomendasi: **ya**, karena trial adalah pintu jalur berbayar yang ditutup.

2. **Lampiran: cukup gambar + PDF, atau izinkan file teks juga?**
   Rekomendasi: **gambar (png/jpg/webp) + PDF**. File teks jarang membantu
   dan lebih sulit divalidasi isinya.
