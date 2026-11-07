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
| `risk_percent` | 0,10–1,00 (0 = default 0,5) | 400 |
| `token` Match-Trader | wajib, minimal 20 karakter | 400 |
| `ctrader` | belum tersedia | 400 |

> Token **tidak pernah** dikembalikan. Respons hanya memuat
> `has_credential: true`.

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

## 4. Langganan Simbol

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
