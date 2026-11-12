# PLAN IMPLEMENTASI — Per-Symbol Timeframe Pairs (watch & auto)

Status: Fase 1 SELESAI. Fase 2–5 belum.
Repo: `go-trade` (engine, branch `develop`) + `go-ila-platform` (branch `main`).

---

## 0. Ringkasan masalah (terverifikasi dari kode + DB produksi)

`SubscribeRequest.Pairs` adalah satu daftar global untuk SEMUA simbol.
`timeframePairs()` (main.go:654) menggabung pair semua simbol menjadi satu map,
lalu daftar yang sama dikirim untuk setiap simbol.

Akibat: satu-satunya baris switch di DB adalah `USTEC · D·15M`, sehingga
XAUUSD (dan semua simbol lain) hanya dievaluasi pada `D·15M`.
Log produksi: `"rinci":"EURUSD:1 GBPUSD:1 US30:1 US500:1 USTEC:1 XAGUSD:1 XAUEUR:1 XAUUSD:1"`
(padahal ILA 3.0 di UI menunjukkan `XAUUSD:4`).

### Dua bug tersembunyi yang WAJIB ikut diperbaiki
1. `timeframePairs()` membuang identitas simbol saat membangun map union.
2. `hitungPasangan()` (service.go:1313) membentuk fingerprint
   `nama+":"+p.Htf+"/"+p.Ltf` — **simbol juga dibuang**. Akibatnya perubahan
   switch satu simbol bisa TIDAK mengubah fingerprint, sehingga sync tidak
   pernah terpicu (interval sync 2 menit).

### Yang TIDAK boleh disentuh (sudah benar, diverifikasi)
| Komponen | Status |
|---|---|
| tabel `account_pair_timeframe` | punya `Symbol` di unique key |
| `Set(userID, accountID, symbol, ...)` | sudah per-simbol |
| validasi `auto` wajib punya `watch` | sudah ada di service |
| UI matrix `SaklarTimeframeAkun.tsx` | sudah benar |
| `putuskanTimeframe()` (gate eksekusi, Layer 2) | benar — JANGAN diubah |
| `slotDue()`, `menungguLTF()`, `latestCandleClose()` (NY close) | benar — JANGAN diubah |
| `WantedPairs()` saat ini (registry.go:78) | hanya pembaca; dirombak di Fase 2 |

---

## 1. Kontrak gRPC (Fase 1 — SELESAI)

```proto
message SubscribeRequest {
  string subscriber_id = 1;
  repeated string symbols = 2;
  repeated Pair pairs = 3;                    // DIPERTAHANKAN (backward compat)
  bool want_status_updates = 4;
  map<string, string> symbol_feeds = 5;
  map<string, SymbolPairs> symbol_pairs = 6;  // BARU
}

message SymbolPairs {
  repeated Pair pairs      = 1;
  bool          restricted = 2;
}
```

Verifikasi yang sudah dilakukan:
- [x] `go-trade/proto/engine/v1/engine.proto` — ditambah
- [x] `go-ila-platform/proto/engine/v1/engine.proto` — ditambah (identik, hanya `go_package` beda)
- [x] `go-trade`: `buf generate proto --path proto/engine/v1/engine.proto` → sukses
- [x] `go-ila-platform`: `buf generate --template buf.gen.tmp.yaml proto` → sukses
- [x] Keduanya: `SymbolPairs` di **field 6** (`protobuf:"bytes,6,rep,name=symbol_pairs..."`)
- [x] `go build ./...` hijau di DUA repo
- [x] `engine_grpc.pb.go` TIDAK berubah (service tidak berubah → aman)
- [x] Artefak nyasar `gen/proto/` sudah dihapus

