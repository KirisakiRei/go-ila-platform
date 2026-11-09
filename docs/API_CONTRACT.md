# Kontrak API Backend → Frontend

> **Versi:** 1.0 · **Basis:** `/api/v1`
> Dokumen ini adalah acuan tunggal frontend. Semua endpoint sudah diuji
> end-to-end terhadap PostgreSQL dan Runner.

---

## 1. Bentuk Balasan

Semua balasan memakai amplop yang sama:

```json
{
  "success": true,
  "message": "Pesan untuk manusia",
  "data": { },
  "meta": { }
}
```

Gagal:

```json
{
  "success": false,
  "message": "Alasan yang bisa ditampilkan",
  "error": "BAD_REQUEST"
}
```

### 1.1. Daftar `error`

| Kode | HTTP | Arti |
|---|---|---|
| `BAD_REQUEST` | 400 | Payload tidak sah |
| `UNAUTHORIZED` | 401 | Token tidak ada / kedaluwarsa |
| `FORBIDDEN` | 403 | Tidak berhak (tier, kepemilikan) |
| `NOT_FOUND` | 404 | Data tidak ada |
| `ALREADY_EXISTS` | 409 | Sudah terdaftar |
| `CONFLICT` | 409 | Keadaan menolak operasi |
| `TOO_MANY_REQUESTS` | 429 | Terlalu banyak percobaan |
| `SERVICE_UNAVAILABLE` | 503 | Dependensi tidak terjangkau |
| `INTERNAL_SERVER_ERROR` | 500 | Kegagalan internal |

> Pesan `500` selalu generik. Detail tidak pernah dikirim ke klien.

### 1.2. Autentikasi

Endpoint bertanda 🔒 memerlukan **cookie sesi**. Tidak ada header
`Authorization` — backend tidak menerimanya sama sekali.

| Cookie | HttpOnly | Isi |
|---|---|---|
| `ila_session` | ✅ | JWT sesi, 12 jam |
| `ila_csrf` | ❌ | token anti-CSRF |

Server menetapkan keduanya saat login. Browser mengirimkannya otomatis bila
permintaan memakai `credentials: 'same-origin'`.

**Permintaan pengubah keadaan** (`POST`, `PUT`, `PATCH`, `DELETE`) wajib
menyertakan header:

```
X-CSRF-Token: <nilai cookie ila_csrf>
```

Tanpa header itu atau bila nilainya berbeda → `403`.

> Cookie memakai `SameSite=Strict`, `HttpOnly` (sesi saja), dan `Secure` di
> produksi. Frontend **tidak boleh** menyimpan token di `localStorage`,
> `sessionStorage`, atau variabel yang bisa dibaca ulang setelah reload.

---

## 2. Autentikasi

### POST `/auth/register` — publik

```json
{ "name": "Budi", "email": "budi@contoh.id", "password": "rahasia12345" }
```

Balasan `201`: objek `User`.

Ketentuan: `name` 3–100, `email` valid, `password` 10–72.

### POST `/auth/login` — publik

```json
{ "email": "budi@contoh.id", "password": "rahasia12345" }
```

Balasan `200` beserta dua `Set-Cookie`:

```json
{
  "expires_at": "2026-11-08T02:00:00Z",
  "user": { }
}
```

> Token **tidak** ada di badan balasan. Satu-satunya tempatnya adalah cookie
> `ila_session`.

**Batas percobaan:** 5 gagal per 15 menit per `email + IP` → `429`.
Setelah login berhasil, hitungan direset.

### POST `/auth/logout` — publik

Menghapus kedua cookie. Balasan `200` `{ "logged_out": true }`.
Aman dipanggil tanpa sesi, sehingga frontend tidak perlu menunggu respons.

### GET `/auth/csrf` — publik

Menerbitkan ulang cookie CSRF dan mengembalikannya:

```json
{ "csrf_token": "QwfIQREK..." }
```

Dipanggil frontend setelah sesi dipulihkan, karena cookie CSRF bisa hilang
saat browser dibersihkan sebagian. Token lama otomatis tidak berlaku.

