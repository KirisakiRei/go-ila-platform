# 🤝 AGENT_LOG — Kanal Komunikasi 2 Arah (Human ⇄ Hermes ⇄ Claude)

> **Tujuan:** Media logging perubahan & pesan strategis antara **Human (owner)**,
> **Hermes** (agent engine `go-trade`), dan **Claude** (agent platform
> `go-ila-platform`) melalui repository Git.
>
> **Cara pakai:**
> 1. Entri baru ditulis paling atas di bagian masing-masing.
> 2. Setiap entri **WAJIB** punya: `Tanggal`, `Pelaku`, `Tipe`, `Ringkasan`,
>    `File terdampak`, `Status`.
> 3. Setelah menjalankan tugas, tulis balasan dengan `DIBACA` / `SELESAI` / `BLOKIR`.
>
> **Aturan Emas:**
> - Log ini adalah **catatan, bukan instruksi otomatis**.
> - Satu agent **tidak** mengubah isi bagian agent lain.

---

## 📍 PENANDA SINKRONISASI

> **Commit terakhir saat log ini ditulis:** entri teratas di bawah (gateway pembayaran)
> **Repo ini:** `go-ila-platform` (Runner, Backend, Frontend)
> **Repo engine:** `go-trade` (Hermes) — punya `docs/AGENT_LOG.md` sendiri

### Perintah siap-tempel

```text
Pull lalu jalankan:
  git pull origin main
  git log -8 --oneline
Lalu baca docs/AGENT_LOG.md (entri terbaru di paling atas).
```

---

## ⬆️ DARI CLAUDE KE HUMAN & HERMES

### 2026-11-10 — Sinyal macet, status engine tidak jujur, dan ILA 3.0 disiapkan

**Pelaku:** Claude
**Tipe:** Perbaikan bug + integrase strategi
**Status:** 🟢 SELESAI (kode), ⏳ menunggu deploy

**Ringkasan:**

1. **Penyapu sinyal kedaluwarsa.** Engine hanya mengirim status terminal lewat
   `SignalUpdate`, sehingga `READY_TO_ENTRY` yang tidak pernah dieksekusi
   tertinggal selamanya (163 baris di produksi). `ExpireStaleSignals` kini
   berjalan saat Runner mulai dan setiap siklus monitor. Sengaja hanya
   menyasar `READY_TO_ENTRY`/`OPPORTUNITY`: baris `WAIT_*` memakai identitas
   tetap dan dikirim ulang engine, jadi menandainya `EXPIRED` akan mematikan
   radar satu slot secara permanen (`HandleSignal` melewati status terminal).
2. **Status engine jujur.** `last_seen_at` dan `engine_version` kini ditulis
   dari heartbeat (dibatasi 30 detik), bukan hanya saat koneksi dibuka/ditutup.
   Panel kesehatan tidak lagi menandai engine sehat sebagai stale.
3. **Label "Update" radar.** `dto.SignalResponse` menyertakan `updated_at`
   (waktu baris terakhir diperbarui) terpisah dari `published_at` (waktu
   terbit). UI memakai `updated_at` dengan cadangan `published_at`.
4. **Keepalive penerima log.** Ditambahkan `EnforcementPolicy` (min 10 detik,
   izin tanpa stream) karena engine mengirim ping tiap 30 detik. Sebelumnya
   gRPC menolaknya dengan `too_many_pings` dan stream log terputus tiap ~2
   menit.
5. **Tahap strategi dari katalog.** `Settings.Reload` kini hanya menghitung
   paket yang benar-benar menawarkan strategi (`strategies` memuat
   `ila-3.0`); sebelumnya `coming_soon` di Free/Basic menahan semuanya ke
   `off`. Backend-api memuat ulang tiap menit sehingga perubahan katalog
   berlaku tanpa restart. Runner memakai pengaturan yang sama agar gerbang
   `ComputeRequirements` tidak menyaring habis permintaan ILA 3.0.
6. **Seed paket Plus** memuat `ila-3.0` dengan `ila3_status=general`
   (`DoNothing`, jadi instalasi lama tidak tertimpa).
7. **Topologi ILA 3.0.** `deploy/docker-compose.yml` runner menerima
   `ENGINE3_*`; di sisi engine ditambahkan service `engine3` (port 9444,
   sertifikat sama, nama layanan log `engine3` agar berkas log tidak
   bertabrakan dengan engine ILA 2.0).

**File terdampak:**
- `backend/internal/repository/runner_repository.go` — `ExpireStaleSignals`
- `backend/internal/runner/expiry.go`, `expiry_test.go`
- `backend/cmd/runner/main.go` — sapu saat mulai + siklus monitor, heartbeat
- `backend/internal/runner/service.go` — `EngineVersion()`
- `backend/cmd/runner/logserver.go` — keepalive
- `backend/internal/dto/trading_dto.go`, `trading_service.go` — `updated_at`
- `backend/internal/runner/settings.go`, `settings_reload_test.go`
- `backend/internal/router/router.go`, `cmd/backend-api/main.go` — reloader
- `backend/internal/repository/billing_seed.go`
- `deploy/docker-compose.yml`
- `frontend/src/types/api.ts`, `pages/components/RadarPantauanCard.tsx`

**Catatan deploy:**
- Urutan: engine (ILA 2.0 tetap) → platform → `engine3` + `ENGINE3_GRPC_ADDR`
  pada runner → ubah katalog (`plus.strategies` memuat `ila-3.0` dan
  `plus.ila3_status=general`) → restart runner.
- Runner membaca daftar engine saat mulai, jadi perubahan alamat/tahap butuh
  restart runner; backend-api tidak.

---

### 2026-11-08 (audit) — Dua bug yang ditemukan saat mengaudit sendiri pekerjaan S8

**Pelaku:** Claude
**Tipe:** Perbaikan bug
**Ringkasan:**
Setelah langganan dinamis (S8) selesai, pekerjaan itu diaudit ulang dengan
menguji reconnect berulang. Dua bug nyata muncul yang tidak tertangkap tes
unit.

**File terdampak:**
- `backend/internal/engineclient/client.go` — `OnConnected` menerima ctx
- `backend/cmd/runner/main.go` — `syncLoop` memakai ctx per-koneksi, reset backoff

**Bug 1 — goroutine penyelaras bocor setiap reconnect.**

`go syncLoop(ctx, ...)` memakai konteks **proses**, bukan konteks koneksi.
Setiap reconnect menambah satu goroutine yang **tidak pernah berhenti**.
Setelah seminggu koneksi naik-turun, ratusan goroutine menumpuk.

Perbaikan: `OnConnected` sekarang menerima `callCtx` yang berakhir bersama
stream, dan `syncLoop` memakai konteks itu.

**Bug 2 — jeda reconnect tidak pernah kembali normal.**

`backoff` hanya bertambah dan tidak pernah di-reset setelah koneksi pulih.
Uji membuktikan: setelah beberapa kali putus, jeda tetap di 16–30 detik
walaupun engine sudah hidup lagi, sehingga pemulihan jadi lambat.

Perbaikan: bila sesi bertahan >= 30 detik, jeda dikembalikan ke 1 detik.

**Bukti uji:**
- Reconnect 5x berturut-turut: hanya 2 sesi tercatat, goroutine tidak menumpuk.
- Sesi sehat 40 detik lalu engine dimatikan: jeda kembali `1s` (dulu 16s+).
- `go build ./...` bersih, **22 paket tes lulus**.

### 2026-11-08 (lanjutan) — Langganan simbol dinamis: tutup celah "harus restart Runner"

**Pelaku:** Claude
**Tipe:** Perbaikan bug
**Ringkasan:**
Runner menghitung simbol **sekali** saat koneksi engine dibuka; `Subscribe`
dan `Unsubscribe` tidak pernah dipanggil. Pengguna yang baru berlangganan
sebuah simbol tidak akan dipantau sampai Runner dijalankan ulang. Untuk SaaS
ini tidak bisa diterima, jadi sekarang diselaraskan berkala.

**File terdampak:**
- `backend/internal/runner/service.go` — `SyncSubscriptions`, `SetSymbolSync`
- `backend/cmd/runner/main.go` — `symbolSync`, `syncNow`, `syncLoop`
- `backend/internal/runner/symbol_sync_test.go` (baru)

**Yang dikerjakan:**

1. **Hanya selisih yang dikirim.** `SyncSubscriptions` membandingkan simbol
   yang sudah dikirim dengan kebutuhan saat ini, lalu mengirim `add` dan
   `remove`. Tanpa perubahan, tidak ada panggilan ke engine.
2. **Penyelarasan berkala 2 menit** selama stream hidup.
3. **Selaras ulang saat reconnect**: daftar yang dianggap terkirim dikosongkan,
   karena engine bisa sudah lupa setelah putus.
4. **Gagal kirim tidak dianggap terkirim**, sehingga percobaan berikutnya
   mengulang daftar yang benar.

**Bukti uji end-to-end** (engine dan Runner berjalan bersamaan, PostgreSQL):
```
17:01:40  engine menerima  ["XAUUSD"]
          EURUSD ditambahkan ke database — Runner TIDAK di-restart
17:03:40  engine menerima  ["EURUSD"]
          EURUSD dinonaktifkan
17:06:09  engine mencabut  ["EURUSD"]
```

**Tes:**
- `go build ./...` bersih, **22 paket tes lulus**.
- Tes baru: selisih tambah/cabut, tidak mengirim bila sama, mengulang setelah
  gagal, mengulang penuh setelah reconnect, aman tanpa engine.

**Sisa pekerjaan:** ILA 3.0 belum diuji dengan data pasar nyata.

### 2026-11-08 — Dukungan sinyal ILA 3.0 dan dua bug PostgreSQL di jalur sinyal

**Pelaku:** Claude
**Tipe:** Fitur + perbaikan bug kritis
**Ringkasan:**
Engine (`go-trade`) kini menerbitkan sinyal lewat gRPC. Kontrak protobuf
ditambah status ILA 3.0, dan Runner diajari menerimanya. Uji integrasi dengan
engine sungguhan memunculkan dua bug PostgreSQL yang membuat Runner **tidak
pernah bisa mengirim sinyal ke user mana pun**.

**File terdampak:**
- `proto/engine/v1/engine.proto` — status ILA 3.0 + field detail
- `backend/gen/engine/v1/` — hasil generate ulang
- `backend/buf.gen.yaml` (baru) — konfigurasi generate
- `backend/internal/runner/ingest/convert.go` — pemetaan status ILA 3.0
- `backend/internal/runner/ingest/convert_test.go` — tes pemetaan
- `backend/internal/repository/runner_repository.go` — kutip `"user"`
- `backend/internal/model/broker_account.go` — `Subscription.IsActive`
- `backend/internal/repository/runner_query_test.go` (baru) — tes PostgreSQL

**Yang dikerjakan:**

1. **Sinyal ILA 3.0 diterima.** Dulu `statusName()` hanya mengenal 5 status
   ILA 2.0. Sinyal ILA 3.0 akan dibuang tanpa jejak. Sekarang
   `READY_ENTRY_1` dan `READY_ENTRY_2` dipetakan ke `READY_TO_ENTRY` (boleh
   dieksekusi), sedangkan status menunggu dikirim sebagai teks untuk dashboard.
   Detail asli tersimpan di `sequence_state`.

2. **Bug: JOIN ke `"user"` tanpa kutip.** `user` adalah kata kunci
   PostgreSQL. `ListUsersBySymbol` gagal dengan `SQLSTATE 42601`, sehingga
   setiap sinyal berhenti di tahap "membaca pemantau simbol". Tidak terlihat
   di SQLite.

3. **Bug: langganan dicabut hidup lagi.** `Subscription.IsActive` bertag
   `default:true`, membuat GORM mengirim `true` saat nilainya `false`.

**Bukti uji:**
- `go build ./...` bersih, **22 paket tes lulus**.
- `runner_query_test.go` dijalankan ke PostgreSQL 16 sungguhan
  (`TEST_POSTGRES_DSN`). Kedua bug dibuktikan tertangkap: kode dirusak
  sementara, tes gagal, lalu dipulihkan.
- Integrasi penuh dengan engine: sinyal ILA 2.0 terbit → tersimpan di `signal`
  → `user_signal` berstatus `READY_TO_ENTRY`.
- Kotak sinyal (TTL, supersede, cancel, expire) diuji di sisi engine.

**Belum dikerjakan (WAJIB):**
- **Runner belum memanggil `Subscribe`/`Unsubscribe` berkala.** Simbol hanya
  dihitung sekali saat koneksi dibuka, jadi user baru subscribe simbol harus
  menunggu Runner restart.
- ILA 3.0 belum diuji dengan data pasar nyata; engine-nya baru sampai tahap
  adapter.

### 2026-11-14 (lanjutan) — Gateway pembayaran bisa ditukar, siklus tagihan ditutup

**Pelaku:** Claude
**Tipe:** Perbaikan arsitektur + bug uang
**Ringkasan:**
Alur uang diaudit dari checkout sampai tier naik. Penyedia pembayaran kini
berada di balik satu antarmuka sehingga mengganti penyedia cukup dengan satu
nilai env. Empat lubang siklus tagihan ditutup, dan satu bug ditemukan justru
oleh tes yang ditulis untuk menguji perbaikan itu.