### Matriks kompatibilitas (kontrak perilaku — patuhi persis)
| Kondisi | Perilaku engine |
|---|---|
| `symbol_pairs` kosong + `pairs` kosong | semua pair (legacy) |
| `symbol_pairs` kosong + `pairs` terisi | pakai `pairs` global (legacy) |
| `symbol_pairs[X].restricted=false` | set default penuh untuk X |
| `symbol_pairs[X].restricted=true` + isi | HANYA itu untuk X |
| `symbol_pairs[X].restricted=true` + kosong | X TIDAK dievaluasi (Opsi A) |
| `symbol_pairs` terisi tapi X tidak ada | default penuh untuk X (BUKAN nil) |
| satu subscriber `restricted=false` | pembatasan simbol itu batal |

---

## 2. Fase 2 — Engine (`go-trade`, branch `develop`)

### 2.1 `internal/engine/registry.go`
- [ ] Tambah tipe:
  ```go
  // PairPolicy adalah kebijakan pair untuk satu simbol.
  type PairPolicy struct {
      Pairs      []Pair
      Restricted bool
  }
  ```
- [ ] Tambah field di `Subscriber`:
  ```go
  // SymbolPairs adalah kebijakan pair per simbol. Bila nil/kosong, engine
  // memakai Pairs global (perilaku lama).
  SymbolPairs map[string]PairPolicy
  ```
- [ ] Tambah method `SetSymbolPairs(id string, sp map[string]PairPolicy) bool`
      (mengikuti pola `SetPairs`, normalize tiap `Pairs`).
- [ ] **Rombak `WantedPairs(symbol)`** (baris 78) agar membaca kebijakan per-simbol saja:
  - kumpulkan `policies` dari semua subscriber yang memantau `symbol`;
    prioritas: `SymbolPairs[symbol]` bila ada, jika tidak → fallback `s.Pairs`
    (bila `len(s.Pairs)==0` berarti "tidak membatasi" → `return nil,false`).
  - `return nil,false` bila ada satu saja subscriber dengan `Restricted=false`
    yang memantau symbol itu.
  - bila semua restricted: gabung pair; **bila hasil gabungan kosong
    → `return []Pair{}, true`** (symbol sengaja tidak dievaluasi).
  - Pertahankan signature `([]Pair, bool)` agar `publisher.go` tidak perlu
    diubah struktural.

### 2.2 `internal/engine/publisher.go` — `pairsFor()` (baris 190–229)
- [ ] Ubah cabang `if len(out)==0 { return nil }` menjadi:
  ```go
  if len(out) == 0 {
      if !dibatasi { return semua }   // tidak ada yang membatasi
      // semua pair dimatikan sengaja: jangan evaluasi apa pun
      p.warnSemuaMati(symbol)         // WARN sekali per simbol (dedup)
      return nil                      // nil = tidak ada pair valid
  }
  ```
  ⚠️ **CATATAN PENTING:** saat ini `return nil` sudah berarti "tidak evaluasi",
  dan `dibatasi=true` + kosong juga menghasilkan `nil`. Jadi perubahan ini
  **fungsional netral** — tujuannya hanya menambah WARN diagnostik. Jangan
  sampai salah mengubah `dibatasi=false` menjadi `nil` (itu akan mematikan
  engine untuk simbol yang belum dikonfigurasi).
- [ ] Tambah `warnSemuaMati` dengan dedup (`map[string]struct{}` + mutex), pesan
      `"evt":"pairs.semua.mati"`, field `symbol`.

### 2.3 `internal/grpcserver/server.go`
- [ ] Di `Subscribe()` (~baris 76): isi `SymbolPairs: kebijakanDari(req.GetSymbolPairs())`.
- [ ] Di `StreamEvents()` (~baris 133): idem.
- [ ] Tambah helper `kebijakanDari(map[string]*enginev1.SymbolPairs) map[string]engine.PairPolicy`.
- [ ] Perluas log subscribe: `"symbol_pairs", len(req.GetSymbolPairs())`.