### GET `/me` 🔒

Balasan `200`: objek `User`.

### Objek `User`

| Field | Tipe | Keterangan |
|---|---|---|
| `id` | string | UUID |
| `name` | string | |
| `email` | string | |
| `role` | string | `user` / `admin` / `support` |
| `tier` | string | `free` / `basic` / `plus` |
| `created_at` | string | RFC3339 |

---

## 2.1. RBAC Panel Admin 🛡️

Setiap route `/api/v1/admin/*` dijaga **permission**, bukan sekadar "punya sesi
admin". Frontend wajib memakai daftar permission dari `/admin/auth/me` untuk
menyembunyikan menu dan tombol, supaya operator tidak menekan aksi yang pasti
ditolak `403`.

### GET `/admin/auth/me` 🔒

```json
{
  "user": { "id": "...", "name": "Admin", "email": "admin@...", "role": "admin" },
  "role": { "code": "admin", "name": "Admin", "is_superadmin": false },
  "permissions": ["user.view", "account.view", "cms.edit", "..."]
}
```

Aturan pemakaian di klien:

- `permissions` adalah sumber kebenaran untuk tampil/tidaknya sebuah aksi.
- `role.is_superadmin` dipakai untuk mengunci aksi milik superadmin.
- Bila `permissions` kosong, tampilkan panel tanpa aksi tulis sama sekali.
- Jangan menebak dari `user.role`: role RBAC (`role.code`) bisa berbeda dari
  kolom `user.role`, dan kewenangan dapat diubah admin lain kapan saja.

### Role bawaan

| Role | Isi kewenangan |
|---|---|
| `superadmin` | seluruh permission |
| `admin` | seluruh permission **kecuali** empat kewenangan terkunci di bawah |
| `support` | baca pelanggan, akun, trading, sinyal, status sistem |
| `finance` | billing, harga paket, kupon, kredit, ubah tier |

### Kewenangan yang sengaja dikunci

| Permission | Keterangan |
|---|---|
| `user.delete` | hapus akun pelanggan |
| `system.settings` | lihat & ubah flag sistem |
| `system.maintenance` | ubah mode platform (normal/read-only/pemeliharaan) |
| `billing.settings` | ubah kunci gateway pembayaran |

`system.settings` dan `system.maintenance` **hanya** berlaku untuk role
`superadmin`. Memberikannya lewat role buatan tidak membuat admin biasa bisa
memakainya. Empat kewenangan inilah yang membuat akun `admin` tidak bisa
mengubah sebagian pengaturan meski panel menampilkan menunya.

### Permission per area

| Permission | Route admin yang dijaga |
|---|---|
| `user.view` | `GET /users`, `/users/:id`, `/users/:id/notes`, `/trials` |
| `user.edit` | `POST /users/:id/notes`, `/users/:id/support-notes` |
| `user.suspend` | `POST /users/:id/suspend`, `/unsuspend` |
| `user.ban` | `POST /users/:id/ban`, `/unban` |
| `user.delete` | `POST /users/:id/delete` (superadmin) |
| `user.tier` | `PATCH /users/:id/tier`, `POST /trials/:id/extend`, `/end` |
| `user.legal` | `GET /users/:id/legal` |
| `user.export` | `GET /users-export` |
| `account.view` | `GET /users/:id/accounts` |
| `account.freeze` | `POST /accounts/:id/freeze`, `/unfreeze` |
| `subscription.edit` | seluruh route katalog pair (`/pairs`, `/pair-categories`, `/pair-groups`) |
| `trade.view` | `GET /users/:id/positions` |
| `billing.view` | invoice, kupon, kredit, paket, laporan |
| `billing.manual` / `billing.refund` | `POST /invoices/:id/mark-paid` / `/refund` |
| `billing.plan` | `PATCH /plans/:code`, `PUT /plans/:code/guard` |
| `billing.voucher` | kupon & kredit (`POST/PATCH /coupons`, `/credit-codes`, `/users/:id/credit`) |
| `cms.view` / `cms.edit` / `cms.publish` | dokumen legal, FAQ, pengumuman, isi halaman |
| `role.view` / `role.edit` | `/roles`, `/permissions` |
| `audit.view` | `GET /audit`, `/audit/:id`, `/audit-catalog` |
| `system.view` | status engine, runner, broker, laporan operasional |
| `system.settings` | `/system/settings` (superadmin) |
| `system.maintenance` | `PUT /system/mode` (superadmin) |