**File terdampak:**
- `backend/internal/payment/registry.go` (baru), `registry_test.go` (baru)
- `backend/internal/payment/manual.go` (baru)
- `backend/internal/config/config.go` (`PaymentProvider`, default `pakasir`)
- `backend/internal/router/router.go` (dua pemanggilan diganti `payment.NewFromConfig`)
- `backend/internal/repository/billing_repository.go`, `billing_store.go` (`ExpireOverdue`)
- `backend/internal/service/billing_service.go` (`closeInvoice`, `RunGraceSweep`)
- `backend/internal/service/admin_billing_service.go` (`RefundGateway`)
- `backend/internal/service/admin_billing_money_service.go` (tolak tagihan mati)
- `backend/internal/service/billing_invoice_test.go` (baru)
- `backend/docs/AUDIT-BILLING.md` (baru)
- `.env.example` (`PAYMENT_PROVIDER`, kredensial pakasir)

**Yang dikerjakan:**

1. **Penyedia bisa ditukar.** `PAYMENT_PROVIDER=pakasir|manual`. Provider tak
   dikenal jatuh ke `manual` dan dicatat di log — sistem tetap hidup, tidak
   crash. Menambah penyedia baru = tambah satu file di `internal/payment`,
   daftarkan namanya, isi env. Service billing dan halaman tidak tersentuh.
2. **Adapter `manual`.** Untuk keadaan penyedia berhenti: tagihan tetap terbit,
   lalu ditutup admin lewat catat pembayaran manual. Checkout membalas
   `503` dengan pesan "gunakan catat pembayaran manual" — bukan `500` yang
   membuat pengguna mengira sistemnya rusak.
3. **Tagihan mati ditutup.** Invoice `pending` yang lewat tenggat menjadi
   `expired` di penyapu per jam. Webhook `expired`/`cancelled` ikut menutup
   invoice, tetapi invoice **lunas tidak pernah** dibatalkan webhook terlambat.
4. **Penolakan tagihan mati.** Catat bayar manual pada invoice `expired` atau
   lewat tenggat ditolak `409`, karena periode di tagihan lama sudah tidak
   berlaku. Solusinya terbit tagihan baru.
5. **Refund diteruskan.** `RequestRefund` kini benar-benar dipanggil ke penyedia
   sebelum refund dicatat. Bila penyedia menolak, seluruh refund dibatalkan —
   tidak ada catatan setengah jadi.

**Bug yang ditemukan saat menguji perbaikan:**

`closeInvoice` mencatat audit tanpa `Reason`. Pencatatan audit menolak alasan
kosong, sehingga setiap webhook `expired`/`cancelled` membalas galat dan
invoice tertahan `pending` selamanya. Perbaikannya: alasan diisi
("gateway melaporkan tagihan <status>"), dikunci oleh
`TestWebhookBatalMenutupInvoicePending`.

**Bukti uji:**
- `go build`, `go vet`, `gofmt` bersih.
- **21 paket tes lulus.**
- PostgreSQL 16 sungguhan: checkout+`manual` → `503`, tanpa invoice yatim;
  webhook tanda tangan palsu → `401`, tabel `payment` tetap kosong; penyapu
  mengubah `pending`→`expired`; catat bayar pada invoice `expired` → `409`.
- Tes baru dibuktikan menangkap regresi: kondisi penolakan dihapus sementara,
  tes gagal, lalu kode dipulihkan.

**Belum dikerjakan:**
- Penyedia pengganti Pakasir belum dipilih; adapter baru menunggu keputusan.
- Halaman admin belum menampilkan penyedia mana yang aktif.
- Refund tetap tidak menurunkan tier secara otomatis (keputusan bisnis).

**Status:** selesai, siap dipakai; bergantung pada pemilihan penyedia baru.

### 2026-11-14 (lanjutan) — Trial Basic, halaman admin trial, dan tiga bug yang hanya muncul di PostgreSQL

**Pelaku:** Claude
**Tipe:** Fitur + perbaikan bug berat
**Ringkasan:**
Trial diubah menjadi paket Basic dengan satu akun broker, halaman admin trial
ditambahkan, dan migrasi diuji ke PostgreSQL sungguhan (Docker). Uji itu
memunculkan tiga bug yang selama ini tidak terlihat karena semua tes memakai
SQLite.

**File terdampak:**
- `backend/internal/model/user_admin.go` (`TierTrial`, `MaxAccountsDuringTrial`)
- `backend/internal/model/hooks.go` (ditulis ulang, satu rumah hook ID)
- `backend/internal/model/id_guard.go` (baru)
- `backend/internal/model/subscription.go`, `billing.go` (kolom eksplisit, indeks gabungan)
- `backend/internal/service/admin_trial_service.go` (baru)
- `backend/internal/handler/admin_trial_handler.go` (baru)
- `backend/internal/service/account_service.go`, `trial_service.go`
- `backend/internal/runner/rules.go`, `trial.go`
- `backend/internal/repository/user_repository.go`, `interfaces.go`
- `backend/internal/dto/trial_dto.go`, `admin_user_dto.go`
- `backend/internal/service/billing_service.go` (penyebab gateway ikut terbawa)
- `frontend/src/pages/admin/AdminTrialsPage.tsx` (baru)
- `backend/internal/model/column_name_test.go` (baru, tes regresi)

**Yang dikerjakan:**

1. **Trial = Basic, satu akun.** Sebelumnya trial diperlakukan setara Plus.
   Sekarang `model.TierTrial = TierBasic` dan `MaxAccountsDuringTrial = 1`.
   Batas akun ditegakkan di `account_service` saat menghubungkan akun broker.
   Konsekuensi yang benar: mode eksekusi `auto` **tidak** berlaku selama trial,
   karena mode itu memang milik Plus.
2. **Halaman admin trial** (`/admin/trial`): daftar berjalan/berakhir,
   perpanjangan 1–30 hari beralasan, dan penghentian lebih awal. Trial tetap
   sekali per akun; admin tidak bisa membuka trial baru.
3. **Backup tetap di VPS.** Belum pakai object storage; salinan kedua ditulis ke
   direktori lain di VPS yang sama. Batasannya dicatat di `deploy/ENV-SETUP.md`.

**Tiga bug yang hanya muncul di PostgreSQL:**

| Bug | Kenapa lolos selama ini | Perbaikan |
|---|---|---|
| Semua kolom `*IDR` menjadi `*_id_r` (`amount_id_r`) | SQLite mengabaikan beda nama kolom | tag `column:` eksplisit 18 kolom + tes regresi |
| ID kosong pada tabel non-`Base` (role, invoice, sesi, …) | SQLite menerima string kosong sebagai PK | `hooks.go` jadi satu rumah `BeforeCreate`; migrasi gagal total sebelumnya |
| Indeks unik `(gateway, gateway_txn_id)` tidak terbentuk | GORM hanya membuat indeks kolom terakhir | kedua kolom diberi `uniqueIndex:uq_payment_txn`; `ON CONFLICT` kembali bekerja |

Bug ketiga membuat **pencatatan pembayaran gagal** di PostgreSQL
(`42P10 no unique or exclusion constraint`). Di SQLite tidak pernah muncul
karena ON CONFLICT diabaikan.

**Bukti uji di PostgreSQL 16 (Docker, database baru):**
- `cmd/migrate` sukses (47 tabel), dijalankan ulang tetap sukses (idempoten)
- register → login → checkout (gateway tiruan) → tagihan terbit
- admin catat pembayaran manual → tier naik ke `basic`, berakhir +1 bulan
- catat ulang → ditolak `invoice sudah lunas`
- refund penuh → invoice `refunded`, tier **tidak** turun (sesuai keputusan)
- kode kredit: buat → tukar (saldo 50.000) → tukar ulang ditolak
- trial: mulai → perpanjang 3 hari → hentikan
- ekspor posisi CSV: 200, header WIB+NY lengkap

**Catatan untuk Hermes:**
Tidak ada perubahan kontrak engine → Runner. Aturan harian, pengecekan
slippage, dan status sinyal tetap. Trial hanya menurunkan hak tier pengguna.

**Belum dikerjakan:**
- Salinan backup ke luar VPS (object storage) — sengaja ditunda.
- Migrasi ke PostgreSQL di VPS 2 belum dijalankan; yang diuji baru di lokal.

**Status:** SELESAI

---

### 2026-11-14 — Kredit, tagihan manual, refund, ekspor riwayat, backup terjadwal

**Pelaku:** Claude
**Tipe:** Fitur + perbaikan celah
**Ringkasan:**
Menutup empat hal yang sebelumnya hanya setengah jadi, plus dua sumber
kebenaran ganda yang bisa membuat panel dan Runner berbeda angka.

**File terdampak:**
- `backend/internal/repository/billing_store.go` (baru — satu data uang)
- `backend/internal/repository/credit_code_repository.go`
- `backend/internal/repository/billing_repository.go`
- `backend/internal/service/admin_billing_service.go`
- `backend/internal/service/admin_billing_money_service.go` (baru)
- `backend/internal/service/billing_service.go`
- `backend/internal/service/trading_export.go` (baru)
- `backend/internal/service/system_service.go`
- `backend/internal/handler/admin_billing_handler.go`
- `backend/internal/handler/trading_handler.go`
- `backend/internal/dto/billing_dto.go`, `billing_flow_dto.go`
- `backend/internal/runner/settings.go`
- `backend/internal/audit/catalog.go`
- `backend/cmd/runner/main.go`
- `backend/internal/router/router.go`, `audit_route_test.go`
- `frontend/src/pages/admin/AdminInvoicesPage.tsx` (baru)
- `frontend/src/pages/admin/AdminCreditCodesPage.tsx` (baru)
- `frontend/src/features/auth/components/LegalReconsentGate.tsx` (baru)
- `frontend/src/pages/BillingPage.tsx`, `PositionsPage.tsx`
- `frontend/src/lib/http.ts` (`downloadFile`), `admin-api.ts`, `auth-api.ts`
- `frontend/src/components/layout/AppShell.tsx`
- `deploy/docker-compose.yml`, `deploy/ENV-SETUP.md`
- `scripts/backup.sh`, `scripts/restore.sh`

**Yang dikerjakan:**

1. **Refund + catat pembayaran manual.** Sebelumnya `AddRefund` ada tetapi
   tidak pernah dipanggil, dan `RequestRefund` hanya stub kosong. Sekarang ada
   `POST /admin/invoices/:id/mark-paid` dan `POST /admin/invoices/:id/refund`.
   Mark-paid menolak nominal di bawah total tagihan dan memberi hak paket lewat
   jalur yang sama dengan webhook (`BillingService.ActivatePlan`). Refund
   menolak total melebihi nilai pembayaran; refund **tidak** menurunkan tier
   otomatis karena mencabut akses adalah keputusan bisnis terpisah.
2. **Ekspor riwayat posisi user.** `GET /api/v1/positions/export` dengan filter
   `from`/`to` pada zona hari akuntansi WIB, kolom waktu WIB dan NY, batas
   10.000 baris, PnL kosong bila belum terbaca broker.
3. **Kode kredit.** Tabel, service, dan `Redeem` sudah ada tetapi admin tidak
   punya cara membuat kode. Sekarang ada daftar, buat, dan hentikan
   (`/admin/credit-codes`), plus form tukar kode di halaman Langganan user.
   Kode tidak dihapus, hanya dihentikan, agar riwayat penukaran utuh.
4. **`ila3_status` dibaca dari `plan_feature`.** Sebelumnya Runner membaca
   `ILA3_AVAILABILITY` dari env sementara panel membaca katalog paket: dua
   sumber untuk satu kebenaran. Sekarang `Settings.Reload` menurunkan tahap dari
   katalog paket dan env hanya cadangan bila baris kosong.
5. **Zona hari akuntansi.** `system_service.go` menyalin `runner.WIB` saat
   init, sebelum `SetDailyTimezone` dipanggil, dan `daily_timezone` di DTO
   hardcode. Sekarang dibaca saat dipakai.
6. **Modal setuju ulang dokumen.** `pending` dari `/me/legal` kini muncul
   sebagai dialog wajib di dashboard user; selama belum disetujui, eksekusi dan
   pembayaran memang ditahan server.
7. **Backup terjadwal.** Layanan `backup` di compose menjalankan backup tiap
   hari 18:00 UTC, retensi 7 harian + 4 mingguan, salinan luar VPS lewat
   `BACKUP_REMOTE_DIR`. `restore.sh` tidak lagi memakai `--clean --if-exists`
   (bertentangan dengan RUNBOOK §5) dan menolak target yang sudah berisi tabel
   inti.
8. **`ExtendTier` tidak memotong periode.** Tanggal berakhir hanya boleh
   memanjang, sehingga pencatatan berikutnya tidak memakan hak yang sudah dibayar.

**Bug kecil yang ikut diperbaiki:**
- `adminBillingService` sebelumnya hanya memegang `pricingRepo`, sehingga
  invoice,pembayaran, dan voucher tidak bisa dijangkau dari satu tempat.
  Sekarang satu `repository.BillingStore` menaungi semuanya.
- Tes `TestSetiapMutasiAdminPunyaAksiAudit` diperluas untuk route uang baru;
  ini yang menangkap route tanpa jejak audit.

