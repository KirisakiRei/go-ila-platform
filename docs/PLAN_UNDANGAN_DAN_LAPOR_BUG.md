# Plan Implementasi — Undangan Komunitas (Invite Link) + Lapor Bug

**Status:** menunggu persetujuan
**Ruang lingkup:** `go-ila-platform` (backend + frontend). `go-trade` tidak disentuh.
**Prinsip:** satu rilis, dua fitur, deploy sekali.

---

## 0. Keputusan yang sudah dikunci

| Hal | Keputusan |
|---|---|
| Batas IP | **Tidak ada** untuk pendaftaran lewat invite sah |
| Masa berlaku | **Wajib**, bawaan 7 hari, admin bisa ubah |
| Kuota | **Angka wajib ≥ 1**, diatur admin. Tidak ada "tanpa batas" |
| Kuota hangus | Dikembalikan bila akun gagal dibuat (satu transaksi DB) |
| Tier | **plus** (bisa diubah admin per undangan) |
| Email tujuan | **Opsional**. Bila diisi → link terikat, form terisi otomatis |
| Payment/billing | **Tidak disentuh**, tetap tertutup oleh `INVITE_ONLY` |
| Lampiran lapor bug | Gambar (png/jpg/webp) + PDF, maks **1 MB** (§6) |

### Arti "tidak ada batas IP"
Pendaftaran lewat invite sah **dibebaskan** dari `RegisterLimiter` (5/IP/jam, 1/email/jam).
Yang tetap berlaku: `RateLimiter` per IP pada endpoint (§2.2) — ini menahan banjir
permintaan HTTP, bukan menahan anggota komunitas yang berbagi satu IP kantor.
**Batas sesungguhnya adalah kuota + masa berlaku.** Itu yang dipakai, bukan IP.

---

## 1. Model data

### 1.1 `model.invite_code`

```go
type InviteCode struct {
    Base
    // CodeHash SHA-256 dari kode mentah. Kode asli tidak pernah disimpan:
    // bila database bocor, link yang ada di dalamnya tidak bisa dipakai.
    CodeHash string `gorm:"type:varchar(64);uniqueIndex;not null" json:"-"`
    // PrefixPotongan empat karakter awal, hanya untuk dikenali admin di daftar.
    // Tidak cukup untuk menebak kode utuh.
    PrefixPotongan string `gorm:"type:varchar(12);not null;default:''" json:"prefix"`
    // EmailTujuan kosong berarti link berlaku untuk siapa pun pemegangnya.
    EmailTujuan string `gorm:"type:varchar(150);not null;default:''" json:"email"`
    Tier        string `gorm:"type:varchar(20);not null;default:'plus'" json:"tier"`
    // MaxUses wajib >= 1. Tidak ada nilai 0 = tanpa batas.
    MaxUses int       `gorm:"not null;default:1" json:"max_uses"`
    UsedCount int     `gorm:"not null;default:0" json:"used_count"`
    ExpiresAt time.Time `gorm:"not null" json:"expires_at"`
    IsActive  bool      `gorm:"not null;default:true" json:"is_active"`
    Note      string    `gorm:"type:varchar(255);not null;default:''" json:"note"`
    CreatedBy string    `gorm:"type:varchar(36);not null" json:"created_by"`
}
func (InviteCode) TableName() string { return "invite_code" }
```

Kolom yang **sengaja tidak ada**: `created_at` manual (sudah ada di `Base`),
`used_count` sebagai sumber kebenaran tunggal untuk kuota (lihat §3.2),
dan kolom tier terpisah untuk "tier saat daftar" (dibekukan ke user, bukan ke invite).

### 1.2 `model.invite_redemption`

Mencatat **siapa memakai link mana**. Ini yang membuat admin bisa menjawab
"akun ini datang dari undangan mana" — dan bukti pemakaian tidak bisa hilang
walau undangannya dicabut.

```go
type InviteRedemption struct {
    Base
    InviteID  string `gorm:"type:varchar(36);index;not null"`
    UserID    string `gorm:"type:varchar(36);uniqueIndex;not null"`
    Email     string `gorm:"type:varchar(150);not null"`
    UsedAt    time.Time `gorm:"not null"`
}
func (InviteRedemption) TableName() string { return "invite_redemption" }
```

