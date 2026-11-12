# PLAN — Kartu Overview Signals & Radar (grid slot per metode)

Status: BELUM DIKERJAKAN.
Basis: `frontend/src/pages/SignalsPage.tsx` (blok `allPairs.map`, ± baris 526–650).

---

## 0. Koreksi dari human (WAJIB dipatuhi)

1. **Peringatan "arah belum sepakat" DIBATALKAN.**
   Tiap pasangan timeframe bergerak sendiri-sendiri. `W · H4` SELL dan `H1 · M5` BUY
   adalah kondisi normal, bukan konflik. Jangan ada elemen UI apa pun yang
   membandingkan arah antar slot.

2. **Penanda "◀ slot terdekat" DIBATALKAN.**
   Penanda panah/teks seperti itu bukan bahasa UI. Penggantinya: aksen visual
   pada kartu slot (border/latar), tanpa simbol panah.

3. **Aksen warna diperbolehkan**, karena `emerald`/`rose` SUDAH dipakai di
   halaman ini untuk BUY/SELL dan profit/loss. Memakainya untuk arah =
   konsisten, bukan gaya baru.

---

## 1. Masalah yang diperbaiki (terverifikasi dari production)

| # | Masalah | Bukti |
|---|---|---|
| 1 | Harga palsu `@ 0,0000` | Semua baris radar punya `entry_price=0`, `risk_reward=0` |
| 2 | Slot dipilih acak | `pairSigs.find(...)` ambil baris pertama, arah bisa kebalik dari 3 slot lain |
| 3 | Tahap tidak informatif | Semua jatuh ke label `Pantau HTF` |
| 4 | Timeframe tidak terlihat | Tidak ada info `D1 · H1` / `H4 · M15` di overview |

Data status nyata di production saat plan disusun:
```
ila-2.0 | WAIT_HTF_SWEEP        | 18
ila-2.0 | WAIT_LTF_CISD         |  1
ila-2.0 | WAIT_HTF_CANDLE_CLOSE |  1
ila-3.0 | WAIT_ILA3_HTF_KEY_LEVEL | 32
```

---

## 2. Label tahap (keputusan final)

### ILA 2.0 — alur: Sweep → Candle → CISD → Ready
| Status engine | Label |
|---|---|
| `WAIT_HTF_SWEEP` | `Sweep` |
| `WAIT_HTF_CANDLE_CLOSE` | `Candle` |
| `WAIT_LTF_CISD` | `CISD` |
| `READY_TO_ENTRY` | `Ready` |
| `INVALIDATED` | `Batal` |

### ILA 3.0 — alur: Key Level → Reversal → CISD 1 → Ready 1 → Expansi 1 → Retrace → CISD 2 → Ready 2
| Status engine | Label |
|---|---|
| `WAIT_ILA3_HTF_KEY_LEVEL` | `Key Lvl` |
| `WAIT_ILA3_MTF_REVERSAL` | `Reversal` |
| `WAIT_ILA3_LTF_CISD_1` | `CISD 1` |
| `READY_ENTRY_1` | `Ready 1` |
| `ILA3_EXPANSION_1_ACTIVE` | `Expansi 1` |
| `WAIT_ILA3_MTF_FVG_RETR` | `Retrace` |
| `WAIT_ILA3_LTF_CISD_2` | `CISD 2` |
| `READY_ENTRY_2` | `Ready 2` |
| `ILA3_EXPANSION_2_ACTIVE` | `Expansi 2` |
| `ILA3_TARGET_REACHED` | `Target` |

Catatan: `READY_ENTRY_1/2` dipetakan backend ke `READY_TO_ENTRY`, jadi label
`Ready` sudah menangani keduanya. Entri di atas hanya jaring pengaman.

---

## 3. Perubahan berkas

### 3.1 BARU — `frontend/src/features/signals/lib/slot-stage.ts`
Fungsi murni, mudah diuji, tanpa React.

- [ ] `type SlotStage = { label: string; rank: number; ready: boolean }`
  - `rank` dipakai untuk mengurutkan slot (0 = paling dekat entry).
- [ ] `stageILA2(status: string): SlotStage`
- [ ] `stageILA3(status: string): SlotStage`
- [ ] `stageOf(strategy, status): SlotStage` — dispatcher.
- [ ] `slotPalingDekat(slots: TimeframeSlot[], strategy): string | null`
      mengembalikan `key` slot dengan `rank` terendah (dipakai untuk aksen).

### 3.2 `frontend/src/features/signals/lib/signal-slots.ts`
- [ ] Tambah `ringkasanSlot(slots: TimeframeSlot[])` → `{ total, ready, terdekatKey }`.
      Dipakai header kartu untuk "4 slot · 1 ready".

### 3.3 `frontend/src/pages/SignalsPage.tsx` — blok kartu overview
Ganti isi `CardContent` (ILA 2.0 sub-card + ILA 3.0 sub-card) dengan grid slot.