> Permission `account.edit`, `account.balance`, `account.delete`,
> `signal.view`, `signal.trace`, `trade.export`, `billing.export`,
> `audit.export`, dan `billing.settings` sudah terdaftar di katalog tetapi
> **belum punya route**. Jangan menampilkan aksinya sebelum endpoint-nya ada.

---

## 3. Akun Broker

### GET `/accounts` 🔒

Balasan `200`: array `Account`.

### POST `/accounts` 🔒

```json
{
  "broker_type": "matchtrader",
  "label": "Akun Demo",
  "mode": "manual",
  "risk_percent": 0.5,
  "is_dry_run": true,
  "token": "<access token Match-Trader>",
  "base_url": ""
}
```

Balasan `201`: objek `Account`.

**Aturan yang ditegakkan:**

| Aturan | Nilai | Error |
|---|---|---|
| Akun per tier | Free 0, Basic 1, Plus 2 | 403 |
| `mode: auto` | hanya Plus | 403 |
| `risk_percent` | 0,10–1,00% per trade (default 0,5%) | 400 |
| Total risiko seluruh akun | Basic maks 1,0%, Plus maks 2,0% | 403 |
| `token` Match-Trader | wajib untuk matchtrader, minimal 20 karakter | 400 |
| `ctrader` | tersedia via alur OAuth `/accounts/ctrader/*` | 400 bila token manual |

> Token **tidak pernah** dikembalikan. Respons hanya memuat
> `has_credential: true`.
>
> **Pagar Risiko:** User **hanya** dapat mengatur `risk_percent` per akun
> (0,10%–1,00%). Besaran lot, Stop Loss, Take Profit, dan rasio Risk:Reward
> **tidak pernah** dapat diubah user; seluruhnya dihitung otomatis oleh Runner
> dan sinyal engine untuk menjaga integritas manajemen risiko.

### GET `/accounts/ctrader/auth-url` 🔒

Mengambil tautan OAuth cTrader Open API untuk menghubungkan akun cTrader.

Balasan `200`:
```json
{
  "auth_url": "https://openapi.ctrader.com/apps/auth?client_id=...&redirect_uri=...&scope=trading"
}
```

### POST `/accounts/ctrader/callback` 🔒

Menukarkan kode otorisasi cTrader OAuth menjadi akun broker terdaftar.

```json
{
  "code": "auth-code-dari-ctrader",
  "label": "cTrader Utama",
  "mode": "manual",
  "risk_percent": 0.5,
  "is_dry_run": false
}
```

Balasan `201`: objek `Account`. Kredensial disimpan terenkripsi dengan keyring internal,
dan token diperbarui otomatis di latar belakang saat kedaluwarsa.

### PATCH `/accounts/:id` 🔒

```json
{ "label": "Baru", "mode": "manual", "risk_percent": 0.75,
  "is_active": true, "is_dry_run": false }
```

Semua field opsional. Balasan `200`: objek `Account`.

### DELETE `/accounts/:id` 🔒

Menghapus kredensial **permanen** dan meminta Runner memutus sesi.
Balasan `200` tanpa `data`.

> Posisi yang masih hidup di broker **tidak** disentuh.

### Objek `Account`