`UserID` unik: satu akun tidak bisa diklaim dua undangan.

### 1.3 Lapor bug

```go
type BugReport struct {
    Base
    UserID   string `gorm:"type:varchar(36);index;not null"`
    Category string `gorm:"type:varchar(24);not null"`
    Title    string `gorm:"type:varchar(120);not null"`
    Body     string `gorm:"type:text;not null"`
    PageURL  string `gorm:"type:varchar(255);not null;default:''"`
    UserAgent string `gorm:"type:varchar(255);not null;default:''"`
    Status   string `gorm:"type:varchar(16);not null;default:'baru'"`
    AdminNote string `gorm:"type:text;not null;default:''"`
}

type BugReportAttachment struct {
    Base
    ReportID  string `gorm:"type:varchar(36);index;not null"`
    FileName  string `gorm:"type:varchar(160);not null"` // nama asli, hanya untuk tampilan
    MimeType  string `gorm:"type:varchar(80);not null"`
    SizeBytes int64  `gorm:"not null"`
    StoredAs  string `gorm:"type:varchar(160);not null"` // UUID, bukan nama asli
}
```

### 1.4 Migrasi

Daftarkan keempat model di:
- `backend/cmd/migrate/main.go` (`models := []any{...}`)
- `backend/internal/model/schema_migrate_test.go` (`allModels()`)

Container `migrate` di compose menjalankan ini sebelum `backend-api`, jadi
`docker compose up` menghasilkan tabel baru tanpa langkah manual.

---

## 2. Backend

### 2.1 `service/invite_service.go`

Antarmuka:

```go
type AdminInviteService interface {
    Create(ctx, actor AdminActor, req dto.CreateInviteRequest) (*dto.CreatedInvite, error)
    List(ctx, filter dto.AdminInviteFilter) ([]dto.InviteItem, error)
    Revoke(ctx, actor AdminActor, id, reason string) error
}

type InviteService interface {
    // Periksa mengembalikan ringkasan untuk halaman daftar. Tidak memakai kuota.
    Periksa(ctx, code string) (*dto.InvitePublic, error)
    // Klaim memakai kuota + membuat akun dalam SATU transaksi.
    Klaim(ctx, code string, req dto.RegisterRequest) (*dto.UserResponse, error)
}
```

`Create`:
1. Validasi: `max_uses >= 1`, `expires_at` di masa depan, `tier` dikenal.
2. Buat kode acak **256-bit** (`crypto/rand`), encode base32 tanpa padding → 52 karakter.
   Dikodekan huruf besar + angka tanpa `0/O/1/I` agar tidak salah ketik.
3. Simpan **hanya** `CodeHash` (SHA-256) + 4 karakter awal sebagai prefix.
4. `CreatedInvite` mengembalikan `Code` **mentah sekali** + `URL` siap salin.
5. Catat audit `invite.create`.

`List`: tidak pernah mengembalikan kode mentah (hanya prefix), plus `used_count`,
sisa kuota, dan status turunan (`aktif`/`habis`/`kedaluwarsa`/`dicabut`).

`Revoke`: set `is_active=false`. Tidak menghapus baris; `invite_redemption` tetap utuh.
Siapa pun yang sudah memakai tidak terpengaruh.

### 2.2 Titik masuk HTTP

| Method | Path | Siapa | Keterangan |
|---|---|---|---|
| `POST` | `/api/v1/admin/invites` | admin (`user.edit`) | buat, balasan memuat kode mentah |
| `GET` | `/api/v1/admin/invites` | admin (`user.edit`) | daftar, tanpa kode mentah |
| `POST` | `/api/v1/admin/invites/:id/revoke` | admin (`user.edit`) | cabut |
| `GET` | `/api/v1/invites/:code` | publik, `RateLimitByIP` | `Periksa`, tanpa memakai kuota |

**Terverifikasi dari `deploy/Caddyfile`:** log akses ditulis oleh Caddy ke
`/var/log/ila/caddy.log` dengan format JSON, dan URI **ikut tercatat**. Jadi kode
di path akan masuk log. Ini **diterima dengan sadar**, karena:

- Yang bisa membaca `/var/log/ila/` adalah root dan proses kontainer; bukan publik.
- Kerugian bila terbaca dibatasi kuota + masa berlaku + bisa dicabut seketika.
  Bukan seperti password: kode yang bocor tidak memberi akses ke akun mana pun.
- Alternatif (header `X-Invite-Code`) membuat log lebih bersih tetapi mengharuskan
  fetch lintas-origin dari halaman publik — menambah CORS dan kerumitan yang
  tidak sebanding.

Bila nanti Anda ingin kode tidak pernah menyentuh log, itu pekerjaan terpisah dan
cukup satu blok `log { format filter { request>uri delete } }` di Caddyfile.
Tidak dikerjakan sekarang karena menghapus URI membuat semua log akses kehilangan
path, sehingga penelusuran masalah lain jadi buta.

`POST /auth/register` menerima field baru opsional:

```go
InviteCode string `json:"invite_code"`
```

Alur di `AuthHandler.Register`:

```
if INVITE_ONLY:
    if invite_code kosong  -> 403 ErrPendaftaranTertutup
    else -> inviteService.Klaim(...)   // validasi ulang penuh di service
else:
    ... alur lama ...
```

**Batas IP:** jalur `Klaim` **tidak** memanggil `registerLimiter` (1 pendaftaran per
email, 5 per IP per jam). Yang tetap menjaga endpoint publik `/api/v1/invites/:code`
adalah `RateLimitByIP` (120 permintaan/menit per IP, bawaannya), yang menahan
banjir percobaan tebak kode tanpa menahan anggota komunitas yang berbagi satu IP.

Grup `auth` sendiri **tidak** memasang `RateLimitByIP` hari ini — batasnya datang
dari `registerLimiter` dan `loginLimiter` di dalam handler. Karena alur invite
melewati `registerLimiter`, TIDAK ADA penyaring IP untuk `POST /auth/register`
lewat jalur ini. Itu disengaja sesuai keputusan Anda, dan alasan mengapa kuota +
masa berlaku adalah batas sesungguhnya. Bila nanti terasa perlu, cukup pasang
`RateLimitByIP` pada grup `auth` — satu baris, tanpa menyentuh alur undangan.

### 2.3 `Klaim` — inti keamanan

**Batasan nyata yang ditemukan saat menyusun plan ini:** repository proyek ini
menyimpan `*gorm.DB` sendiri (`internal/repository/user_repository.go:22`) dan
**tidak** mengambil transaksi dari `context`. Tidak ada pola `txFromContext` di
seluruh `internal/`. Artinya "satu transaksi membungkus `authService.Register`
dan UPDATE kuota" **tidak bisa** dijalankan sebagaimana adanya: `Create` user akan
commit sendiri-sendiri, dan pembatalan kuota tidak akan ikut.

Rombak semua repository ke `tx-from-context` terlalu besar untuk satu fitur dan
menyentuh kode eksekusi. Jadi dipakai **dua penyelesaian yang benar tanpa rombak**:

#### A. Klaim kuota lebih dulu, kembalikan bila gagal

```
1. Kunci baris undangan:  SELECT ... FOR UPDATE            (di dalam tx)
2. Validasi: aktif, belum kedaluwarsa, used_count < max_uses,
   email tujuan cocok (case-insensitive)
3. UPDATE invite_code SET used_count = used_count + 1
   WHERE id = ? AND used_count < max_uses                  (di dalam tx)
   RowsAffected == 0  ->  ErrInviteHabis   (rollback)
4. COMMIT  -> kuota resmi terpakai
5. authService.Register(user)          // di luar tx, jalur registrasi yang sama
6. bila langkah 5 GAGAL:
       UPDATE invite_code SET used_count = used_count - 1 WHERE id = ?
       (kompensasi — inilah "kuota kembali" yang Anda setujui)
7. bila langkah 5 BERHASIL:
       set tier/email_verified/onboarding  +  INSERT invite_redemption
```

Kenapa urutan ini **tetap aman** walau registrasi di luar tx:

- **Kuota tidak bisa terlampaui.** Langkah 3 adalah satu `UPDATE` bersyarat yang
  atomik. Dua orang mengklaim slot terakhir bersamaan: hanya satu yang
  `RowsAffected == 1`; yang lain menerima `ErrInviteHabis`. Ini titik yang
  benar-benar menentukan, dan ia **tidak** bergantung pada transaksi registrasi.
- **Kuota tidak terbakar sia-sia.** Langkah 6 mengembalikannya saat registrasi
  gagal (email sudah terdaftar, legal ditolak, dsb) — persis saran no. 3 Anda.
- **Risiko sisa yang diterima:** antara langkah 4 dan 6, proses bisa mati mendadak
  (crash/OOM) sehingga kuota berkurang tanpa akun. Jendelanya milidetik dan
  dampaknya satu slot. Itu lebih baik daripada alternatif B.

#### B. Yang **tidak** dipakai: bungkus registrasi dalam tx

Menuntut `authService` menerima `*gorm.DB` dari luar, artinya mengubah
`UserRepository`, `LegalService`, `SubscribeDefaults`, dan repo lain yang dipakai
registrasi. Itu menyentuh kode legal dan langganan yang sedang berjalan hanya
untuk satu jendela milidetik. **Tidak sepadan.** Dicatat di sini supaya tidak
ada yang mencobanya lagi tanpa tahu biayanya.

#### Penjagaan tambahan

- `ErrInviteTidakSah` dipakai untuk **semua** kegagalan undangan (tidak ada,
  dicabut, kedaluwarsa, habis) dengan **pesan identik**, supaya orang tidak bisa
  membedakan "kode tidak ada" dari "kode habis" sambil menebak.
- `ErrInviteTidakCocok` terpisah, karena itu bisa diperbaiki pengguna
  ("undangan ini untuk alamat lain").
- Keempat kolom yang diperiksa (aktif, kedaluwarsa, kuota, email) diperiksa
  **lagi** di langkah 2 saat submit, bukan hanya saat halaman dibuka.

### 2.4 Lapor bug

| Method | Path | Siapa |
|---|---|---|
| `POST` | `/api/v1/bug-reports` | user (multipart) |
| `GET` | `/api/v1/bug-reports` | user, miliknya sendiri |
| `GET` | `/api/v1/bug-reports/:id/attachments/:aid` | pemilik ATAU admin |
| `GET` | `/api/v1/admin/bug-reports` | admin (`user.view`) |
| `PATCH` | `/api/v1/admin/bug-reports/:id` | admin (`user.edit`) |

Keamanan lampiran:
- Validasi tipe dari **magic bytes**, bukan `Content-Type` browser
- Nama simpan = UUID; nama asli hanya metadata
- Kirim dengan `Content-Disposition: attachment` + `X-Content-Type-Options: nosniff`
- **Tidak pernah** disajikan sebagai HTML/SVG
- Streaming ke disk (`io.Copy`), tidak dibaca seluruhnya ke memori (limit kontainer 160 MB)

Penyimpanan: `${LOG_DIR_HOST}/bug-report/<report_id>/<stored_as>` — volume yang sudah ada.

---

## 3. Frontend

### 3.1 Halaman daftar dengan undangan

`/daftar?invite=KODE`

| Keadaan | Tampilan |
|---|---|
| Tanpa kode + invite only | Copy "Akses lewat undangan" (sudah dibuat) |
| Kode tidak sah / habis / lewat waktu | Kartu "Undangan tidak berlaku" + tombol ke halaman masuk |
| Kode sah | Formulir daftar, dengan email terisi otomatis bila terikat |

`RegisterForm` menerima prop opsional `inviteCode` dan `emailTerkunci`.
Kirim `invite_code` di payload. Pesan galat dari server ditampilkan apa adanya
(khusus undangan: "Undangan ini sudah dipakai penuh", "Undangan untuk alamat lain").

Tambahkan `sumber` ringan: bila ada `?invite=`, `RegisterPage` tidak menampilkan
copy "invite only" melainkan memverifikasi kodenya lebih dulu. Ini menghindari
anggota komunitas melihat "pendaftaran ditutup" padahal dia memegang link sah.