- [ ] Ganti `pairSigs.find(...)` menjadi pemetaan per slot:
      `mapSignalsToSlots(pairSigs.filter(byStrategy), strategy)`.
      (pakai `mapSignalsToSlots` yang sudah ada — jangan bikin mapper baru)
- [ ] Header kartu: `{n} Setup` tetap; tambah `· {ready} ready` bila `ready > 0`.
- [ ] Blok ILA 2.0: grid `grid-cols-4` (desktop) / `grid-cols-2` (mobile).
- [ ] Blok ILA 3.0: sama, **dilipat** secara bawaan (hemat tinggi kartu).
- [ ] Satu komponen kecil lokal `SlotCell` di dalam berkas yang sama:
      - baris 1: `slot.label` (mis. `H4 · M15`) — `font-mono text-[10px]`
      - baris 2: arah — `BUY`/`SELL` (emerald/rose, ikon TrendingUp/Down),
        atau `—` bila `NEUTRAL`
      - baris 3: label tahap dari `slot-stage.ts`
      - **harga Entry/SL/TP hanya bila `stage.ready`**; selain itu tidak ada angka.
- [ ] Aksen slot terdekat: `border-foreground/60 bg-secondary` untuk slot
      `rank` terendah; slot `ready` memakai `border-foreground` penuh.
      **Tanpa panah, tanpa teks tambahan.**
- [ ] Hapus pemakaian `formatNumber(latestIla2.entry_price, 4)` pada baris radar.

### 3.4 Tidak diubah
- `StatusPill`, `StatusDot`, token `--success` (sistem monokrom) — jangan disentuh.
- `SignalDetail`, `TimeframeRadarMatrix`, chart — di luar cakupan.
- Layout desktop halaman (dua kolom) — tetap.

---

## 4. Aturan tampilan (biar tidak menambah gaya)

- Warna arah: `text-emerald-600` / `text-rose-600` (sudah dipakai di halaman ini).
- Aksen slot: `bg-secondary` + `border-foreground/60` (token yang sudah ada).
- Tidak ada animasi, tidak ada timer, tidak ada carousel.
- Tidak ada progress bar berwarna.
- Grid tampil semua slot sekaligus — tidak ada yang disembunyikan bergiliran.

---

## 5. Test

### `frontend/src/test/slot-stage.test.ts` (baru)
- [ ] `WAIT_HTF_SWEEP` → `Sweep`, rank di bawah `CISD` dan `Ready`
- [ ] `WAIT_LTF_CISD` → `CISD`
- [ ] `READY_TO_ENTRY` → `Ready`, `ready === true`
- [ ] status tak dikenal → label netral, tidak `undefined`
- [ ] `WAIT_ILA3_HTF_KEY_LEVEL` → `Key Lvl`
- [ ] `WAIT_ILA3_LTF_CISD_1` → `CISD 1`
- [ ] urutan rank: `Key Lvl` < `Reversal` < `CISD 1` < `Ready`
- [ ] `slotPalingDekat` memilih slot `CISD` di atas slot `Sweep`

### `frontend/src/test/signal-slots.test.ts` (tambah)
- [ ] `ringkasanSlot` menghitung `ready` dengan benar
- [ ] 4 slot ILA 2.0 semuanya terpetakan dari 4 baris radar

### Regresi yang dijaga
- [ ] Slot `WAIT_*` TIDAK menampilkan angka harga apa pun.
- [ ] Slot `READY` menampilkan Entry/SL/TP/R:R.

---

## 6. Verifikasi

```bash
cd go-ila-platform/frontend
npx tsc --noEmit -p tsconfig.json     # harus bersih
npx vitest run                        # 20 file / 114+ test lulus
npx vite build                        # sukses
```

Cek manual di halaman Signals & Radar:
- [ ] Tidak ada lagi `@ 0,0000` maupun `R:R 1:0,0` pada kartu pair.
- [ ] Keempat slot ILA 2.0 tampil beserta label timeframenya.
- [ ] ILA 3.0 terlipat, bisa dibuka.
- [ ] Slot paling dekat entry punya aksen lebih tegas.
- [ ] Mobile: grid 2 kolom, tidak meluber.

---

## 7. Risiko & penanganan

| Risiko | Penanganan |
|---|---|
| `mapSignalsToSlots` butuh `strategy` tepat | Saring `pairSigs` per `strategy_version` dulu |
| Kartu jadi tinggi (8 slot) | ILA 3.0 dilipat bawaan |
| `pairSigs` hanya 50 sinyal | Sudah cukup: radar selalu diurutkan lebih dulu oleh `ListForUser` |
| Label ILA 3.0 panjang (`Expansi 1`) | `text-[10px]` + `whitespace-nowrap`, kolom desktop cukup (4 × ~90px) |
| `formatNumber` masih diimpor padahal tak terpakai | Cek `tsc`/lint, hapus impor bila jadi yatim |
