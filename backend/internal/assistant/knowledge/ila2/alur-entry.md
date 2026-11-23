# Alur Entry: dari HTF ke LTF

Dokumen ini mendefinisikan alur lengkap dari **konteks HTF (High Timeframe)** sampai **entry di LTF (Low Timeframe)**, termasuk aturan entry, stop loss, dan target.

Gunakan dokumen ini sebagai aturan baku saat mengevaluasi chart candlestick. Jangan menambah aturan di luar dokumen ini. Dokumen ini merujuk ke dokumen lain tanpa menyalin isinya:

| Dokumen | Isi |
|---|---|
| `key-level-irl.md` | SIBI/BISI: pembentukan, invalidasi, iFVG |
| `key-level-erl.md` | Swing high/low: pembentukan, invalidasi, FTM |
| `c2-closure-irl.md` | C2 closure pada SIBI/BISI |
| `c2-closure-erl.md` | C2 closure pada swing high/low |
| `cisd.md` | -CISD dan +CISD di LTF |
| `time-cycle.md` | Time cycle: penentu HTF dan LTF yang dipakai |

---

# Ringkasan Alur

| Tahap | Isi |
|---|---|
| **1. HTF, C2 closure** | Key level **ERL atau IRL** yang valid mendapat **C2 closure**, dan C2 harus **di dalam cycle**. |
| **2. C3** | Candle HTF tepat setelah C2 aktif, dan C3 juga harus **di dalam cycle**. Entry di LTF hanya dicari **selama C3 berjalan** (open sampai close). Jika tidak ada setup, invalid. |
| **3. LTF, CISD** | Cari **CISD** dengan key level **wajib ERL**. |
| **4. Entry** | Di **close candle** yang menembus level CISD. |
| **5. Stop loss** | Di swing high/low yang terbentuk saat harga menyapu ERL. |
| **6. Target** | **RR 1:2**. |

---

# Pasangan Timeframe HTF ke LTF

Setiap HTF punya LTF pasangannya. Konteks HTF dibaca di HTF, dan entry dicari di LTF pasangannya.

HTF dan LTF yang dipakai ditentukan oleh **time cycle** yang dipilih (lihat `time-cycle.md`).

| HTF | LTF |
|---|---|
| Quarterly | Weekly |
| Monthly | Daily |
| Weekly | 4H |
| Daily | 1H |
| 4H | 15m |
| 1H | 5m |
| 15m | 1m |

- Contoh: HTF 4H berarti entry dicari di 15m, dan HTF Daily berarti entry dicari di 1H.
- Jika HTF chart tidak ada di tabel ini, tanyakan kepada pengguna. Jangan menebak LTF-nya.

---

# 1. HTF: C2 Closure

- Konteks HTF boleh dari **ERL** (swing high atau swing low) **atau IRL** (SIBI atau BISI).
- Key level harus **valid**, lalu mendapat **C2 closure** sesuai `c2-closure-erl.md` atau `c2-closure-irl.md`.
- Status C2 bisa **berpindah** atau **gugur**. Yang berlaku adalah **C2 aktif** (C2 terakhir yang belum gugur).

# 2. C3

- Setelah C2 closure, candle HTF berikutnya adalah **C3**, lalu **C4**, dan seterusnya.
- Penamaan ini dihitung dari **C2 aktif**. Jika C2 berpindah ke candle lain, penomoran C3, C4, dan seterusnya mengikuti C2 yang baru.
- Penamaan C2, C3, C4, dan seterusnya **hanya untuk HTF**, tidak dipakai di LTF.
- **C3 wajib di dalam cycle** (lihat `time-cycle.md`), sama seperti C2 closure.
- Entry di LTF hanya dicari **selama C3 berjalan, dari open sampai close**. Jika selama C3 tidak ada setup, maka setup **invalid**.
- Contoh: HTF 4H dengan LTF 15m. Garis "C2 Closure" dan "C3" di chart merujuk ke candle 4H, dan garis itu ikut ditarik di chart 15m pada waktu yang sama.

# 3. Arah Setup

| C2 closure pada | Arah | CISD yang dicari di LTF |
|---|---|---|
| **Swing high** atau **SIBI** | Bearish (short) | **-CISD** |
| **Swing low** atau **BISI** | Bullish (long) | **+CISD** |