### 3.2 Halaman admin `/admin/undangan`

Menu baru di grup `Utama` (dekat Users), permission `user.edit`.

- Tabel: prefix kode, tier, kuota (`used/max`), masa berlaku, status, catatan
- Tombol **Buat Undangan**: tier (bawaan plus), kuota (bawaan 1), masa berlaku
  (bawaan 7 hari), email tujuan (opsional), catatan
- Setelah dibuat: modal berisi **link lengkap + tombol Salin**, dengan peringatan
  "link ini tidak bisa dilihat lagi"
- Tombol **Cabut** per baris (wajib alasan, tercatat audit)
- Baris terpakai penuh menampilkan badge `Habis`

Memakai komponen yang sudah ada (`Table`, `Card`, `Modal`, `Badge`, `Can`,
`PageHeader`) — tidak ada gaya baru.

### 3.3 Halaman `/laporkan-bug`

Satu halaman, bukan wizard (formulir pendek tidak butuh langkah):
- Select kategori (wajib): `tampilan`, `sinyal`, `eksekusi`, `akun-broker`,
  `billing`, `login`, `kinerja`, `lainnya`
- Judul (wajib, ≤120)
- Deskripsi (wajib)
- Lampiran opsional (1 berkas, ≤1 MB, png/jpg/webp/pdf)
- `page_url` + `user_agent` diisi otomatis, tidak terlihat
- Riwayat: `/laporkan-bug/riwayat` — daftar laporan sendiri + status

Menu: grup "Bantuan" di sidebar, `/laporkan-bug`.

### 3.4 Admin `/admin/laporan-bug`

- Tabel: tanggal, pelapor, kategori, judul, status
- Filter status + kategori
- Detail: isi laporan, info perangkat/halaman, lampiran (tautan unduh)
- Ubah status (`baru` → `ditinjau` → `selesai` / `ditolak`) + catatan admin

---

## 4. Urutan pengerjaan

Setiap fase: tulis tes → `go build ./...` / `npx tsc --noEmit` → tes → commit.
**Setiap penjaga diuji dengan mutasi:** hapus perbaikannya, pastikan tes MERAH,
lalu kembalikan.

| Fase | Isi | Selesai bila |
|---|---|---|
| **1** | Model invite + migrasi + tes skema | `schema_migrate_test` hijau, tabel terbentuk |
| **2** | `InviteService`: Create/List/Revoke + hash + tes | tes merah saat hash dihapus |
| **3** | `Klaim` + transaksi atomik + tes kuota | tes merah saat `WHERE used<max` dihapus |
| **4** | Endpoint admin undangan + permission + tes router | deklarasi permission teruji |
| **5** | Register menerima `invite_code` + bebaskan limiter | tes: tanpa kode 403, kode sah 201, kode habis 403 |
| **6** | Frontend daftar + verifikasi kode + tes UI | tes merah saat `invite_code` tidak dikirim |
| **7** | Halaman admin undangan + tes | simpan-klik-salin berfungsi |
| **8** | Model + endpoint lapor bug (multipart) | tes magic-bytes, path traversal, akses silang |
| **9** | Halaman lapor bug + riwayat | tes unggah & riwayat |
| **10** | Admin triage laporan | tes ubah status |
| **11** | `docs/AGENT_LOG.md`, build, deploy VPS 2, verifikasi | log bersih, halaman hidup |

---

## 5. Tes yang wajib ada (bukan sekadar hijau)

**Undangan**
1. Kode sah → akun jadi, `tier=plus`, `email_verified_at` terisi, kuota naik 1
2. Kode habis → 403, akun **tidak** dibuat, kuota tidak berubah
3. Kode kedaluwarsa → 403
4. Kode dicabut → 403
5. `max_uses=1` dipakai **dua kali bersamaan** → tepat 1 akun (uji konkurensi)
6. Undangan terikat email, email beda → 403 dengan pesan "alamat lain"
7. Email sudah terdaftar → gagal, **kuota kembali** (kritis — saran no. 3 Anda)
8. Kode mentah **tidak** ada di respons `GET /admin/invites`
9. `max_uses=0` ditolak saat membuat undangan
10. `expires_at` di masa lalu ditolak saat membuat undangan