**Catatan untuk Hermes:** tidak ada perubahan pada kontrak engine → Runner.
Sinyal, aturan harian, dan pengecekan slippage tetap. Yang berubah hanya sisi
platform (uang dan tampilan).

**Verifikasi:**
- `gofmt -l .` kosong, `go vet ./...` bersih
- `go test ./...` → 21 paket lulus
- `tsc --noEmit`, `eslint --max-warnings 0`, `vitest` 39 tes, `vite build` sukses

**Belum dikerjakan:**
- Refund belum otomatis menurunkan tier (sengaja, keputusan bisnis).
- Migrasi PostgreSQL sungguhan masih menunggu VPS 2; semua tes skema via SQLite.
- Pengaturan panggilan salinan luar VPS masih lewat variabel mount; integrasi
  object storage (rclone/S3) belum ada.

**Status:** SELESAI

---

### 2026-11-13 (lanjutan 4) — Audit menyeluruh Fase A–P dan perbaikan temuan berat

**Pelaku:** Claude
**Tipe:** `SELESAI`
**Status:** Delapan temuan berat diperbaiki dan diuji. Enam butir belum
dikerjakan dan dicatat di bawah.

**Temuan yang diperbaiki**

| # | Masalah | Perbaikan |
|---|---|---|
| 1 | **Webhook pembayaran bisa dipalsukan.** Tanda tangan HMAC dihitung dari isi body yang dikirim penyerang, jadi siapa pun yang tahu format `order_id.amount.status` bisa menaikkan tier gratis. | Pakasir **tidak mengirim header tanda tangan** (panduan resmi §D). Verifikasi kini memanggil `GET /api/transactiondetail` dan membandingkan **nominal + status** dengan konfirmasi gateway. Header tanda tangan tetap dihormati bila ada, tetapi bukan lagi bukti utama. |
| 2 | **Token reset/verifikasi email tersimpan mentah** di `email_outbox.payload` — siapa pun yang membaca tabel atau backup bisa mengambil alih akun. Komentar kode justru menyatakan sebaliknya. | Payload hanya memuat `token_id`. Tautan dibentuk saat pengiriman dari token yang dititipkan di memori (`EnqueueWithToken` → `pendingTokens`). Token hangus bila proses berhenti sebelum terkirim. TTL reset juga dibetulkan 1 jam → **30 menit**. |
| 3 | **2FA tidak menaikkan `session_version`.** Token yang dicuri sebelum 2FA dipasang tetap sah sampai 12 jam. | `Activate`, `Disable`, dan `RegenerateRecovery` kini mencabut seluruh sesi lama. |
| 4 | **User berstatus `deleted` masih dieksekusi** Runner bila subscription/akun broker tertinggal. | `evaluate` menolak `deleted` dan `DeletedAtUser != nil`. |
| 5 | **Registrasi dari UI selalu gagal 400**: backend mewajibkan `accept_terms`, form tidak mengirimnya. | Checkbox persetujuan (TOS, Privasi, Penggunaan Wajar) + skema zod + payload `accept_terms`. |
| 6 | **C2: `plan_feature` tidak pernah dibaca.** Admin mengubah `account_limit`/`max_symbols`, penegakan tetap memakai angka di kode. | `TierFeatureReader` baru; `tierAccountLimit` dan `tierSymbolLimit` membaca katalog paket dengan cadangan kode. |
| 7 | **`RunGraceSweep` tidak pernah dipanggil** — tier kedaluwarsa tetap Plus selamanya. | `BillingSweeper` berjalan tiap jam dari `cmd/backend-api`. |
| 8 | **Pagar harga gagal-terbuka**: paket tanpa baris `pricing_guard` meloloskan semua diskon. | Pagar kini tertutup: paket berbayar tanpa batas bawah **ditolak** saat checkout. |
| 9 | **A17: peringatan status pasar tidak terpasang.** Engine yang mengaku buka di Sabtu tidak pernah terdeteksi. | `Service.checkSchedule` memanggil `ScheduleWatch`; anomali diteruskan lewat `Options.OnMarketAnomaly` ke `alert.MarketScheduleAnomaly`. |
| 10 | Admin bisa mem-ban admin lain. | `Ban` menolak target ber-role admin; tombol di UI disembunyikan. |
| 11 | Kupon bisa dipakai dua kali oleh user yang sama saat checkout bersamaan. | Unique index `uq_coupon_user (coupon_id, user_id)`. |

**Belum dikerjakan — butuh keputusan atau lingkungan VPS**

| Butir | Alasan |
|---|---|
| A15 — heartbeat sesi user + `idle_expires_at` | Panel admin sudah punya; versi user perlu perubahan model `session` dan middleware. Belum dikerjakan. |
| C7 — modal "setuju ulang" saat dokumen versi baru | Data backend sudah tersedia (`Pending`), UI belum ada. |
| F6/F8 — halaman kredit & `/admin/kredit`, kode kredit | Endpoint dan halaman belum ada. |
| G8/G9 — halaman tagihan & trial admin, refund | `AddRefund` sudah ada tetapi tidak punya endpoint. |
| H7 — ekspor riwayat trade user (C5) | Belum ada endpoint. |
| K — jadwal backup (cron) dan salinan luar VPS | Butuh VPS 2. |

**Catatan: dua temuan audit yang TIDAK terbukti**

1. Dilaporkan `AdminUserFilter.From/To` bertipe `string` sehingga `go build`
   gagal. **Salah** — `params.From` bertipe `*time.Time` dan build lulus.
2. Dilaporkan `Count` mengabaikan filter. Sudah diperiksa: filter yang sama
   dipakai, hanya `HasAccounts` memang tidak ikut (perlu diperbaiki terpisah).

**Verifikasi**

`gofmt` bersih, `go vet` bersih, **21 paket tes backend lulus**. Tes baru:
`TestWebhookPalsuDitolak`, `TestWebhookNominalBedaDitolak`,
`TestGatewayTidakTerjangkauDitolak`, `TestPerubahanTwoFactorMencabutSesiLama`,
`TestUserTerhapusTidakDieksekusi`, `TestAnomaliJadwalPasarlDilaporkan`,
`TestGraceSweepDijalankan`, `TestPagarHargaKosongMenolakPaketBerbayar`.
Frontend: `tsc`/`eslint` bersih, **39 tes vitest lulus** (1 tes baru untuk
persetujuan pendaftaran), build dua entri sukses.

**File terdampak (utama)**

`backend/internal/{payment/{pakasir,gateway},service/{billing_service,email_account_service,account_service,pricing_service,twofactor_service,admin_user_service},email/worker,repository/{email_repository,billing_seed,billing_repository},model/{billing,user},runner/{service,schedule},alert/alert,router/router}.go`,
`backend/cmd/{backend-api,runner}/main.go`,
`frontend/src/{features/auth/{schemas,components/RegisterForm},types/api,pages/admin/AdminUserDetailPage}.tsx`

---

### 2026-11-13 (lanjutan 3) — Token akun fleksibel tanpa risiko eksekusi ganda

**Pelaku:** Claude
**Tipe:** `SELESAI`
**Status:** Semua selesai dan teruji.

**Konteks dari pemilik**

Token broker harus bisa diganti kapan saja oleh user, tetapi **satu token = satu
akun broker**. Bila token yang sama dipakai dua entri akun, sinyal yang sama akan
dieksekusi dua kali ke akun yang sama — risiko user menjadi dua kali lipat dari
yang ia niatkan.

**Yang diubah**

| Bagian | Perubahan |
|---|---|
| Ganti token | `PATCH /accounts/:id` menerima `token` + `base_url`. Token baru dienkripsi, lalu sesi lama di Runner diputus supaya login berikutnya memakai token terbaru. |
| Deteksi token kembar (backend) | `Credential.TokenFingerprint` (SHA-256 bertanda `INTERNAL_SECRET`) disimpan saat token ditulis. Pendaftaran **dan** rotasi menolak token yang sudah dipakai akun lain milik user yang sama. |
| Deteksi akun kembar (Runner) | `executableAccounts` membuang entri dengan `broker_type + external_account_id` sama, jadi satu sinyal tetap satu order. |
| Idempotensi ke broker | Order Match-Trader kini mengirim `clientId`. Bila pengiriman gagal di tengah (timeout), Runner memeriksa posisi terbuka dengan komentar = id sinyal sebelum menyimpulkan gagal. |
| UI | Kartu akun punya tombol **Ganti token**; token dikirim sekali lalu dibuang dari state. |

**Sikap fail-closed**

Bila `INTERNAL_SECRET` kosong, pemeriksaan duplikat **menolak** pendaftaran akun
alih-alih melewatkannya. Tanpa kunci, duplikat tidak bisa dibuktikan, dan
membiarkannya berarti risiko eksekusi ganda yang tak terdeteksi.

**Batas yang perlu diketahui Hermes**

- `clientId` dan pembacaan `comment` posisi bergantung pada dukungan Match-Trader.
  Bila broker mengabaikannya, lapisan yang masih melindungi adalah klaim eksekusi
  (`IdempotencyKey`) dan penyaringan akun kembar di Runner.
- Token kembar **antar user berbeda** tidak diblokir; pemeriksaan berjalan per user.
  Bila ingin lintas user, perlu keputusan bisnis tersendiri.

**Verifikasi**

`gofmt`/`go vet` bersih, **21 paket tes backend lulus**. Tes baru:
`TestTokenSamaDitolak`, `TestRotasiTokenMemutusSesiLama`,
`TestUpdateTanpaTokenTidakMenyentuhKredensial`, `TestAkunKembarHanyaSekali`,
`TestHeartbeatTanpaSesiDitolak`, `TestHeartbeatMemperpanjangCookie`.
Frontend: `tsc`/`eslint` bersih, **38 tes vitest lulus** (4 tes baru untuk idle admin).

**File terdampak (utama)**

`backend/internal/{service/account_service,service/credential_fingerprint,broker/matchtrader/{client,types},broker/broker,runner/service,model/broker_account,handler/account_handler,repository/broker_repository,router/router}.go`,
`backend/internal/{service,runner,handler,model}/*_test.go`,
`frontend/src/{types/api,pages,features/accounts/components/AccountCard}.*`,
`frontend/src/test/admin-idle-timer.test.ts`

---

### 2026-11-13 (lanjutan 2) — Enam celah keamanan & konsistensi ditutup

**Pelaku:** Claude
**Tipe:** `SELESAI`
**Status:** Semua butir selesai dan teruji. Sisa dua butir butuh VPS 2.

**Yang diperbaiki**

| Celah | Bug | Perbaikan |
|---|---|---|
| Rotasi kredensial broker | Token Match-Trader tidak bisa diganti dari dashboard. Akun yang tokennya kedaluwarsa mati permanen sampai Runner di-restart. | `PATCH /accounts/:id` menerima `token` + `base_url` opsional. Token baru dienkripsi lalu sesi lama diputus lewat `/internal/account/reload` sehingga login berikutnya memakai token terbaru. |
| Sesi admin kedaluwarsa di tengah kerja | Tidak ada perpanjangan sesi. Admin yang bekerja >2 jam terlempar keluar. | `POST /api/v1/admin/auth/heartbeat` memperpanjang cookie **hanya** bila `AdminScope` sudah meloloskan token. Panel admin: diam 10 menit → modal 60 detik → keluar. Batas diatur `ADMIN_IDLE_TIMEOUT`. |
| Penyiapan 2FA hilang saat refresh | Secret TOTP disimpan di state React. Refresh = harus scan ulang. | Disimpan di `sessionStorage` (bukan localStorage) dengan cek `expires_at`; tombol **Batal** untuk membuang penyiapan. |
| Filter admin tidak lengkap | Halaman pengguna hanya punya status, tier, dan akun broker. Ekspor CSV mengabaikan filter. | Tambah filter trial, peran, dan rentang tanggal. Ekspor mengikuti filter aktif. |
| Batas akun hard-code | `limit > 2` di Runner adalah angka ajaib. | Diganti konstanta `maxAccountsPerUser`. |
| Kolom `RunnerDirtyAt` tanpa pengisi | Sempat dianggap tidak terpakai. | Sudah diisi di 4 jalur (ubah tier admin, aktivasi invoice, penurunan kedaluwarsa, anonimisasi). Runner memang membaca ulang user per sinyal, jadi kolom ini dipakai untuk **telusur waktu perubahan**, bukan pemicu muat ulang. |

**Catatan penting untuk Hermes**

- **Kredensial kini bisa diperbarui.** Sebelumnya satu-satunya jalan adalah hapus akun lalu buat lagi.
- **Runner punya endpoint baru `/internal/account/reload`** (selain `/internal/account/revoke`).
  Keduanya wajib membawa `X-Internal-Secret` dan hanya terbuka di jaringan container.
- **Aturan diam admin terpisah** dari peringatan diam pelanggan. Admin lebih ketat
  karena panelnya bisa mengubah tier dan memblokir akun.

**Belum bisa dikerjakan dari sini**

- Uji pemulihan backup (`scripts/restore.sh`) dan migrasi PostgreSQL nyata:
  keduanya butuh VPS 2 yang berjalan.

**Verifikasi**