### 2.4 Test — `internal/engine/pasangan_test.go`
- [ ] Symbol tanpa policy → set penuh (`dibatasi=false`).
- [ ] Satu pair → 1 pair (`dibatasi=true`).
- [ ] `restricted=true` + kosong → **tidak dievaluasi**.
- [ ] **KEY TEST:** dua simbol dengan policy berbeda TIDAK bocor
      (XAUUSD:2 pair, EURUSD:4 pair → masing-masing dapat miliknya).
- [ ] Jalur legacy `pairs` global masih jalan.
- [ ] Satu subscriber `restricted=false` membatalkan pembatasan.
- [ ] `go test ./internal/engine/ ./internal/grpcserver/` hijau.

---

## 3. Fase 3 — Platform (`go-ila-platform`, branch `main`)
Boleh paralel dengan Fase 2 (kontrak sudah final).

### 3.1 `backend/internal/runner/timeframe_requirements.go`
- [ ] Tambah tipe:
  ```go
  // SymbolPairPolicy adalah kebijakan pair untuk satu simbol pada satu metode.
  type SymbolPairPolicy struct {
      Pairs      []TimeframeRequirement
      Restricted bool
  }
  ```
- [ ] Tambah `ComputeSymbolPairs(ctx, symbol) (map[string]SymbolPairPolicy, error)`:
  - flag OFF → `Restricted=false` (semua pair)
  - helper `s.timeframeSwitchActive(ctx)` → **reuse**, jangan duplikasi.
  - `TimeframeSwitches` error → kembalikan error (pemanggil sudah `continue`).
  - tidak ada baris switch untuk simbol → `Restricted=false` untuk metode itu
  - ada baris → `Restricted=true`, isi hanya `state.Watch==true`
    (boleh kosong = semua dimatikan)
  - urutan deterministik (pakai `sort.Slice` yang sudah ada)
- [ ] **Jangan hapus** `ComputeTimeframeRequirements` — masih dipakai untuk
      membangun `Pairs` legacy. Tandai sebagai `Deprecated`.

### 3.2 `backend/internal/runner/service.go` — `hitungPasangan()` (baris 1313)
- [ ] **Sertakan simbol & restricted di fingerprint.** Format baru:
  ```
  symbol + ":" + strategy + ":" + htf + "/" + ltf
  ```
  plus penanda eksplisit per simbol: `symbol + ":" + strategy + ":restricted=0|1"`.
  Tujuan: perubahan switch satu simbol SELALU mengubah fingerprint → sync jalan.

### 3.3 `backend/cmd/runner/main.go`
- [ ] Tambah `symbolPairs(ctx, target, svc, symbols, logger) map[string]*enginev1.SymbolPairs`
      — **tanpa union lintas simbol**; satu entri per simbol.
- [ ] Ganti `timeframePairs()` menjadi pembangun `Pairs` legacy saja
      (union seperti sekarang, dipertahankan agar engine lama tetap benar).
- [ ] Update dua call site:
  - baris 461 (`StreamEvents`): tambah `SymbolPairs: symbolPairs(...)`
  - baris 538 (`Subscribe`): tambah `SymbolPairs: symbolPairs(...)`
- [ ] Log saat menyusun: `"symbol_pairs", len(...)`.

### 3.4 Test
- [ ] `timeframe_requirements_test.go`: flag OFF → `Restricted=false`;
      simbol tanpa baris → `Restricted=false`; simbol dengan 2 watch → 2 pair;
      simbol semua OFF → `Restricted=true` + kosong.