**Register & limiter**
11. `INVITE_ONLY=true` + tanpa kode → 403 `ErrPendaftaranTertutup`
12. `INVITE_ONLY=true` + kode sah → 201 (limiter IP **tidak** dipanggil)
13. `INVITE_ONLY=false` → alur lama utuh

**Lapor bug**
14. Lampiran `.svg` berisi `<script>` → ditolak (magic bytes)
15. `filename="../../.env"` → nama simpan tetap UUID
16. User A membuka lampiran user B → 403
17. Unduhan membawa `Content-Disposition: attachment` + `nosniff`
18. Unggahan 1,5 MB → pesan "maksimal 1 MB" yang terbaca (bukan 500)

---

## 6. Batas unggah — terkunci oleh Caddy, bukan backend

**Temuan (terverifikasi):** `deploy/Caddyfile` memasang `request_body { max_size 1MB }`
untuk **seluruh** situs. Request lebih besar ditolak Caddy dengan **413 sebelum
sampai backend**. Artinya batas 2 MB yang lazim dipakai tidak akan pernah tercapai.

**Keputusan: batas lampiran = 1 MB**, tanpa mengubah Caddyfile.

Alasan menurunkan batas, bukan menaikkan batas Caddy:
- `max_size 1MB` dipasang sengaja sebagai pengaman umum, bukan angka sembarangan.
  Menaikkannya membuka semua endpoint sekaligus, termasuk yang tidak butuh.
- 1 MB cukup untuk tangkapan layar (PNG 1080p sekitar 300–600 KB) dan PDF satu halaman.
- `mem_limit: 160m` pada `backend-api` tetap jadi alasan kedua untuk menjaga kecil.

Batas ditetapkan di **tiga tempat yang harus sama**:
1. Caddy `max_size 1MB` (tidak diubah)
2. Handler: `http.MaxBytesReader(c.Writer, c.Request.Body, 1<<20)`
3. Frontend: validasi ukuran sebelum unggah + teks bantuan "maks 1 MB"

Tes wajib: unggah 1,5 MB ditolak dengan pesan yang jelas (bukan 500 dari
`MaxBytesReader`), dan pesan itu sampai ke pengguna.

---

## 7. Risiko & penanganan

| Risiko | Penanganan |
|---|---|
| Link bocor ke luar komunitas | Kuota + masa berlaku. Default admin form: kuota 1 |
| Dua orang mengklaim slot terakhir bersamaan | `UPDATE ... WHERE used_count < max_uses` + `RowsAffected` |
| Database bocor | Hanya hash yang tersimpan, bukan kode |
| Anggota komunitas satu IP kantor | Batas pendaftaran per email/IP dibebaskan untuk alur invite |
| Kode masuk log Caddy | Diterima sadar: kuota + masa berlaku + bisa dicabut (§2.2) |
| Lampiran jadi jalur XSS | Magic bytes + `attachment` + `nosniff`, tidak pernah inline |
| `AUTO_EXECUTE` terbuka untuk semua undangan plus | **Anda** memegang rem `AUTO_EXECUTE=false` / `DRY_RUN_ALL=true` di panel admin |

---

## 8. Di luar ruang lingkup (tidak dikerjakan)

- Payment, checkout, invoice, kredit — tetap tertutup `INVITE_ONLY`
- Uji coba — tetap nonaktif
- Email otomatis pengiriman undangan — link disalin manual oleh admin
- Kuota "tanpa batas" — sengaja tidak ada
- Batas IP per anggota — sengaja dibebaskan
- Redaksi URI di log Caddy — bisa ditambah nanti, satu baris

---

## 9. Yang perlu Anda setujui

1. **Plan ini** — boleh saya mulai?
2. **Kode undangan lihat-sekali**: kode mentah hanya tampil saat dibuat, tidak
   bisa dilihat lagi di daftar. Setuju?
3. **Email terverifikasi otomatis** untuk pendaftar lewat undangan
   (memegang link = bukti kepemilikan email). Setuju?

Kalau ketiganya "ya", saya mulai Fase 1 dan jalan terus sampai selesai.