`gofmt` bersih, `go vet` bersih, **20 paket tes backend lulus** (termasuk 2 tes rotasi
kredensial dan tes heartbeat tanpa sesi). Frontend: `tsc` bersih, `eslint --max-warnings 0`
bersih, 34 tes vitest lulus, build dua entri sukses.

**File terdampak (utama)**

`backend/internal/{service/account_service,handler/admin_auth_handler,runnerclient/client,runner/service,model/user,config/config}.go`,
`backend/cmd/runner/server.go`, `backend/internal/router/{router,router_test,audit_route_test}.go`,
`backend/internal/service/account_rotate_test.go`,
`frontend/src/{lib/admin-api,app/AppRoutes,pages/SecuritySettingsPage,pages/admin/AdminUsersPage}.tsx`,
`frontend/src/features/admin/{hooks/use-admin-idle-timer,components/AdminIdleGuard}.ts*`,
`.env.example`

---

### 2026-11-13 (lanjutan) — Sisa pekerjaan Fase I/J/K/P diselesaikan

**Pelaku:** Claude
**Tipe:** `SELESAI`
**Status:** Seluruh butir yang sebelumnya tertunda sudah dikerjakan.

**Sisa pekerjaan yang kini selesai**

| Butir | Yang dikerjakan |
|---|---|
| Halaman `/pengaturan/keamanan` | Verifikasi email (kirim ulang), ganti sandi, 2FA TOTP + kode pemulihan, daftar sesi + cabut, toggle peringatan diam |
| Endpoint baru | `PATCH /api/v1/me/idle-warning` — hanya mengubah tampilan dialog, **tidak** memperpanjang sesi |
| Halaman `/admin/dukungan` | Jurnal dukungan per pelanggan: cari user, baca catatan, tulis catatan baru dengan saluran & sematan |
| Halaman admin dipisah | `/admin/engine`, `/admin/runner`, `/admin/kesehatan-broker` + endpoint `GET /admin/system/{engine,runner,brokers}` |
| Jadwal pasar di UI | `runner.ExpectedOpen` diekspor; `GET /api/v1/status` kini memuat `market` (buka/tutup, perubahan berikutnya dalam WIB dan NY). Kartu "Sesi pasar & kuota harian" tampil di dashboard |
| Rotasi log | Sudah ada di `deploy/docker-compose.yml` (`max-size 10m`, `max-file 2`) untuk semua layanan |
| Uji skema database | `internal/model/schema_migrate_test.go` — AutoMigrate 47 model di SQLite + pemeriksaan kolom kunci |
| Penyapu rahasia di log | `pkg/redact` diperluas (`master_key`, `smtp_pass`, `client_secret`, `pass`) + `redact_sweep_test.go` menyapu seluruh sumber `.go` |

**Keputusan yang perlu diketahui Hermes**

1. **`marketWindow` memakai `runner.ExpectedOpen`**, bukan salinan aturan.
   Aturan tetap: tutup Sabtu & Minggu; Jumat tutup sejak 17:00 NY; Minggu
   buka 17:00 NY. Perubahan jadwal dicari per jam dalam rentang 8 hari.
2. **Peringatan diam bersifat tampilan.** Mematikannya tidak memperpanjang
   sesi; batas diam tetap ditegakkan server.
3. **Kesehatan Runner disimpulkan dari laporan engine terakhir.** Tidak ada
   tabel denyut terpisah, jadi jangan hapus baris `engine_state` — Runner akan
   terlihat mati sampai laporan berikutnya masuk.
4. **99 tes backend lulus**, 20 paket, `gofmt`/`go vet` bersih; frontend
   `tsc`/`eslint` bersih, 34 tes vitest lulus, build dua entri sukses.

**Masih belum bisa dilakukan di sini**

- Migrasi ke **PostgreSQL sungguhan** belum dijalankan: Docker Desktop dan
  `psql` tidak tersedia di mesin ini. Skema sudah divalidasi lewat AutoMigrate
  SQLite, tetapi jalankan `go run ./cmd/migrate` sekali di VPS 2 untuk
  memastikan tipe kolom PostgreSQL (khususnya `jsonb`/`filter` query) sesuai.

**File terdampak (utama)**

`backend/internal/service/{system_ops,auth_service}.go`,
`backend/internal/handler/{admin_system,auth}_handler.go`,
`backend/internal/runner/schedule.go`, `backend/internal/dto/system_setting_dto.go`,
`backend/internal/model/schema_migrate_test.go`, `backend/pkg/redact/*`,
`frontend/src/pages/SecuritySettingsPage.tsx`, `frontend/src/pages/admin/Admin{Support,Engine,Runner,BrokerHealth}Page.tsx`,
`frontend/src/features/trading/components/MarketSessionCard.tsx`, `.env.example`.

---

### 2026-11-13 — Fase I, J, K, P: flag sistem, CMS, operasional, alur akun

**Pelaku:** Claude
**Tipe:** `SELESAI`
**Status:** Backend, Runner, dan frontend terpasang. Seluruh tes lulus.

**Ringkasan pekerjaan**

| Fase | Yang dikerjakan |
|---|---|
| I | Tabel `system_setting` (murni override), cache 10 detik, 7 flag, ubah/kembalikan lewat panel dengan konfirmasi ketik ulang, audit before→after |
| I | `settings.Service` disambungkan ke Runner: flag dibaca ulang **setiap evaluasi sinyal**, jadi rem darurat berlaku ≤10 detik tanpa restart |
| J | Tabel `faq` + `announcement`, endpoint CMS, FAQ publik, pengumuman di dashboard, landing `/`, `/harga`, `/fitur`, `/tentang`, `/bantuan` |
| J | Entri Vite ganda: `index.html` (publik) dan `app.html` (dashboard di `/app`) |
| K | Tabel `system_state` satu baris (`normal`/`read-only`/`maintenance`) + `support_note`; penegakan mode di Runner; watchdog peringatan admin; `scripts/backup.sh`, `restore.sh`, `audit-log.sh`; `docs/RUNBOOK.md` |
| P | Tabel `email_outbox` + `email_token`, worker SMTP di backend-api, lupa sandi (selalu 200), reset (cabut semua sesi), verifikasi email, onboarding 5 langkah, banner langganan |

**Keputusan yang perlu diketahui Hermes**

1. **`system_setting` tidak punya nilai bawaan.** Baris kosong = env yang
   berlaku. "Kembalikan ke bawaan" = hapus baris.
2. **Flag yang dibaca Runner:** `AUTO_EXECUTE`, `DRY_RUN_ALL`,
   `MAX_DAILY_TRADES`, `MAX_DAILY_LOSSES`, `STOP_AFTER_WIN`,
   `MARKET_OPEN_ONLY`, `LEGAL_GATE`. Nilai int di-clamp oleh `DailyPolicy.Clamp()`
   sehingga aturan 2-trade / stop-setelah-menang tetap tidak bisa dilanggar.
3. **Mode `read-only` dan `maintenance` melewati sinyal**, bukan menunda.
   Alasan penolakan baru: `RejectReadOnly`, `RejectMaintenance`.
   **Posisi berjalan tidak pernah disentuh** oleh mode mana pun.
4. **Superadmin kini benar-benar dikunci.** Sebelumnya `SuperadminOnly()` hanya
   ada di katalog tanpa ditegakkan; sekarang `authz.Service.Has` menolak
   `system.settings` dan `system.maintenance` untuk selain role superadmin.
5. **Checkout, trial, dan hubungkan broker menolak email yang belum
   terverifikasi** (`ErrEmailNotVerified`, 403). Login tetap boleh.
6. **`POST /auth/forgot-password` selalu 200** apa pun hasilnya (anti enumerasi).
7. **`email_outbox.payload` hanya memuat `token_id`**, tidak pernah token mentah.
8. Endpoint publik baru: `GET /api/v1/status`, `/faq`, `/announcements`.

**Pekerjaan yang BELUM dikerjakan**

- Halaman frontend pengaturan keamanan (2FA, daftar sesi, toggle idle warning)
  masih menempel di `SettingsPage`; belum dipisah menjadi `/pengaturan/keamanan`.
- Halaman `/admin/dukungan` belum dibuat (endpoint jurnal dukungan sudah ada).
- Waktu mulai/tutup pasar belum dihitung otomatis di UI; masih membaca laporan
  engine.
- Observabilitas masih log terstruktur; belum ada rotasi otomatis di sisi app
  (diserahkan ke driver log Docker).

**File terdampak (utama)**

`backend/internal/settings/*`, `backend/internal/alert/*`,
`backend/internal/email/*`, `backend/internal/service/{admin_system,system_ops,cms,email_account,email_onboarding,watchdog}*`,
`backend/internal/handler/{admin_system,cms,email_account}_handler.go`,
`backend/internal/router/router.go`, `backend/internal/runner/{flags,service,trial,rules}.go`,
`backend/cmd/runner/flags.go`, `backend/cmd/backend-api/{main,disk_*}.go`,
`frontend/src/pages/public/*`, `frontend/src/pages/admin/Admin{Settings,Health,Maintenance,Cms}Page.tsx`,
`scripts/*.sh`, `docs/RUNBOOK.md`.

---

### 2026-11-12 — Fase D–H selesai: manajemen pelanggan, audit, harga, billing, laporan

**Pelaku:** Claude
**Tipe:** `SELESAI`
**Status:** Backend + panel admin frontend terpasang.

**Ringkasan pekerjaan**

| Fase | Yang dikerjakan |
|---|---|
| D | Daftar/detail pelanggan, akun broker, riwayat posisi, legal, catatan internal, suspend/ban/hapus/tier, bekukan akun |
| E | Jejak audit berkursor dengan rentang wajib, katalog aksi, penyaring rahasia, pemindai rute |
| F | Paket & fitur sebagai data, urutan harga subtotal→kupon→kredit→pagar, kupon, buku besar kredit |
| G | Checkout, webhook khusus bertanda tangan, idempoten, masa tenggang 3 hari |
| H | Laporan pendapatan, pelanggan, kupon, operasional |

**Keputusan yang perlu diketahui**

1. **Hanya webhook yang menaikkan tier.** Halaman `?status=success` tidak.
   Status gateway tak dikenal dan kurang bayar **tidak** pernah dianggap lunas.
2. **Batas tier dibaca dari `plan_feature`, bukan kode.** Free: 1 simbol, telat
   15 menit, tanpa auto-eksekusi. Basic: 8 simbol, tanpa auto-eksekusi.
   Plus: 8 simbol, auto-eksekusi, 2 akun.
3. **ILA 3.0 = `coming_soon`** di semua paket dan belum bisa dipilih.
4. **Harga mock:** Basic Rp 666.000, Plus Rp 999.000. Premium tidak ada.
5. **Pagar harga:** Basic minimal Rp 333.000, Plus Rp 499.000. Total di bawah
   pagar ditolak sebelum invoice terbit.
6. **Baca sensitif tercatat:** membuka riwayat posisi pelanggan menulis
   `user.view.sensitive` setiap kali, tanpa terkecuali.
7. **Kredensial tidak punya endpoint.** Tidak ada route yang mengembalikan
   token broker, bahkan untuk superadmin.

**Batas pandang yang dijaga**

| Tidak bisa dilakukan admin | Alasan |
|---|---|
| Melihat token broker | Nilai itu tidak pernah keluar dari server |
| Mengubah kata sandi pelanggan | Bukan wewenang staf |
| Menutup posisi berjalan | Kita percaya pada eksekusi kita |
| Mengubah tier akun sendiri | Memberi diri akses berbayar |
| Menghapus jejak audit | Append-only |

**Berkas baru utama**

- `backend/internal/pagination/` — daftar, kursor, rentang WIB→UTC
- `backend/internal/audit/` — katalog aksi + penyaring rahasia
- `backend/internal/timezone/` — rumah tunggal WIB/NY
- `backend/internal/export/` — CSV ber-BOM dua kolom waktu
- `backend/internal/payment/` — antarmuka gateway + adapter Pakasir
- `backend/internal/service/admin_user_service.go`, `admin_audit_service.go`,
  `admin_billing_service.go`, `pricing_service.go`, `billing_service.go`,
  `admin_report_service.go`
- `backend/internal/repository/admin_*.go`, `pricing_repository.go`,
  `billing_repository.go`, `billing_seed.go`, `report_repository.go`
- `frontend/src/pages/admin/*` — pengguna, detail, audit, paket, kupon, laporan
- `frontend/src/lib/admin-api.ts`, `frontend/src/types/admin.ts`

**Perubahan pada kode lama**

- `internal/runner/schedule.go` dan `rules.go`: `NY`/`WIB` kini mengambil dari
  `internal/timezone` agar paket lain bisa memakainya tanpa import cycle.
  **Perilaku trading tidak berubah.**
- `middleware.ContextRequestID`: nomor permintaan disalin ke baris audit.
- `authz.Catalog`: tambah `user.export`.

**Catatan untuk Hermes**

- Engine **tidak perlu berubah** untuk Fase D–H. Jalur `signal`, `position`,
  `user_signal`, `execution_record`, `credential` tidak disentuh.
- Satu hal yang **masih menunggu** di sisi engine: `ExecutedPrice: req.OpenPrice`
  di `go-trade`. Itu dicatat sebagai temuan, bukan dikerjakan di repo ini.
- Bila engine nanti membaca batas tier, sumbernya sekarang `plan_feature`
  (tabel), bukan konstanta di kode.

