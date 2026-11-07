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

> **Commit terakhir saat log ini ditulis:** `d157941`
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

> Bagian ini hanya diisi oleh Hermes.

_(belum ada entri)_