| Field | Tipe | Keterangan |
|---|---|---|
| `id` | string | |
| `broker_type` | string | `matchtrader` |
| `label` | string | nama tampilan |
| `external_account_id` | string | nomor akun di broker |
| `mode` | string | `manual` / `auto` |
| `risk_percent` | number | risiko per trade |
| `is_active`, `is_frozen` | bool | |
| `is_dry_run` | bool | simulasi |
| `has_credential` | bool | kredensial tersimpan |
| `last_verified_at` | string? | |
| `last_error` | string | |
| `connected` | bool | dari telemetry Runner |
| `balance`, `equity`, `free_margin` | number | |
| `last_synced_at` | string? | |

---

## 4. Katalog Pair & Pilihan Simbol per Akun

Katalog pair dikelola terpusat oleh admin (dictionary berstruktur: Kategori → Sub-kategori → Pair) agar pengguna tidak mengetik simbol secara manual dan terhindar dari salah ketik.

Pilihan pair kini disimpan per akun broker, sehingga tiap akun bisa memantau pair yang berbeda dan dieksekusi secara independen oleh Runner.

### GET `/pairs` 🔒

Membaca seluruh katalog pair yang aktif untuk checklist di frontend.

Balasan `200`:
```json
{
  "categories": [
    {
      "key": "forex",
      "label": "Forex",
      "groups": [
        {
          "key": "usd",
          "label": "USD Quote",
          "pairs": [
            { "symbol": "EURUSD", "name": "Euro / US Dollar", "aliases": "", "correlated_symbols": ["GBPUSD"] }
          ]
        },
        {
          "key": "umum",
          "label": "Umum",
          "pairs": [
            { "symbol": "XAUUSD", "name": "Gold / US Dollar", "aliases": "GOLD", "correlated_symbols": ["XAGUSD", "XAUEUR"] }
          ]
        }
      ]
    },
    {
      "key": "indices",
      "label": "Indices",
      "groups": [
        {
          "key": "umum",
          "label": "Umum",
          "pairs": [
            { "symbol": "USTEC", "name": "US Tech 100", "aliases": "NAS100,US100,NDX", "correlated_symbols": ["US500", "US30"] }
          ]
        }
      ]
    }
  ],
  "symbols": ["EURUSD", "USTEC", "XAUUSD"]
}
```

### GET `/accounts/:id/pairs` 🔒

Membaca status checklist simbol yang sedang aktif pada satu akun beserta snapshot katalog.

Balasan `200`:
```json
{
  "account_id": "acc-123",
  "risk_percent": 0.5,
  "selected": ["EURUSD", "XAUUSD"],
  "catalog": { ... }
}
```

### PUT `/accounts/:id/pairs` 🔒

Memperbarui seluruh checklist pair yang aktif pada satu akun (all-or-nothing).

```json
{
  "symbols": ["EURUSD", "USTEC", "gold"]
}
```

Balasan `200`: objek balasan yang sama dengan `GET /accounts/:id/pairs`.

**Aturan yang ditegakkan:**
- Semua simbol divalidasi terhadap katalog aktif. Alias (mis. `gold` untuk `XAUUSD`, `nas100` untuk `USTEC`) otomatis dipetakan ke simbol kanonik engine.
- Bila ada satu simbol tidak sah atau dinonaktifkan admin, seluruh permintaan ditolak `400 BAD_REQUEST` dan tidak ada perubahan disimpan.
- Jumlah simbol terpilih dibatasi oleh tier paket pengguna (Basic maks 8, Plus maks 8).
- Posisi terbuka yang sudah berjalan di broker tetap dikelola Runner sampai ditutup walau simbolnya di-uncheck.

### Korelasi SMT (`correlated_symbols`)

Setiap pair boleh punya **lebih dari satu** instrumen pembanding SMT (mis. `XAUUSD` → `XAGUSD`, `XAUEUR`). Field ini murni **data**: platform hanya menyimpan dan meneruskannya, sedangkan cara membandingkan lilin sepenuhnya milik engine.