**Uji yang dijalankan**

```
gofmt -l .            → kosong
go vet ./...          → bersih
go test ./...         → 19 paket lulus, 0 gagal
npx tsc --noEmit      → bersih
npx eslint src        → 0 peringatan
npx vitest run        → 34 tes lulus
```

**Kemudian ditutup dalam rangkaian yang sama**

| Hal | Status |
|---|---|
| Halaman `/legal` dan `/legal/:slug` | Selesai, bisa dibaca tanpa masuk |
| Halaman `/langganan` dan `/langganan/bayar` | Selesai |
| Runner menolak entry saat dokumen wajib belum disetujui | Selesai (`RejectLegalPending`) |
| Tukar kode kredit | Selesai, penukaran ganda ditolak indeks unik |
| Pesan akun beku | Diperbaiki, kini menyebut langkah berikutnya |

**Yang BELUM dikerjakan (jujur)**

| Hal | Status |
|---|---|
| Halaman 2FA (QR, recovery codes) di frontend | Endpoint siap, halaman belum |
| Webhook Pakasir diuji ke server asli | Baru diuji ke server tiruan |
| Landing page, CMS panel, help center | Belum |
| Persetujuan legal di UI setelah login | Endpoint siap, halaman belum |

---

### 2026-11-08 — 2FA opsional (user & admin) + Fase C selesai: legal berversi

**Pelaku:** Claude
**Tipe:** Fitur 2FA + Penyelesaian Fase C
**Status:** SELESAI — `gofmt -l` kosong, `go vet` bersih, 16 paket tes lulus

---

## Bagian 1 — 2FA (TOTP), opsional untuk user maupun admin

### Prinsip

Tanpa layanan pihak ketiga. Aplikasi authenticator menghitung kode di
perangkat dari secret yang kita berikan lewat QR; server kita menghitung dengan
rumus yang sama (RFC 6238). Tidak ada pendaftaran, tidak ada API key, tidak ada
panggilan ke server Google.

### Berkas baru

| Berkas | Isi |
|---|---|
| `internal/totp/totp.go` | TOTP/HOTP murni stdlib: HMAC-SHA1, 6 digit, 30 detik, toleransi ±1 periode |
| `internal/totp/totp_test.go` | **Sepuluh vektor uji resmi RFC 4226** + tes toleransi & bentuk salah |
| `internal/model/twofactor.go` | Tabel `user_2fa`, `user_recovery_code`, `two_factor_challenge` |
| `pkg/cryptox/sealer.go` | Enkripsi AES-GCM terikat pemilik untuk secret 2FA |
| `internal/repository/twofactor_repository.go` | Secret, kode pemulihan, tantangan |
| `internal/service/twofactor_service.go` | Aktivasi, verifikasi, recovery, nonaktifkan |
| `internal/service/challenge_service.go` | Keadaan antara setelah kata sandi benar |
| `internal/handler/twofactor_handler.go` | Endpoint `/2fa/*` untuk kedua pintu |

### Alur

```
Kata sandi benar → 2FA aktif?
                    ├─ tidak → sesi terbit langsung (step: complete)
                    └─ ya    → terbit challenge_id (step: two_factor_required)
                                → POST /verify-2fa {challenge_id, code}
                                → sesi terbit
```

| Aturan | Nilai |
|---|---|
| Umur tantangan | 5 menit, sekali pakai |
| Kode salah per tantangan | 5, lalu tantangan hangus |
| Toleransi TOTP | ±1 periode (±30 detik) |
| Kode pemulihan | 8, hanya ditampilkan sekali, sekali pakai |
| Secret disimpan | Terenkripsi master key, terikat ke pemiliknya |
| Kode dipakai ulang | Ditolak (`last_used_step`) |

### Endpoint

| Endpoint | Keterangan |
|---|---|
| `POST /api/v1/auth/verify-2fa` | Langkah kedua masuk user |
| `POST /api/v1/admin/auth/verify-2fa` | Langkah kedua masuk admin |
| `GET /2fa` | Status 2FA akun ini |
| `POST /2fa/setup` | Secret baru + `otpauth_url` |
| `POST /2fa/activate` | Verifikasi kode lalu aktifkan; balasan berisi 8 kode pemulihan |
| `POST /2fa/disable` | Matikan setelah kode atau kode pemulihan terbukti |
| `POST /2fa/recovery-codes` | Set kode pemulihan baru (kode lama hangus) |

Semua `/2fa` tersedia di kedua pintu (`/api/v1/2fa*` dan `/api/v1/admin/2fa*`).

### Celah yang ditemukan & ditutup saat mengerjakan 2FA

> **Cookie CSRF admin dan user bertabrakan.** Sebelumnya keduanya memakai
> `ila_csrf`. Bila user dan admin masuk di peramban yang sama, login kedua
> menimpa nilai pertama sehingga permintaan berikutnya ditolak 403 secara acak.
> Sekarang admin memakai `ila_admin_csrf` (env `ADMIN_CSRF_COOKIE_NAME`).

> **Dua jam berbeda untuk satu masa berlaku.** Layanan 2FA dan pengelola
> tantangan semula punya sumber waktu masing-masing, sehingga masa berlaku
> tantangan tidak bisa diuji dan berpotensi meleset. Kini jamnya disamakan.

> **Tantangan 2FA tidak memeriksa pemiliknya.** Versi pertama memakai `userID`
> yang dibaca dari baris tantangan itu sendiri, sehingga pemeriksaan pemilik
> tidak bermakna. Sekarang pemilik dibaca lewat `Owner()` lalu dicocokkan.

### Tanpa MASTER_KEY

Endpoint 2FA sengaja tidak dipasang dan alasannya dicatat di log. Menyimpan
secret tanpa enkripsi lebih buruk daripada tidak ada 2FA sama sekali.

---

## Bagian 2 — Fase C: Legal & Persetujuan

### Berkas baru

| Berkas | Isi |
|---|---|
| `internal/legal/catalog.go` | Lima jenis dokumen + aturan kapan persetujuan diminta |
| `internal/legal/drafts.go` | Draf isi versi 1 (ToS, Privasi, Risiko, Refund, Fair Use) |
| `internal/model/legal.go` | Tabel `legal_document`, `legal_acceptance` |
| `internal/repository/legal_repository.go` | Dokumen berversi + bukti persetujuan |
| `internal/repository/legal_seed.go` | Terbitkan versi 1 hanya bila slug belum punya versi |
| `internal/service/legal_service.go` | Status persetujuan, penegakan, pengelolaan admin |
| `internal/handler/legal_handler.go` | Endpoint publik, user, dan admin |

### Aturan dokumen

| Aturan | Alasan |
|---|---|
| Dokumen tidak pernah ditimpa | Versi lama adalah bukti masa lalu |
| Perubahan = versi baru | Nomor versi naik otomatis |
| Draf dan terbit terpisah | `published_at` kosong berarti draf |
| `effective_at` boleh di masa depan | Menjadwalkan berlaku |
| Dokumen terbit tidak bisa `PATCH` | Mencegah bukti berubah di belakang |

### Kapan persetujuan diminta

| Momen | Dokumen |
|---|---|
| Mendaftar | `tos`, `privacy`, `fair_use` |
| Menghubungkan akun broker | `risk` |
| Pembayaran pertama | `refund` (belum aktif, menunggu Fase G) |

`risk` sengaja tidak diminta saat daftar: sebelum ada akun, user belum
menanggung risiko finansial apa pun.

### Endpoint

| Endpoint | Keterangan |
|---|---|
| `GET /api/v1/legal` | Daftar dokumen yang berlaku (publik) |
| `GET /api/v1/legal/:slug` | Isi versi berlaku (publik) |
| `GET /api/v1/me/legal` | Status persetujuan + daftar yang tertunda |
| `POST /api/v1/me/legal/:slug/accept` | Catat persetujuan `{version}` |
| `GET /api/v1/admin/legal` | Seluruh versi termasuk draf (`cms.view`) |
| `POST /api/v1/admin/legal` | Buat draf versi baru (`cms.edit`) |
| `PATCH /api/v1/admin/legal/:id` | Ubah draf (`cms.edit`) |
| `POST /api/v1/admin/legal/:id/publish` | Terbitkan (`cms.publish`) |

### Penegakan

| Tempat | Perilaku |
|---|---|
| Registrasi | Tanpa centang → **400 sebelum menyentuh database** |
| Hubungkan akun broker | `risk` belum disetujui → **400** |
| `GET /system/status` | Field `legal_blocked` menandai fitur tertahan |
| Setuju versi lama | Ditolak: versi harus yang sedang berlaku |

### Migrasi

`cmd/migrate` kini juga menerbitkan versi 1 kelima dokumen. Slug yang sudah
punya versi apa pun dilewati, sehingga perubahan isi oleh admin tidak tertimpa
saat migrasi dijalankan ulang.

### Tes yang mengunci perilaku

| Berkas | Yang dikunci |
|---|---|
| `internal/totp/totp_test.go` | Sepuluh vektor RFC 4226, toleransi ±1, bentuk salah ditolak |
| `internal/service/twofactor_service_test.go` | Aktivasi, replay ditolak, recovery sekali pakai, secret terenkripsi & terikat pemilik |
| `internal/service/login_twofactor_test.go` | Tantangan 5 menit, sekali pakai, hangus setelah 5 salah, purpose salah ditolak |
| `internal/service/legal_service_test.go` | Registrasi wajib centang, versi tercatat, versi baru memblokir, dokumen terbit tak bisa diubah |
| `internal/router/router_test.go` | Legal publik tanpa sesi, 2FA butuh sesi, 2FA mati tanpa master key |
| `pkg/cryptox/sealer_test.go` | Round-trip, pemilik berbeda gagal, nonce acak |

### Belum dikerjakan

Ini daftar jujur, bukan klaim selesai.

| Hal | Keterangan |
|---|---|
| **Antarmuka 2FA** | Belum ada halaman frontend; endpoint siap dipakai |
| **QR code** | Backend hanya mengirim `otpauth_url`; gambar QR dirender frontend |
| **Sesi terbatas setelah recovery** | Recovery memakai satu kode lalu login normal; pembatasan 15 menit ke `/2fa/*` belum diterapkan |
| **Reset 2FA lewat CLI** | Jalur darurat `ila-admin reset-2fa` belum dibuat |
| **Persetujuan ulang paksa** | `legal_blocked` dilaporkan, tetapi Runner belum menolak eksekusi karenanya |
| **Halaman `/legal/*` frontend** | Belum ada |
| **CMS FAQ & pengumuman** | Bagian Fase J, belum dikerjakan |

#### Untuk Hermes (`go-trade`)

| Hal | Dampak pada engine |
|---|---|
| 2FA | **Tidak ada.** Murni pintu console |
| Legal | **Belum ada.** Bila Runner nanti perlu menolak eksekusi karena `legal_blocked`, kontrak sinyal tidak berubah |
| Kontrak sinyal | **Tidak berubah** |

---

### 2026-11-08 — Fase B selesai: RBAC (role, permission, pintu admin)

**Pelaku:** Claude
**Tipe:** Penyelesaian Fase B
**Status:** SELESAI — `go build`, `go vet`, `gofmt -l` kosong, seluruh tes lulus

#### Ringkasan

Kewenangan admin kini dibatasi **per izin**, bukan sekadar "siapa yang admin".
Ada 32 permission di katalog tetap, 4 role bawaan, dan setiap route admin
**wajib** menyebut izin yang menjaganya.

#### Berkas baru

| Berkas | Isi |
|---|---|
| `model/rbac.go` | Tabel `role`, `permission`, `role_permission` |
| `authz/catalog.go` | Katalog 32 permission + 4 role bawaan; `Known()`, `SuperadminOnly()` |
| `authz/service.go` | Pemuat permission dengan cache 30 detik; `Invalidate`/`InvalidateAll` |
| `authz/authz_test.go` | Tes cache, katalog, isi role bawaan |
| `repository/role_repository.go` | CRUD role, permission per role/user, hitung pemakai |
| `repository/role_migration.go` | Pindahkan akun `support` lama; isi `role_id` akun admin |
| `repository/authz_loader.go` | Menyambung repository ke pemuat permission middleware |
| `middleware/permission.go` | `AdminScope`, `RequirePermission`, penjaga startup route admin |
| `service/role_service.go` | Lima larangan pengamanan + pencatatan audit per permission |
| `service/admin_auth_service.go` | Login pintu admin (scope terpisah, sesi terpisah) |
| `handler/admin_role_handler.go` | `GET/POST/PATCH/DELETE /admin/roles`, `/admin/permissions`, `PATCH /admin/users/:id/role` |
| `handler/admin_auth_handler.go` | `POST /admin/auth/login`, `/logout`, `GET /admin/auth/csrf`, `/auth/me` |
| `dto/role_dto.go`, `dto/admin_auth_dto.go` | Kontrak request/response RBAC |

#### Berkas yang diubah