# 4. LTF: Mencari CISD

- Entry dicari di LTF **selama C3 berjalan** (open sampai close). Jika tidak ada setup, setup invalid.
- LTF yang dipakai mengikuti **pasangan timeframe** di tabel di atas.
- Aturan CISD (tiga kondisi key level ERL dan konfirmasi body) ada di `cisd.md`.
- Di LTF key level yang dipakai **wajib ERL**. IRL tidak dipakai sebagai key level di LTF.

# 5. Entry, Stop Loss, dan Target

## Entry
- Entry di **close candle yang menembus level CISD**, yaitu candle konfirmasi.

## Stop Loss (Invalidation)

| Arah | Letak stop loss |
|---|---|
| **Short (-CISD)** | Di **swing high yang terbentuk saat harga menyapu ERL**, yaitu **puncak rangkaian candle bullish**. |
| **Long (+CISD)** | Di **swing low yang terbentuk saat harga menyapu ERL**, yaitu **dasar rangkaian candle bearish**. |

## Target
- Target = **RR 1:2** (reward dua kali risiko).

```text
risiko = |entry - stop_loss|

# Short
target = entry - 2 * risiko

# Long
target = entry + 2 * risiko
```

Contoh (short, Gold 15m, angka dari chart): entry ~3.839,93, stop loss ~3.858,28, risiko ~18,35, target ~3.803,24 (jarak ~36,69, yaitu dua kali risiko).

---

# Contoh Alur Lengkap (Bearish)

1. **HTF 1D (Silver):** swing high Daily mendapat C2 closure. Candle berikutnya adalah C3.
2. **LTF 1H, selama C3 berjalan:** swing high 1H valid dan terkena wick di puncak kenaikan.
3. **-CISD:** body candle bearish menembus open candle bullish pertama di rangkaian naik. -CISD terkonfirmasi.
4. **Entry** di close candle yang menembus. **Stop loss** di puncak rangkaian naik. **Target** RR 1:2.

Contoh lain: **HTF 15m (Gold)** dengan C2 closure di **SIBI**, **LTF 1m** dengan swing high 1m yang FTM dan iFVG (kondisi 2), lalu -CISD.

Alur bullish adalah kebalikannya: C2 pada swing low atau BISI, lalu +CISD di LTF dengan swing low, stop loss di dasar rangkaian turun, dan target RR 1:2.

---

# Aturan Penting

- Entry di LTF hanya dicari **selama C3 berjalan** (open sampai close). Jika tidak ada setup di C3, setup invalid.
- **C2 closure dan C3 wajib di dalam cycle.** Setup yang sudah berjalan saat cycle berakhir dibiarkan saja.
- Konteks HTF boleh **ERL atau IRL**, key level di LTF untuk CISD **wajib ERL**.
- Aturan key level berlaku penuh di semua timeframe: invalid hanya karena **close body**, hanya key level sejenis yang **paling baru** yang bisa valid, dan invalid bersifat permanen.
- Konfirmasi dan entry memakai **close body**, bukan wick.

# Panduan Output untuk AI

1. **Timeframe**: sebutkan HTF dan LTF-nya (LTF mengikuti pasangan di tabel). Jika timeframe tidak terlihat di chart, tanyakan kepada pengguna.
2. **HTF**: sebutkan key level (ERL atau IRL), statusnya, C2 closure aktif, dan apakah candle HTF saat ini adalah C3, dan apakah C2 dan C3 berada di dalam cycle.
3. **Arah**: bearish atau bullish, mengikuti key level C2.
4. **LTF**: evaluasi CISD sesuai `cisd.md` (kondisi key level dan konfirmasi).
5. **Rencana posisi** (jika CISD terkonfirmasi): sebutkan entry, stop loss, dan target RR 1:2 beserta hitungannya.
6. Jika CISD belum terkonfirmasi dan C3 masih berjalan, nyatakan statusnya `menunggu` dan sebutkan syarat yang belum terpenuhi. Jika C3 sudah close tanpa setup, nyatakan setup **invalid**.
7. Jika candle masih berjalan (belum close), nyatakan bahwa status belum final.
8. Jika data di chart tidak terbaca jelas, sebutkan nilai yang diperkirakan dan beri tahu bahwa itu perkiraan.