- Nilai selalu berupa simbol kanonik, huruf besar, urut abjad, tanpa duplikat.
- Alias yang dikirim admin/admin API dipetakan ke simbol kanonik (`gold` → `XAUUSD`).
- Korelasi diri sendiri (`XAUUSD` → `XAUUSD`) dan simbol yang tidak terdaftar ditolak `400 BAD_REQUEST`.
- Instrumen pembanding yang dinonaktifkan admin otomatis tidak diminta ke engine.
- Runner hanya meminta feed instrumen pembanding untuk strategi evaluasi SMT (`ila-3.0`). Simbol tersebut **tidak** menghasilkan order untuk pengguna yang tidak melangganinya.

---

## 4.1. Manajemen Katalog Pair (Admin Only 🛡️)

Endpoint khusus admin (permission `subscription.edit`):

| Endpoint | Method | Keterangan |
|---|---|---|
| `/admin/pairs` | GET | Membaca katalog lengkap admin (termasuk yang nonaktif & jumlah subscribers) |
| `/admin/pair-categories` | POST | Membuat kategori baru (`key`, `label`, `sort_order`, `is_enabled`) |
| `/admin/pair-categories/:id` | PATCH / DELETE | Mengubah atau menghapus kategori (hapus diblokir bila masih punya pair) |
| `/admin/pair-groups` | POST | Membuat sub-kategori baru (`category_id`, `key`, `label`, dll.) |
| `/admin/pair-groups/:id` | PATCH / DELETE | Mengubah atau menghapus sub-kategori (hapus diblokir bila masih punya pair) |
| `/admin/pairs` | POST | Mendaftarkan simbol baru (`symbol`, `category_id`, `group_id`, `aliases`, `correlated_symbols`, dll.) |
| `/admin/pairs/:id` | PATCH / DELETE | Mengubah atau menghapus simbol (hapus diblokir bila ada subscriber aktif) |

Seluruh mutasi tercatat otomatis pada log audit admin (`pair.category.change`, `pair.group.change`, `pair.change`).

---

## 5. Langganan Simbol (Kompatibilitas Global)

Endpoint warisan untuk memantau simbol secara global (berlaku untuk semua akun aktif pengguna bila `broker_account_id` dikosongkan):

### GET `/subscriptions` 🔒

Balasan `200`: array `Subscription`.

### POST `/subscriptions` 🔒

```json
{ "symbol": "XAUUSD", "strategy": "ila-2.0", "broker_account_id": "" }
```

Balasan `201`: objek `Subscription`.

**Aturan:**

| Aturan | Nilai | Error |
|---|---|---|
| Simbol | dibakukan ke huruf besar, `/` dibuang | 400 |
| Strategi | `ila-2.0` / `ila-3.0` (default `ila-2.0`) | 400 |
| `ila-3.0` | wajib Plus **dan** sudah dirilis | 403 |
| Maksimal simbol aktif | 8 | 403 |
| Melanggan ulang pasangan sama | mengembalikan baris lama (idempoten) | — |

### DELETE `/subscriptions/:id` 🔒

Menonaktifkan (tidak menghapus riwayat). Balasan `200`.

### Objek `Subscription`

| Field | Tipe |
|---|---|
| `id` | string |
| `symbol` | string |
| `strategy` | string |
| `broker_account_id` | string? |
| `is_active` | bool |

---

## 5. Strategi

### GET `/strategies` 🔒

```json
[
  {
    "version": "ila-2.0",
    "name": "ILA 2.0",
    "status": "active",
    "available": true,
    "allowed": true,
    "selected": true,
    "description": "Strategi stabil untuk semua tier."
  },
  {
    "version": "ila-3.0",
    "name": "ILA 3.0",
    "status": "coming_soon",
    "available": false,
    "allowed": true,
    "selected": false,
    "description": "Strategi lanjutan khusus tier Plus."
  }
]
```

| Field | Arti |
|---|---|
| `status` | `active` / `beta` / `coming_soon` |
| `available` | sudah dirilis platform |
| `allowed` | boleh untuk tier user |
| `selected` | sedang aktif oleh user |

> Tombol aktif hanya bila `available && allowed`.
> `selected` hanya untuk **menampilkan** pilihan; toggle dilakukan lewat
> endpoint `/subscriptions`.