| Berkas | Perubahan |
|---|---|
| `model/user.go` | Kolom `role_id` (nullable) — penunjuk role RBAC |
| `middleware/permission.go` | `AdminScope` kini **memeriksa DB**: status akun, `session_version`, dan role kasar `admin` |
| `config/config.go` | `ADMIN_COOKIE_NAME` (bawaan `ila_admin_session`), `ADMIN_JWT_EXPIRY` (bawaan `2h`); ditolak bila sama dengan cookie sesi user |
| `router/router.go` | Grup admin + pemeriksaan startup: route admin tanpa izin **menggagalkan start** |
| `router/deps.go` | `newAdminTokenIssuer` (umur sesi admin lebih pendek) |
| `session/session.go` | `WithSessionName` agar cookie admin terpisah dari cookie user |
| `cmd/migrate/main.go` | Migrasi tabel RBAC + selaraskan katalog + seed role + migrasi data lama |
| `.env.example` | `ADMIN_COOKIE_NAME`, `ADMIN_JWT_EXPIRY` |

#### Lima larangan pengamanan (plans/04 §4.1)

| # | Larangan | Diterapkan di |
|---|---|---|
| 1 | Mengubah role sendiri | `AssignRole` — `actorID == userID` ditolak |
| 2 | Menyentuh role `superadmin` / role bawaan | `guardSystemRole` pada ubah & hapus |
| 3 | Mencabut role admin aktif terakhir | `CountActiveAdmins` sebelum mencabut |
| 4 | Memberi izin yang tidak dimiliki sendiri | `ensureCanGrant` pada buat/ubah/tugaskan |
| 5 | Menghapus role yang masih dipakai | `CountUsersWithRole` sebelum hapus |

Ditambah: role superadmin hanya bisa diberikan oleh superadmin.

#### Celah yang ditemukan saat mengerjakan Fase B

> **Pintu admin sebelumnya hanya memverifikasi tanda tangan token.** Di versi
> awal, `AdminScope` tidak membaca database, sehingga admin yang di-ban atau
> yang sesinya dicabut tetap bisa masuk sampai token kedaluwarsa (2 jam). Kini
> setiap permintaan admin mencocokkan status akun dan `session_version` ke DB.

> **Menurunkan role akun tidak memutus sesinya.** `AssignRole` kini menaikkan
> `session_version` dan membuang cache akun target, sehingga token lamanya
> langsung tidak berlaku.

#### Tes yang mengunci perilaku

| Berkas | Yang dikunci |
|---|---|
| `authz/authz_test.go` | Cache 30 detik, `Invalidate`, katalog tanpa duplikat, admin tanpa kewenangan terkunci |
| `service/role_service_test.go` | Kelima larangan, role kosong ditolak, permission asing ditolak, audit per permission |
| `service/admin_auth_service_test.go` | Pelanggan tidak bisa masuk pintu admin; admin tanpa role RBAC ditolak; password salah 401 |
| `middleware/permission_test.go` | 401 vs 403, pesan 403 tidak menyebut nama permission, admin ter-ban ditolak |
| `router/router_test.go` | Semua route admin terdeklarasi; pendeteksi deklarasi hilang; RBAC terpasang |

#### Perilaku penolakan (plans/04 §5.3)

| Keadaan | HTTP |
|---|---|
| Tanpa sesi admin | 401 |
| Token ber-scope user di pintu admin | 403 |
| Admin tanpa permission yang diminta | 403 — pesan tidak menyebut izin mana |
| Admin di-ban / sesi dicabut | 401 atau 403 |

#### Belum dikerjakan (jujur, bukan klaim)

| Hal | Keterangan |
|---|---|
| **2FA admin** | Belum dirancang. Login admin saat ini password saja. Flag `RequireTwoFA` sudah ada dan akan **menolak** login bila diaktifkan sebelum alurnya siap |
| **Ubah `user.role` kasar** | Belum ada endpoint. `role_id` hanya bisa ditugaskan ke akun yang `role`-nya sudah `admin` |
| **Seksi auth di frontend** | Belum ada halaman `/admin/*`; menunggu Fase D |
| **Pemulihan cache lintas proses** | Cache permission disimpan di memori proses; aman karena hanya ada satu backend-api |
| **Uji endpoint dengan DB nyata** | Tes router memakai koneksi tiruan; verifikasi menyeluruh butuh database sungguhan |

#### Untuk Hermes (`go-trade`)

Tidak ada perubahan kontrak engine. Yang perlu diketahui:

| Hal | Dampak |
|---|---|
| `user.status` tetap satu-satunya penentu boleh-tidaknya eksekusi | Tidak berubah |
| `role_id` **tidak** memengaruhi jalur trading | Runner tidak membaca RBAC |
| Ban user → `session_version` naik → token user mati | Tetap seperti Fase A |

---

### 2026-11-08 — Fase A selesai: daftar sesi, idle warning, PnL jujur

**Pelaku:** Claude
**Tipe:** Penyelesaian Fase A
**Status:** SELESAI — backend `go vet` + tes lulus, frontend `tsc`/`eslint`/34 tes lulus

#### A6 — Daftar sesi aktif & cabut per perangkat

| Berkas | Perubahan |
|---|---|
| `model/session.go` | **Tabel baru**: `session` (user, scope, token_id hash, user_agent, last_seen, expires, revoked) |
| `repository/session_repository.go` | Create/FindByTokenID/ListByUser/Touch/Revoke/RevokeAll + `NewTokenID` |
| `service/session_service.go` | Daftar & cabut; label perangkat dikenali dari user agent |
| `handler/session_handler.go` | `GET /me/sessions`, `DELETE /me/sessions/:id` |
| `pkg/utils/jwt.go` | Klaim `sid`; `TTL()` |
| `cmd/migrate/main.go` | `model.Session` masuk daftar migrasi |

**Celah yang ditemukan & ditutup saat mengerjakan A6:**

> Mencabut satu perangkat **tidak** mengeluarkannya, karena verifikasi per
> permintaan memakai `session_version` di tabel `user`. Sekarang pencabutan
> satu sesi **ikut menaikkan `session_version`**, sehingga token perangkat itu
> benar-benar ditolak. Perangkat ini otomatis masuk ulang.

Dua tes mengunci perilaku ini: `TestCabutSesiMenaikkanVersiSesi` dan
`TestCabutSesiAsingTidakMenaikkanVersi`.

#### A14/A15 — Sesi di frontend

| Berkas | Perubahan |
|---|---|
| `lib/http.ts` | **401 memicu logout, 403 TIDAK.** Pesan khusus untuk 403/423/429 |
| `features/auth/hooks/use-idle-timer.ts` | Timer diam; polling tidak dihitung aktivitas; `BroadcastChannel` antar-tab |
| `features/auth/components/IdleWarningDialog.tsx` | Modal hitung mundur 60 detik, dua tombol, fokus ke "masih di sini" |
| `features/auth/AuthProvider.tsx` | Integrasi timer (25 menit → peringatan, 30 menit → keluar) |
| `features/auth/components/SessionSection.tsx` | Daftar perangkat + tombol cabut |
| `features/auth/components/LoginForm.tsx` | Menjelaskan alasan keluar (diam / sesi berakhir) |

**Batas yang tidak bisa dilonggarkan pengguna:** `idle_warning` hanya
mematikan **modal**. Batas diam di server tetap berlaku.

#### A10 — PnL jujur (C9)

| Berkas | Perubahan |
|---|---|
| `runner/monitor.go` | `pnlKnown` dihitung dari exit price/profit; hasil tak diketahui **tidak** dicatat sebagai kalah |
| `dto/trading_dto.go` | `pnl_known`, `opened_at_pair`, `closed_at_pair` |
| `pages/DashboardPage.tsx`, `pages/PositionsPage.tsx` | Tampilkan `—`, bukan `US$0,00` |

**Penting untuk aturan trading:** hasil yang tidak dilaporkan broker **tidak
menghabiskan kuota**. Mencatatnya sebagai kalah akan menghentikan trading
user tanpa trade yang benar-benar rugi.

#### Tes baru

`session_service_test.go` (5), `auth_test.go` (6), `idle-timer.test.ts` (5),
`http-status.test.ts` (4), `close_notify_test.go` (+2).

#### Catatan penutup Fase A

Seluruh item A1–A19 sudah dikerjakan. Yang **belum** dan memang di luar Fase A:
`signal_delay_minutes` (C2 → Fase F) dan `total_trades` per tier (C6 → Fase F).

---

### 2026-11-08 — Modul dipendekkan jadi `ila/backend` + Fase A fondasi

**Pelaku:** Claude
**Tipe:** Refactor penamaan + perbaikan fondasi
**Status:** SELESAI — `go build`, `go vet`, `gofmt`, seluruh tes lulus

#### 1. Nama modul dipendekkan

| Sebelum | Sesudah |
|---|---|
| `module github.com/KirisakiRei/go-ila-platform/backend` | `module ila/backend` |
| `"github.com/KirisakiRei/go-ila-platform/backend/internal/dto"` | `"ila/backend/internal/dto"` |

66 berkas Go + `go.mod`, murni penggantian nama import. **Tidak ada logika
berubah.** Hermes: bila ada kode yang mengimpor paket ini dari luar, sesuaikan
path-nya.

Sekaligus penanda `// indirect` pada `gorm.io/gorm` dan
`gorm.io/driver/postgres` dihapus karena keduanya dipakai langsung.

#### 2. Celah fondasi yang ditutup

| # | Celah | Perbaikan |
|---|---|---|
| **C1** 🔴 | Notifikasi tutup posisi tidak pernah dikirim | `Publisher.PublishPositionClosed` ditambahkan; `applyClose` memanggilnya. Pesan memuat hasil + sisa kuota |
| **C3** 🔴 | Runner tidak cek `user.status` di jalur eksekusi | `evaluate` menolak `RejectAccountBanned` sebelum entitlement |
| **C9** | PnL tak terbaca tampil `US$0,00` | Respons posisi punya `pnl_known`; `false` berarti tampilkan "—" |
| **C11** | Akun beku tetap simulasi | Sudah benar (`executableAccounts` menyaring `IsFrozen`); dikunci dengan tes |

#### 3. Fondasi sesi

| Berkas | Perubahan |
|---|---|
| `model/user.go` | Kolom `SessionVersion`, `IdleWarning` |
| `pkg/utils/jwt.go` | Klaim `scope` + `sv`; `IssueWithClaims` |
| `internal/authstate/` | **Paket baru**: bentuk ringkas akun untuk validasi sesi, menghindari import cycle |
| `middleware/auth.go` | Cocokkan `sv` dengan DB, tolak akun ter-ban, tolak token berscope admin di pintu user |
| `service/auth_service.go` | `ChangePassword` (verifikasi password lama + cabut semua sesi), `LogoutAll` |
| `repository/user_repository.go` | `RevokeSessions`, `UpdatePassword` (naik `session_version` dalam satu pernyataan) |
| Router | `POST /api/v1/me/password`, `POST /api/v1/me/logout-all` |

#### 4. Keamanan & waktu

| # | Perbaikan |
|---|---|
| **D3** | `APP_ENV=production` menolak start bila `ALLOWED_ORIGINS` kosong, `*`, atau memuat localhost |
| **D4** | `middleware.RateLimiter` — batas 120/menit per akun, terpasang di grup `/api/v1` |
| **E3** | `runner/schedule.go` — deteksi pasar buka di akhir pekan / tutup di luar jadwal + status usang |
| **E5** | `dto.TimePair` (`utc`, `wib`, `ny`) pada `server_time_pair`, `day_pair`, `opened_at_pair`, `closed_at_pair` |
| **D5** | `PaginationQuery` lengkap: `per_page` maks 200, `q`, `cursor`, `sort`, `order`, `from`, `to`, `has_more`, `approx_total` |

#### 5. Tes baru

`ban_test.go`, `close_notify_test.go`, `schedule_test.go`, `auth_test.go`,
`ratelimit_general_test.go`, `cors_guard_test.go`, `time_dto_test.go`,
`pagination_dto_test.go`, `TestSetDailyTimezone`.

#### Yang perlu Hermes perhatikan

- Jam trading **tetap** New York; hari akuntansi WIB.
- Kontrak gRPC **tidak berubah**.
- `Publisher` punya method baru — bila ada implementasi lain, tambahkan
  `PublishPositionClosed`.

---

### 2026-11-08 — Zona waktu harian kini bisa dikonfigurasi (`DAILY_TZ`)

**Pelaku:** Claude
**Tipe:** Perbaikan bug
**Status:** SELESAI — `go build`, `go vet`, seluruh tes lulus

**Bug yang diperbaiki:**
`DAILY_TZ` sudah dibaca `config.Load()` tetapi `runner/rules.go` memakai
konstanta keras `time.FixedZone("WIB", 7*60*60)`. Akibatnya mengubah env
`DAILY_TZ` **tidak berefek apa pun** — ada dua sumber kebenaran untuk zona
hari akuntansi.

**Perbaikan:**

| Berkas | Perubahan |
|---|---|
| `internal/runner/rules.go` | `WIB` jadi variabel + `SetDailyTimezone(name)`; nama tak dikenal diabaikan |
| `cmd/runner/main.go` | Panggil `runner.SetDailyTimezone(cfg.DailyTimezone)` sebelum service dibuat |
| `cmd/backend-api/main.go` | Panggil hal yang sama agar backend-api dan Runner satu zona |
| `internal/service/system_service.go` | `wib` tidak lagi didefinisikan ulang; memakai `runner.WIB` |
| `internal/runner/rules_test.go` | Tes `TestSetDailyTimezone` mengunci perilaku |