- [ ] `pairs_sync_test.go`: fingerprint **berubah** saat pair satu simbol diubah
      (ini regresi bug #2).
- [ ] Test anti-bleed: XAUUSD 2 pair, EURUSD 4 pair → tidak saling menimpa.
- [ ] `go test ./internal/runner/` hijau (hati-hati: `internal/router` flaky,
      re-run bila panic di `EvaluateAllClosedPositions`).

### 3.5 UI empty state — `frontend/src/features/signals/components/TimeframeRadarMatrix.tsx`
- [ ] `SlotEmptyState` (baris 166) dapat prop/varian baru `reason`:
      `'scanning'` (sekarang) vs `'no-timeframe-active'`.
- [ ] Varian baru: teks eksplisit "Tidak ada timeframe yang aktif dipindai untuk
      {symbol}" + hint ke Settings (switch watch). **Tanpa warna hijau
      tambahan, ikuti style yang ada.**
- [ ] Data source: slot tidak muncul / `Restricted && len==0` dari API.
      ⚠️ Perlu keputusan: apakah backend mengirim flag `restricted` ke API
      signal-slots? Jika belum, Fase ini bisa memakai sinyal "slot kosong
      semua" sebagai heuristik — tandai TODO.
- [ ] Test di `frontend/src/test/signal-slots.test.ts`.

---

## 4. Fase 4 — Deploy (URUTAN WAJIB)
1. [ ] Push + build **Engine dulu** (`go-trade`). Engine baru tetap menerima
       `Pairs` legacy → aman walau runner belum diupdate.
2. [ ] Verifikasi log engine normal (radar tetap dievaluasi tiap menit).
3. [ ] Baru build **Runner** (`go-ila-platform`).
4. [ ] Set `TIMEFRAME_SWITCH_ENABLED=true` di `.env` VPS 2 (sekarang sudah true).
5. [ ] Verifikasi.

⚠️ JANGAN deploy Runner sebelum Engine — engine lama mengabaikan `symbol_pairs`,
bug tetap ada tanpa gejala.

---

## 5. Fase 5 — Verifikasi
```bash
docker logs --since 5m ila-engine-engine-1  | grep 'radar dievaluasi'   # VPS 1
docker logs --since 5m ila-engine-engine3-1 | grep 'radar dievaluasi'
```
- [ ] XAUUSD 2 switch → `XAUUSD:2`
- [ ] EURUSD 4 switch → `EURUSD:4`
- [ ] Keduanya tidak saling bocor
- [ ] XAUUSD ILA 3.0 4 switch → `XAUUSD:4`, muncul 4 slot baru
- [ ] Semua OFF → simbol absen + WARN `pairs.semua.mati`
- [ ] UI menampilkan empty state eksplisit

---

## 6. Agent logs (WAJIB, dua repo)
- [ ] `go-trade/docs/AGENT_LOG.md` → bagian **⬇️ DARI HUMAN KE HERMES**
      (sesuai aturan file itu; TIDAK ada bagian "DARI CLAUDE" di repo ini).
      Isi: peringatan `hermes` branch tidak disentuh; jangan hapus field 3.
- [ ] `go-ila-platform/docs/AGENT_LOG.md` → bagian **⬆️ DARI CLAUDE KE HUMAN & HERMES**.
- [ ] Commit HANYA `docs/AGENT_LOG.md`:
      `git commit -m "..." -- docs/AGENT_LOG.md`
      (JANGAN `git commit -a` — ada `frontend/src/temp_screens/` milik agent lain).

---

## 7. Titik rawan / larangan
| Larangan | Alasan |
|---|---|
| Hapus field `pairs = 3` | jalur rollback + kompatibilitas legacy |
| Deploy Runner sebelum Engine | bug tetap ada, tanpa gejala |
| Sentuh `origin/hermes` | milik Hermes |
| Ubah `putuskanTimeframe`, `slotDue`, `menungguLTF`, `latestCandleClose` | sudah benar |
| Tulis "DARI CLAUDE" di `go-trade/docs/AGENT_LOG.md` | tidak ada bagian itu |
| `git stash` saat merge berjalan | gagal "could not write index" |
| Mengubah `go.mod` backend turun ke 1.23 | builder `golang:1.24-alpine` |
| Ubah `dibatasi=false` → `nil` di `pairsFor` | akan mematikan simbol belum dikonfigurasi |
| Hapus ordering `status-%` di `query_repository.go` | matrix jadi kosong lagi |