---

## 6. Sinyal

### GET `/signals?limit=50` 🔒

Balasan `200`: array `SignalView`, terbaru lebih dulu.
`limit` maksimal 50.

### Objek `SignalView`

| Field | Tipe | Keterangan |
|---|---|---|
| `signal_id` | string | |
| `symbol` | string | |
| `strategy_version` | string | |
| `engine_version` | string | build engine |
| `direction` | string | `BUY` / `SELL` |
| `entry_price`, `stop_loss`, `take_profit` | number | |
| `risk_reward` | number | |
| `status` | string | lihat tabel status |
| `summary`, `next_action` | string | boleh kosong |
| `reject_reason` | string | alasan ditahan |
| `published_at`, `expires_at` | string | RFC3339 |
| `can_confirm` | bool | tombol eksekusi aktif |
| `delayed` | bool | sinyal tier Free (telat 15 menit) |

### Status sinyal → tampilan

| Status | Badge | Tombol |
|---|---|---|
| `READY_TO_ENTRY` | 🟢 Siap | aktif bila `can_confirm` |
| `ACKED` | 🟡 Diterima | aktif bila `can_confirm` |
| `REJECTED` | 🟠 Tertahan | mati + `reject_reason` |
| `EXPIRED` | ⚪ Kedaluwarsa | mati |
| `CANCELLED` | ⚪ Dibatalkan | mati |
| `SUPERSEDED` | ⚪ Digantikan | mati |
| `CONSUMED` | ✅ Sudah dieksekusi | mati |
| `SKIPPED` | ⚪ Dilewati | mati |

**Tier Free:** sinyal baru muncul setelah 15 menit dan diberi `delayed: true`.
Sinyal yang TTL-nya habis sebelum 15 menit **tidak muncul sama sekali**.

---

## 7. Posisi

### GET `/positions?limit=100` 🔒

Balasan `200`: array `Position`, terbaru lebih dulu.

### Objek `Position`

| Field | Tipe |
|---|---|
| `id` | string |
| `symbol` | string |
| `strategy_version` | string |
| `direction` | string |
| `ticket_number` | string |
| `lot_size` | number |
| `open_price`, `stop_loss`, `take_profit` | number |
| `exit_price` | number |
| `profit_loss` | number |
| `outcome_r` | number |
| `risk_percent` | number |
| `status` | string |
| `is_dry_run` | bool |
| `opened_at` | string |
| `closed_at` | string? |

Status posisi: `OPEN`, `CLOSED_TP`, `CLOSED_SL`, `CLOSED_MANUAL`, `UNTRACKED`.

> Frontend **tidak boleh** menyediakan tombol tutup posisi. Posisi hanya
> berakhir lewat SL/TP broker atau aksi user di aplikasi broker.

---

## 8. Status Sistem

### GET `/system/status` 🔒

```json
{
  "market": { "state": "OPEN", "label": "Pasar buka", "updated_at": "..." },
  "engines": [
    { "strategy": "ila-2.0", "connected": true,
      "engine_version": "v1.4.2", "last_seen_at": "..." }
  ],
  "brokers": { "total": 1, "connected": 1, "any_error": false },
  "daily": {
    "day": "2026-11-07",
    "win_count": 0, "loss_count": 0, "total_trades": 0,
    "max_trades": 2, "open_count": 0,
    "locked": false, "lock_reason": ""
  },
  "server_time": "2026-11-07T15:00:00Z",
  "daily_timezone": "Asia/Jakarta"
}
```

| Field | Arti |
|---|---|
| `market.state` | `OPEN` / `CLOSED` / `ROLLOVER` / `UNKNOWN` |
| `engines[].connected` | Runner sedang tersambung ke engine itu |
| `brokers.connected` | jumlah akun yang sesi broker-nya hidup |
| `brokers.any_error` | ada akun dengan `last_error` |
| `daily.day` | hari **WIB**, bukan UTC |