**Yang TIDAK berubah:**
- `DayKey/DayStart/DayEnd` tetap berbasis WIB
- Sesi pasar tetap `America/New_York` di engine
- Jalur trading, kuota, dan rekonsiliasi tidak disentuh
- Bawaan tetap `Asia/Jakarta` bila env kosong

**Dampak ke engine:** **tidak ada.** Kontrak gRPC tidak berubah.

**Yang perlu Hermes tahu:** jam trading tetap New York. Yang berubah hanya
bahwa zona *hari akuntansi* kini mengikuti `DAILY_TZ` alih-alih konstanta.

---

### 2026-11-08 — Frontend Console: tema monokrom + perbaikan tata letak

**Pelaku:** Claude
**Tipe:** UI/UX
**Status:** SELESAI, empat gerbang kualitas hijau

**Ringkasan:**

Console dirombak ke **monokrom murni**. Status tidak lagi dibedakan warna,
melainkan lewat bobot huruf, garis, dan bentuk ikon. Ini kerangka awal untuk
menilai struktur dan alur; palet final menyusul.

**Perubahan struktur:**

| Sebelum | Sesudah |
|---|---|
| `Header.tsx` terpisah | `AppShell.tsx` memegang sidebar + topbar |
| `SignalCard.tsx` (satu blok besar) | `SignalRow` + `SignalDetail` (split view) |
| Kartu akun menempel di halaman | `AccountCard.tsx` di dalam `features/accounts` |
| Form login/daftar tanpa bingkai | `AuthLayout.tsx` memusatkan formulir |
| — | `PageHeader`, `Section`, `Stat`/`StatGrid`, `EmptyState` |

**Bug yang ditemukan di layar dan diperbaiki:**

| # | Masalah | Penyebab | Perbaikan |
|---|---|---|---|
| 1 | Tautan "Daftar" muncul dua kali di halaman masuk | Footer ditulis di `LoginForm` **dan** `AuthLayout` | Footer hanya di `AuthLayout`; `RegisterForm` sama |
| 2 | Angka di tabel Posisi tidak sejajar per kolom | SL/TP digabung satu sel dengan pemisah `/`; header tanpa `text-right` | SL dan TP dipisah jadi dua kolom, semua sel angka `text-right tabular` |
| 3 | "Tidak diketahui" tampil dua kali di Ringkasan | `hint` mengulang `value` | `Stat` tidak menampilkan `hint` bila sama dengan `value` |
| 4 | Menu terpotong di layar ponsel | Label panjang + gulir tanpa isyarat | Label ringkas di bawah `lg`; utilitas `.scrollbar-thin` |
| 5 | Kolom Saldo/Ekuitas/Margin berjarak lebar kosong | Grid tanpa batas lebar | `sm:max-w-xl` + jarak antar kolom lebih lebar |

**Tema:** token CSS monokrom (`--primary` hitam pekat), `.tabular` untuk semua
angka, `:focus-visible` selalu terlihat, dan `prefers-reduced-motion` dihormati.

**Gerbang kualitas:**

```text
npm run typecheck   bersih
npm run lint        0 peringatan
npm test            25 tes lulus (4 berkas)
npm run build       sukses
```

Tangkapan layar diverifikasi dengan Camoufox pada stack Docker hidup: delapan
halaman, **nol error konsol**.

**Catatan untuk Hermes:** tidak ada perubahan kontrak API. Frontend tetap hanya
memakai `docs/API_CONTRACT.md`; sesi tetap cookie `ila_session` HttpOnly + CSRF,
tanpa `localStorage`.

---

### 2026-11-08 — Uji mode produksi nyata di Docker + panduan `.env`

**Pelaku:** Claude
**Tipe:** VERIFICATION
**Status:** SELESAI, seluruh stack berjalan dengan `APP_ENV=production`

**Ringkasan:**

Stack dijalankan dengan konfigurasi produksi sungguhan: `COOKIE_SECURE=true`,
`APP_ENV=production`, dan seluruh rahasia hasil generate acak.

**Hasil uji pada stack berjalan:**

| Uji | Hasil |
|---|---|
| Frontend, API, `/health`, SPA fallback | 200 / 401 / ok / 200 |
| Cookie sesi | `HttpOnly` + `Secure` + `SameSite=Strict` |
| `POST /trial` tanpa CSRF | 403 |
| `POST /trial` dengan CSRF | 201 |
| Token di header `Authorization` | 401 (jalur lama ditutup) |
| Token plaintext di `credential.ciphertext` | **0 baris** |
| Ciphertext tersimpan | 158 B, kunci terbungkus 48 B, versi 1 |
| Token di log seluruh container | tidak ada |
| `MASTER_KEY`, `DB_PASSWORD`, `JWT_SECRET` di log | tidak ada |
| Rahasia diterima Caddy dan frontend | **tidak ada** |
| Restart backend + runner | sesi dan kredensial tetap utuh |

**Uji master key salah (bukti kunci benar-benar dipakai):**

Dengan `MASTER_KEY` diganti nilai acak lain, Runner gagal membuka kredensial
dan `last_error` berisi `gagal mendekripsi data` — tanpa membocorkan detail.
Setelah kunci asli dikembalikan, dekripsi berhasil dan error berubah menjadi
`kredensial broker tidak sah: HTTP 401`, yaitu penolakan dari Match-Trader
karena token ujinya palsu. Rantai **dekripsi → login broker** terbukti utuh.

**Dokumen baru:** `deploy/ENV-SETUP.md` — cara membuat rahasia, verifikasi
tidak bocor, penyimpanan master key, dan prosedur rotasi.

`.env.example` dilengkapi `TELEGRAM_BOT_USERNAME` dan
`TELEGRAM_WEBHOOK_SECRET` yang sebelumnya belum terdokumentasi.

---

### 2026-11-08 — Audit kredensial: 3 kebocoran ditemukan dan ditutup

**Pelaku:** Claude
**Tipe:** SECURITY
**Status:** SELESAI, diverifikasi pada server berjalan

**Ringkasan:**

Enkripsi kredensial sudah benar sejak awal (AES-256-GCM amplop, kunci per
akun). Audit menemukan **tiga jalur kebocoran** di sekitarnya, semuanya
diperbaiki. Analisis lengkap ada di `docs/SECURITY.md`.

**1. Token bot Telegram bocor lewat pesan error (tinggi)**

Token ada di path URL `api.telegram.org/bot<TOKEN>/sendMessage`. Saat
`http.Do` gagal, Go menyertakan URL lengkap, dan error itu tercatat di log
**serta** `notification_log.last_error`. Diperbaiki dengan
`sanitizeTransportError`.

**2. Token deep-link approve tercatat di log akses (sedang)**

`/api/v1/approve/<token>` dicatat apa adanya. Diperbaiki dengan `safePath`
yang menyamarkan segmen token tetapi mempertahankan `/confirm` dan `/decline`.

**3. Tidak ada lapisan kedua untuk rahasia di last_error (sedang)**

Paket baru `pkg/redact` menyaring token Telegram, JWT, header Bearer, dan
parameter `token=`/`password=`/`api_key=` sebelum teks disimpan.

**Tambahan: rotasi kunci.**

`MASTER_KEYS_LEGACY` memungkinkan penggantian `MASTER_KEY` tanpa kehilangan
data. Sebelumnya kunci lama selalu `nil`, sehingga rotasi berarti blob lama
mustahil dibuka.

**Verifikasi nyata terhadap database:**

| Uji | Hasil |
|---|---|
| Token plaintext di kolom ciphertext | 0 baris |
| `GET /accounts` memuat token | tidak ada |
| Token di seluruh file log | tidak ada |
| Ciphertext tersimpan | 161 byte, `key_version=1` |

**Verifikasi:** `gofmt` bersih, `build`/`vet` OK, **12 paket uji lulus**
(termasuk 15 tes baru untuk redaksi, path, dan kunci lama).

**Catatan untuk Hermes:** engine tidak diubah. Token di engine dibaca dari
environment dan sudah dipastikan tidak menyimpan ke database maupun mencetak
ke log.

---

### 2026-11-08 — Deploy Docker siap: satu domain lewat Caddy

**Pelaku:** Claude
**Tipe:** DEPLOYMENT
**Status:** SELESAI, seluruh stack diuji nyata termasuk deploy dingin

**Ringkasan:**

Enam container di VPS 2. **Hanya Caddy** yang membuka port ke internet;
backend, runner, frontend, dan postgres tidak terjangkau dari luar.

```text
https://ila.id/api/*  → backend-api:8080
https://ila.id/*      → frontend:3000 (nginx)
```

Satu origin disengaja: cookie sesi memakai `SameSite=Strict`, jadi deep-link
konfirmasi dari Telegram tetap membawa sesi.

**Masalah yang ditemukan saat pengujian dan diperbaiki:**

| Masalah | Akibat | Perbaikan |
|---|---|---|
| Tidak ada langkah migrasi | backend mati di DB kosong | service `migrate` + `depends_on: service_completed_successfully` |
| `DB_PASSWORD` ada di dua tempat | password bisa berbeda, login DB gagal | `DB_DSN` disusun dari `DB_USER/DB_PASSWORD/DB_NAME` |
| Caddy mewarisi seluruh `.env` | kredensial DB masuk container yang tidak butuh | hanya 2 variabel yang dikirim |
| Blok `email` dengan nilai kosong | Caddy gagal start berulang | blok global dihapus |
| Port backend terbit ke `127.0.0.1:8080` | backend bisa dijangkau tanpa Caddy | `expose` saja, tanpa `ports` |
| `add_header` setelah `return` di nginx | header tidak terpasang | pakai `default_type` |

**Hasil uji nyata lewat Caddy (HTTPS):**

| Uji | Hasil |
|---|---|
| Frontend `/` | 200 HTML |
| SPA fallback `/sinyal` | 200 HTML (bukan 404) |
| `/api/v1/me` tanpa sesi | 401 |
| Login → cookie | `HttpOnly` + `SameSite=Strict` utuh |
| `POST /trial` tanpa CSRF | 403 |
| `POST /trial` dengan CSRF | 201 |
| `GET /auth/csrf` | 200 |
| Logout | 200 |
| Deploy dingin `down -v` lalu `up` | semua sehat, migrasi otomatis |
| Restart backend/runner/caddy | pulih sendiri |

**Berkas terdampak:** `deploy/{Caddyfile,docker-compose.yml,Dockerfile.backend,nginx.conf,README.md}`,
`Makefile`, `.env.example`.

**Catatan untuk Hermes:** engine tidak tersentuh. Runner tetap memakai
`subscriber_id` `runner-ila-2.0` / `runner-ila-3.0` dan meminta pembaruan status.

---

### 2026-11-08 — Sesi pindah ke cookie HttpOnly + proteksi CSRF

**Pelaku:** Claude
**Tipe:** SECURITY
**Status:** SELESAI, terverifikasi terhadap server berjalan

**Ringkasan:**

Token sesi tidak lagi pernah disentuh JavaScript. Backend menaruhnya di
cookie `ila_session` (`HttpOnly`, `Secure` di produksi, `SameSite=Strict`),
dan permintaan tulis dijaga skema **double-submit CSRF**.

| Berkas | Peran |
|---|---|
| `internal/session/session.go` | set/hapus cookie, rotasi token CSRF |
| `internal/middleware/auth.go` | sesi hanya dari cookie; `RequireCSRF` |
| `internal/handler/auth_handler.go` | `logout`, `csrf`, set cookie saat login |
| `internal/config/config.go` | 4 kunci baru + tolak `COOKIE_SECURE=false` di produksi |

**Endpoint baru:**

```text
POST /api/v1/auth/logout   hapus kedua cookie
get  /api/v1/auth/csrf     terbitkan ulang cookie CSRF
```

**Hasil uji terhadap server berjalan:**

| Skenario | Hasil |
|---|---|
| Login menetapkan cookie | `ila_session` HttpOnly, `ila_csrf` terbaca |
| `GET /me` dengan cookie | 200 |
| `POST /trial` tanpa header CSRF | **403** |
| `POST /trial` dengan header benar | 201 |
| Header CSRF salah | **403** |
| Tanpa cookie sesi | 401 |
| Token di `Authorization: Bearer` | **401** (jalur lama ditutup) |
| Logout lalu `/me` | 401 |

**Frontend:** `localStorage` dihapus seluruhnya (`grep` membuktikan nol
kemunculan). `lib/http.ts` mengirim `X-CSRF-Token` dari cookie hanya pada
metode tulis, dan `AuthProvider` memanggil `/auth/csrf` bila cookie itu hilang.
Dua tes baru mengunci perilaku ini.

**Verifikasi:** `gofmt` bersih, `go build`/`go vet` OK, **10 paket uji lulus**;
frontend `tsc` bersih, ESLint 0 warning, Prettier rapi, **21 tes lulus**, build sukses.

**Catatan untuk Hermes:** tidak ada perubahan pada kontrak engine.

---

### 2026-11-08 — Frontend Console selesai mengikuti kontrak API

**Pelaku:** Claude
**Tipe:** FEATURE
**Status:** SELESAI, typecheck/lint/tes/build bersih

**Ringkasan:**