> Bila `daily.locked` bernilai `true`, entry baru ditahan sampai besok.
> `lock_reason` menjelaskan sebabnya.

---

## 9. Uji Coba (Trial)

### GET `/trial` 🔒

### POST `/trial` 🔒

```json
{
  "used": true,
  "active": true,
  "duration_days": 7,
  "days_remaining": 7,
  "dry_run_only": true,
  "trial_started_at": "...",
  "trial_expires_at": "..."
}
```

| Aturan | Ketentuan |
|---|---|
| Sekali per akun | percobaan kedua → `409` |
| Durasi | 7 × 24 jam |
| Setara tier | Plus |
| Eksekusi | **selalu simulasi** (`dry_run_only: true`) |
| Setelah habis | turun otomatis ke Free |

> `dry_run_only` selalu `true` dan **tidak bisa** dimatikan. Runner memaksa
> simulasi untuk user yang sedang trial, apa pun setelan akunnya.

---

## 10. Telegram

### POST `/telegram/link` 🔒

Balasan `201`:

```json
{ "url": "https://t.me/ILABot?start=<token>", "expires_at": "..." }
```

Token 32 byte, berlaku **5 menit**, sekali pakai. Yang disimpan hanya hash.

### DELETE `/telegram/link` 🔒

Memutus chat dari akun. Balasan `200`.

### POST `/telegram/webhook` — publik, bukan untuk frontend

Diautentikasi header `X-Telegram-Bot-Api-Secret-Token`.
**Frontend tidak boleh memanggil endpoint ini.**

---

## 11. Konfirmasi Entry (Deep-Link)

### GET `/approve/:token` 🔒

Menampilkan detail sebelum konfirmasi:

```json
{
  "signal_id": "sig-...",
  "symbol": "XAUUSD",
  "direction": "BUY",
  "strategy_version": "ila-2.0",
  "entry_price": 4120.5,
  "stop_loss": 4112.3,
  "take_profit": 4149.2,
  "risk_reward": 2.3,
  "expires_at": "...",
  "broker_account_id": "...",
  "can_confirm": true,
  "blocked_reason": ""
}
```

Bila `can_confirm` false, tampilkan `blocked_reason` apa adanya.

### POST `/approve/:token/confirm` 🔒

### POST `/approve/:token/decline` 🔒

Balasan:

```json
{ "signal_id": "sig-...", "status": "CONSUMED", "message": "..." }
```

**Wajib sesi login:** token sah **dan** pemiliknya harus sama. Tanpa login,
frontend menampilkan "Silakan masuk" dan token tetap berlaku sampai habis.

Token dihapus setelah dipakai, apa pun hasilnya.

---

## 12. Aturan yang Tidak Boleh Dilanggar Frontend

| ❌ | Alasan |
|---|---|
| Menghitung ulang sinyal atau SL/TP | Milik engine |
| Menyediakan tombol tutup posisi | Posisi dikelola broker |
| Menampilkan token/kredensial | Tidak pernah dikirim |
| Memanggil `/internal/*` Runner | Hanya backend-api |
| Memakai data user lain | Isolasi wajib |
| Menyimpan token di `localStorage` / `sessionStorage` | Sesi hanya di cookie HttpOnly |
| Memakai header `Authorization` | Backend menolaknya; sesi hanya lewat cookie |
| Mengirim `POST`/`PATCH`/`DELETE` tanpa `X-CSRF-Token` | Ditolak `403` |
| Menganggap `message` sebagai kode | Pakai `error` untuk logika |
| Menampilkan ILA 3.0 aktif bila `available: false` | Selalu patuhi `available && allowed` |
| Memanggil `POST /telegram/webhook` | Bukan untuk klien |
| Menyediakan input lot, RR, atau SL/TP | Hanya `risk_percent` (0,10–1,00%) yang boleh diatur user per akun; lot & SL/TP dihitung otomatis oleh Runner & engine |
| Menampilkan aksi admin yang tidak ada di `permissions` | Pengguna hanya akan menerima `403`; sembunyikan menunya sejak awal |