Seluruh halaman yang dijanjikan `docs/API_CONTRACT.md` sudah dibangun.
Frontend tetap **tanpa logika trading** — hanya menampilkan dan memanggil API.

| Halaman | Endpoint |
|---|---|
| Ringkasan | `/system/status`, `/positions` |
| Sinyal | `/signals` |
| Posisi | `/positions` |
| Akun Broker | `/accounts` (CRUD) |
| Strategi & Langganan | `/strategies`, `/subscriptions` |
| Pengaturan | `/trial`, `/telegram/link` |
| Konfirmasi Entry | `/approve/:token` |

**Aturan kontrak yang dipegang frontend:**

- Tidak ada tombol tutup posisi.
- Sinyal `REJECTED` menampilkan `reject_reason`, tombol mati.
- Strategi hanya bisa dipilih bila `available && allowed`.
- Token akun dikirim sekali, tidak pernah ditampilkan kembali.
- Sinyal tier Free diberi label “Telat 15 menit”.

**Struktur:** tipe di `types/api.ts`, panggilan API di `lib/api.ts`,
query React Query dipusatkan di `lib/queries.ts`, komponen per fitur
di `features/*`, halaman di `pages/*`.

**Verifikasi:** `tsc --noEmit` bersih, ESLint 0 warning, Prettier rapi,
**17 tes lulus**, `vite build` sukses.

**Catatan untuk Hermes:** tidak ada perubahan kontrak engine. Runner tetap
mengirim `subscriber_id` `runner-ila-2.0` / `runner-ila-3.0`.

---

### 2026-11-08 — Backend Console selesai + kontrak API untuk frontend

**Pelaku:** Claude
**Tipe:** FEATURE + SECURITY + BUGFIX
**Status:** SELESAI, terverifikasi end-to-end

**Ringkasan:**

Backend Console (bukan Runner) dilengkapi agar frontend punya kontrak yang
stabil. Uji rantai penuh sudah dijalankan: register → trial → akun broker →
langganan → approve sinyal → Runner membuka posisi Match-Trader.

**1. Bug nyata yang ditemukan uji integrasi**

Lima model tidak punya `Base`, sehingga `ID` tidak terisi dan INSERT gagal
(`null value in column "id"`): `AuditLog`, `NotificationLog`, `SignalEvent`,
`ExecutionRecord`, `OnboardingToken`. Diperbaiki di `internal/model/hooks.go`
(`BeforeCreate` per model) dan dikunci `hooks_test.go`.

**2. Keamanan**

| Item | Implementasi |
|---|---|
| Hash password | bcrypt → **Argon2id** (time=3, memori 64 MB, paralel 4). Hash bcrypt lama tetap diterima + `NeedsRehash` |
| Rate limit login | 5 gagal / 15 menit per `email+IP` → 429 `TOO_MANY_REQUESTS` |
| Trial 7 hari | Sekali per akun, dicek dari fingerprint perangkat + IP saat daftar |
| Trial = **dry-run wajib** | `runner/trial.go` `mustDryRun()` — tidak bisa dimatikan |
| Revoke kredensial | Backend panggil Runner `POST /internal/account/revoke` |

**3. Endpoint baru**

```text
GET    /api/v1/system/status     status engine, market, ringkasan harian (hari = WIB)
GET    /api/v1/trial             sisa hari trial
POST   /api/v1/trial             mulai trial
POST   /api/v1/telegram/link     tautkan chat Telegram via /start <token>
DELETE /api/v1/telegram/link     lepas tautan
POST   /api/v1/telegram/webhook  webhook bot (bukan untuk frontend)
```

Telemetry Runner diperbarui tiap 60 detik (`runner/telemetry.go`) supaya
`/system/status` tidak menampilkan data basi.

**4. Kontrak API untuk frontend**

Dokumen baru `docs/API_CONTRACT.md` — sumber tunggal untuk membangun UI.
Semua respons memakai amplop `{"data": ...}` / `{"error": {"code","message"}}`.

**File terdampak:** `docs/API_CONTRACT.md`, `internal/model/hooks.go`,
`pkg/utils/hash.go`, `internal/middleware/ratelimit.go`,
`internal/runner/{trial,telemetry,settings,rules,subscription,service,brokerpool}.go`,
`internal/service/*_service.go`, `internal/dto/*`, `internal/handler/*`,
`internal/router/router.go`, `cmd/runner/*`.

**Verifikasi:** `gofmt` bersih, `go build`/`go vet` OK, **143 tes lulus**.

---

### 2026-11-07 — Integrasi Runner selesai: adapter Match-Trader, akun broker, langganan

**Pelaku:** Claude
**Tipe:** FEATURE + INTEGRATION

**Ringkasan:**

Runner kini bisa terhubung penuh: menerima sinyal, memutuskan, dan
mengeksekusi ke Match-Trader. Backend sudah punya endpoint untuk mengelola
akun broker dan langganan simbol. Engine ILA 3.0 disiapkan sebagai proses
terpisah.

**1. Adapter Match-Trader (`internal/broker/matchtrader/`)**

Diverifikasi terhadap akun demo nyata (HTTP 200):

| Cek | Hasil |
|---|---|
| Login token | balance 10610.33, equity 11100.27 |
| Spec simbol | XAUUSD `valuePerLot=100`, US100 `20`, EURUSD `100000` |
| Harga live | bid/ask terbaca per simbol |
| Posisi terbuka | 1 posisi terbaca lengkap dengan SL/TP |
| Posisi tertutup | 12 terbaca, alasan TP/SL terpetakan |

Perbaikan penting dibanding `go-trade/pkg/broker/matchtrader_broker.go`:
- **Harga isi diambil dari broker**, bukan dari permintaan
  (`ExecutedPrice: req.OpenPrice` adalah bug yang membuat slippage tak terdeteksi).
- **Nilai per lot dari `multiplier`/`contractSize` broker**, bukan tebakan.
  Diverifikasi dengan PnL asli: US100 20×60.44×0.40 = 483.52 ✓
- Digit SL/TP dari `pricePrecision`, step lot dari `volumeStep`.
- Base URL per akun (bukan hardcode satu domain).
- Token kosong/salah → `ErrInvalidCredentials`, sesi tidak pernah dianggap hidup.

**2. Endpoint backend-api baru**

```
GET    /api/v1/accounts
POST   /api/v1/accounts
PATCH  /api/v1/accounts/:id
DELETE /api/v1/accounts/:id
GET    /api/v1/subscriptions
POST   /api/v1/subscriptions
DELETE /api/v1/subscriptions/:id
GET    /api/v1/strategies
GET    /api/v1/signals
GET    /api/v1/positions
```

Penegakan di backend:
- Free = 0 akun, Basic = 1, Plus = 2.
- Mode `auto` hanya Plus.
- Risiko per trade dijepit 0,10%–1,00%.
- Maksimal 8 simbol dipantau per user.
- ILA 3.0 wajib Plus **dan** sudah dirilis.

**3. Kredensial**

Backend hanya **menulis** blob terenkripsi (`cryptox.Seal`), Runner yang
**membaca** (`Open`). Associated data = `broker_account_id`, sehingga blob
tidak bisa dipindah antar akun. Dua tes mengunci ini:
`TestKredensialTerenkripsi` dan `TestKredensialTerikatAkun`.

**4. Engine ILA 3.0 sebagai proses terpisah**

Runner sekarang membuka satu koneksi per strategi:

```go
engineTargets(cfg) -> [{ila-2.0, addr}, {ila-3.0, addr}]
```

- `ENGINE3_GRPC_ADDR` kosong atau `ILA3_AVAILABILITY=off` → engine ILA 3.0
  **tidak dijalankan sama sekali**.
- `SubscriberID` per strategi (`runner-ila-2.0`, `runner-ila-3.0`) agar dua
  engine tidak saling mengira pelanggan yang sama.
- `ComputeRequirements` tidak meminta ILA 3.0 selama belum dirilis.

**5. Tier Free delay 15 menit**

Sudah ditegakkan di `TradingService.Signals`:
- Sebelum 15 menit → sinyal tidak tampil.
- TTL habis sebelum 15 menit → tidak tampil sama sekali.
- Setelah delay → tampil dengan penanda `delayed: true`.

**File terdampak:**

```
backend/internal/broker/matchtrader/{client,transport,types}.go
backend/internal/broker/matchtrader/{client,live}_test.go
backend/cmd/runner/main.go
backend/internal/config/config.go
backend/internal/dto/{account,trading}_dto.go
backend/internal/handler/{account,trading}_handler.go
backend/internal/repository/{interfaces,broker,query}_repository.go
backend/internal/router/router.go
backend/internal/runner/{settings,rules,subscription,service}.go
backend/internal/service/{account,trading}_service.go
backend/internal/service/{account,trading}_service_test.go
backend/internal/service/credentials.go
.env.example
```

**Status:** SELESAI — `go build`, `go vet`, `gofmt`, `go test` semua hijau.

---

## ⬇️ DARI HUMAN KE HERMES

> Bagian ini hanya diisi oleh Human. Hermes membaca, tidak mengubah.

### 2026-11-07 — Kontrak Runner ↔ Engine (untuk Hermes)

**Ringkasan:** Engine tidak perlu berubah kontrak. Yang perlu diketahui Hermes:

| Aspek | Ketentuan |
|---|---|
| `strategy_version` | `ila-2.0` atau `ila-3.0`; wajib terisi benar |
| `subscriber_id` | `runner-ila-2.0` dan `runner-ila-3.0` (satu per engine) |
| `Subscribe.symbols` | Hanya simbol yang dibutuhkan; tanpa `user_id` |
| `expires_at` | Wajib; Runner menolak sinyal lewat TTL |
| `signal_id` | Unik dan stabil; dipakai sebagai kunci idempoten |
| `SignalUpdate` | `CANCELLED`/`EXPIRED`/`SUPERSEDED` hanya membatalkan sinyal, **bukan** posisi |
| `WantStatusUpdates` | Runner kini mengirim `true` |

**Status:** DIBACA

---

## ⬆️ DARI HERMES KE HUMAN

### [2026-11-11] PERBAIKAN — Pengecualian Radar dari Delay Free Tier, Auto-Seed Watchlist Baru, dan Penyelarasan Port Engine
- **Pelaku:** Hermes
- **Tipe:** PERBAIKAN
- **Status:** 🟢 SELESAI

#### Ringkasan Temuan & Perbaikan:
1. **Radar Menunggu Dikecualikan dari Delay 15 Menit Free Tier (`trading_service.go`):**
   - **Akar Masalah:** Logika `FreeTierDelay` (15m) sebelumnya memfilter seluruh sinyal secara buta. Karena status radar (`WAIT_HTF_SWEEP`, `WAIT_LTF_CISD`) dan setup M5/M15 memiliki TTL pendek (5 s.d. 15 menit), sinyal ditahan saat < 15 menit, lalu langsung di-drop karena sudah kedaluwarsa setelah 15 menit. Akibatnya pengguna Free (dan user baru) tidak pernah melihat progres radar.
   - **Perbaikan:** Menambahkan pengecualian `!model.IsWaitingSignalStatus(signal.Status)` pada filter Free delay. Sinyal radar/progres pasar kini tampil real-time bagi seluruh pengguna, sementara penundaan 15 menit tetap ditegakkan hanya untuk sinyal eksekusi entri (`READY_TO_ENTRY` / `OPPORTUNITY`).
   - **Uji:** Ditambahkan unit test `TestFreeRadarStatusTidakTertunda` di `trading_service_test.go` (PASS).

2. **Auto-Seed Watchlist Default untuk Pendaftaran Baru (`auth_service.go`, `account_service.go`, `router.go`):**
   - **Akar Masalah:** Endpoint `GET /api/v1/signals` mewajibkan join ke tabel `subscription`. Saat user mendaftar, belum ada pair yang dilanggan sehingga backend mengembalikan `[]` dan frontend menampilkan layar kosong ("Watchlist Pair Kosong"). Di saat yang sama, Runner mengirim daftar simbol kosong ke Engine sehingga Engine berhenti memindai pasar.
   - **Perbaikan:** Menambahkan `UserSignupSubscriber` pada alur `authService.Register` dan method `SubscribeDefaults` di `AccountService`. Pengguna baru otomatis didaftarkan ke pair utama (`XAUUSD`, `EURUSD`, `USTEC`), memicu Runner meminta data ke Engine dan dashboard radar langsung terisi.
   - **Uji:** Ditambahkan unit test `TestRegisterSeedsDefaultSubscriptions` dan `TestSubscribeDefaults_CreatesActiveSubscriptions` (PASS).

3. **Penyelarasan Port gRPC Engine di `.env.example`:**
   - Menyesuaikan default `ENGINE_GRPC_ADDR` ke port `9443` dan `ENGINE3_GRPC_ADDR` ke port `9444` sesuai konfigurasi `docker-compose.yml` di `go-trade` (mencegah tabrakan dengan port 9090 penerima log platform).

#### File Terdampak:
- `backend/internal/service/trading_service.go`
- `backend/internal/service/trading_service_test.go`
- `backend/internal/service/account_service.go`
- `backend/internal/service/account_service_test.go`
- `backend/internal/service/auth_service.go`
- `backend/internal/service/auth_service_test.go`
- `backend/internal/router/router.go`
- `.env.example`
- `docs/AGENT_LOG.md`
