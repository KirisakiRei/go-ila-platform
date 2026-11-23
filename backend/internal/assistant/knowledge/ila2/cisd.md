# CISD (Change in State of Delivery) pada LTF

Dokumen ini mendefinisikan **CISD** pada **LTF (Low Timeframe)**:

- **Bearish CISD (-CISD)** (Bagian A)
- **Bullish CISD (+CISD)** (Bagian B), yang merupakan **kebalikan (mirror)** dari bearish

Gunakan dokumen ini sebagai aturan baku saat mengevaluasi chart candlestick. Jangan menambah aturan di luar dokumen ini. Definisi key level ERL (swing high/low), FTM, dan iFVG ada di `key-level-erl.md` dan `key-level-irl.md`. Alur lengkap dari HTF ke entry ada di `alur-entry.md`.

---

# Konteks

- CISD dipakai pada konteks **LTF**, sesuai pasangan HTF > LTF di bawah (misalnya LTF 15m untuk HTF 4H, atau LTF 1m untuk HTF 15m).
- Key level yang dipakai di LTF **wajib ERL** (swing high atau swing low). **IRL (SIBI/BISI) tidak dipakai sebagai key level di LTF.** iFVG hanya muncul sebagai syarat tambahan di kondisi 2.
- Penamaan C2, C3, C4, dan seterusnya **tidak dipakai di LTF**. Penamaan itu hanya untuk HTF.
- Key level dievaluasi **per timeframe**, memakai candle dari timeframe LTF yang sedang dianalisis. Selalu sebutkan timeframe yang dipakai.
- Aturan key level berlaku penuh: invalid hanya karena **close body**, hanya key level sejenis yang **paling baru** yang dihitung, dan key level lama yang digantikan hanya invalid (bukan FTM/iFVG). Lihat `key-level-erl.md`.

# Pasangan Timeframe HTF > LTF

| HTF | LTF |
|---|---|
| Quarterly | Weekly |
| Monthly | Daily |
| Weekly | 4H |
| Daily | 1H |
| 4H | 15m |
| 1H | 5m |
| 15m | 1m |

---

# Istilah

| Istilah | Definisi |
|---|---|
| **LTF / HTF** | Low timeframe / high timeframe. Contoh: 15m adalah LTF dari 4H. |
| **Rangkaian candle bullish** | Rangkaian kenaikan harga (retrace naik) yang dihitung dari candle bullish pertama. |
| **Rangkaian candle bearish** | Rangkaian penurunan harga (retrace turun) yang dihitung dari candle bearish pertama. |
| **Opening / level CISD** | **Open candle pertama** dalam rangkaian. Level ini ditarik sebagai garis horizontal ke kanan. |
| **Body** | Badan candle. Yang dihitung adalah **close body**, bukan wick. |
| **FTM** | Swing high/low yang invalid karena close body di luar levelnya. |
| **iFVG** | SIBI/BISI yang invalid karena close body di luar batas zonanya. |

---

# BAGIAN A: BEARISH CISD (-CISD)

## A1. Syarat key level (ERL: swing high LTF)

Harus memenuhi **salah satu** dari tiga kondisi berikut:

| Kondisi | Swing high LTF | Syarat tambahan |
|---|---|---|
| **1** | **Valid** dan **terkena wick** (wick menyentuh levelnya, tidak ada close body di atasnya) | Tidak ada |
| **2** | **FTM** (ada close body di atas levelnya) | Ada **iFVG terbaru** di rangkaian candle bullish |
| **3** | **FTM** | **Tidak ada iFVG.** Dikonfirmasi lewat **SWL FTM** (swing low terbaru yang di-close body di bawahnya oleh candle bearish). **-CISD dan SWL FTM wajib terpenuhi keduanya.** |

Swing yang dipakai adalah swing **sejenis yang paling baru**. Swing yang lebih lama sudah invalid karena digantikan dan bukan FTM, jadi tidak bisa dipakai.

## A2. Syarat konfirmasi

1. Ada **rangkaian candle bullish** (kenaikan harga).
2. **Level CISD** = **open candle bullish pertama** dalam rangkaian itu.
3. **Body candle bearish menembus level CISD** (close body di bawah level). Wick yang menembus saja tidak dihitung.

Jika ketiganya terpenuhi bersama syarat key level di A1, maka **-CISD terkonfirmasi**.

## A3. Pseudocode (Bearish)

```text
swing_high = swing high LTF terbaru
level_cisd = open(candle bullish pertama dalam rangkaian naik)

# --- syarat key level ---
if swing_high.valid and swing_high.touched_by_wick:
    key_ok = True                                   # kondisi 1
elif swing_high.is_FTM and exists_latest_iFVG_in_bullish_run:
    key_ok = True                                   # kondisi 2
elif swing_high.is_FTM and latest_swing_low.is_FTM: # swing low di-close body di bawahnya
    key_ok = True                                   # kondisi 3 (wajib bersama -CISD)
else:
    key_ok = False

# --- konfirmasi ---
cisd_ok = candle_bearish.close < level_cisd         # body menembus open rangkaian bullish

if key_ok and cisd_ok:
    return "-CISD TERKONFIRMASI"
return "BELUM / TIDAK BERLAKU"
```

## A4. Contoh kasus (Bearish)

- **Kasus 1, swing high valid dan terkena:** Rangkaian candle bullish naik, wick menyentuh swing high LTF tanpa ada close body di atasnya. Lalu candle bearish besar close body di bawah open candle bullish pertama. **Hasil: -CISD terkonfirmasi (kondisi 1).**
- **Kasus 2, FTM dengan iFVG:** Swing high LTF sudah FTM. Di rangkaian candle bullish ada iFVG terbaru. Lalu body bearish menembus open candle bullish pertama. **Hasil: -CISD terkonfirmasi (kondisi 2).**
- **Kasus 3, FTM tanpa iFVG:** Swing high LTF sudah FTM dan tidak ada iFVG. Body bearish menembus open candle bullish pertama (-CISD), **dan** swing low terbaru di-close body di bawahnya (SWL FTM). **Hasil: -CISD terkonfirmasi (kondisi 3).** Jika SWL FTM belum terjadi, belum terkonfirmasi.
- **Kasus 4, hanya wick menembus:** Candle bearish hanya wick-nya yang melewati level CISD, close body tetap di atas. **Hasil: belum terkonfirmasi.**
- **Kasus 5, swing high lama:** Ada swing high yang sudah digantikan swing high yang lebih baru. **Hasil: swing high lama tidak bisa dipakai** (invalid, bukan FTM).

---

# BAGIAN B: BULLISH CISD (+CISD)

Kebalikan dari bearish CISD.

## B1. Syarat key level (ERL: swing low LTF)

Harus memenuhi **salah satu** dari tiga kondisi berikut:

| Kondisi | Swing low LTF | Syarat tambahan |
|---|---|---|
| **1** | **Valid** dan **terkena wick** (wick menyentuh levelnya, tidak ada close body di bawahnya) | Tidak ada |
| **2** | **FTM** (ada close body di bawah levelnya) | Ada **iFVG terbaru** di rangkaian candle bearish |
| **3** | **FTM** | **Tidak ada iFVG.** Dikonfirmasi lewat **SWH FTM** (swing high terbaru yang di-close body di atasnya oleh candle bullish). **+CISD dan SWH FTM wajib terpenuhi keduanya.** |

Swing yang dipakai adalah swing **sejenis yang paling baru**. Swing yang lebih lama sudah invalid karena digantikan dan bukan FTM, jadi tidak bisa dipakai.

## B2. Syarat konfirmasi

1. Ada **rangkaian candle bearish** (penurunan harga).
2. **Level CISD** = **open candle bearish pertama** dalam rangkaian itu.
3. **Body candle bullish menembus level CISD** (close body di atas level). Wick yang menembus saja tidak dihitung.

Jika ketiganya terpenuhi bersama syarat key level di B1, maka **+CISD terkonfirmasi**.

## B3. Pseudocode (Bullish)

```text
swing_low  = swing low LTF terbaru
level_cisd = open(candle bearish pertama dalam rangkaian turun)

# --- syarat key level ---
if swing_low.valid and swing_low.touched_by_wick:
    key_ok = True                                   # kondisi 1
elif swing_low.is_FTM and exists_latest_iFVG_in_bearish_run:
    key_ok = True                                   # kondisi 2
elif swing_low.is_FTM and latest_swing_high.is_FTM: # swing high di-close body di atasnya
    key_ok = True                                   # kondisi 3 (wajib bersama +CISD)
else:
    key_ok = False

# --- konfirmasi ---
cisd_ok = candle_bullish.close > level_cisd         # body menembus open rangkaian bearish

if key_ok and cisd_ok:
    return "+CISD TERKONFIRMASI"
return "BELUM / TIDAK BERLAKU"
```

## B4. Contoh kasus (Bullish)

- **Kasus 1, swing low valid dan terkena:** Rangkaian candle bearish turun, wick menyentuh swing low LTF tanpa ada close body di bawahnya. Lalu candle bullish besar close body di atas open candle bearish pertama. **Hasil: +CISD terkonfirmasi (kondisi 1).**
- **Kasus 2, FTM dengan iFVG:** Swing low LTF sudah FTM. Di rangkaian candle bearish ada iFVG terbaru. Lalu body bullish menembus open candle bearish pertama. **Hasil: +CISD terkonfirmasi (kondisi 2).**
- **Kasus 3, FTM tanpa iFVG:** Swing low LTF sudah FTM (candle hitam besar close body di bawahnya) dan tidak ada iFVG. Body bullish menembus open candle bearish pertama (+CISD), **dan** swing high terbaru di-close body di atasnya oleh candle bullish (SWH FTM). **Hasil: +CISD terkonfirmasi (kondisi 3).**
- **Kasus 4, hanya wick menembus:** Candle bullish hanya wick-nya yang melewati level CISD, close body tetap di bawah. **Hasil: belum terkonfirmasi.**
- **Kasus 5, swing low lama:** Ada swing low yang sudah digantikan swing low yang lebih baru. **Hasil: swing low lama tidak bisa dipakai** (invalid, bukan FTM).

---

# Perbandingan -CISD vs +CISD

| Komponen | Bearish CISD (-CISD) | Bullish CISD (+CISD) |
|---|---|---|
| Key level ERL LTF | Swing high | Swing low |
| Kondisi 1 | Swing high valid dan terkena wick | Swing low valid dan terkena wick |
| Kondisi 2 | Swing high FTM + iFVG terbaru di rangkaian bullish | Swing low FTM + iFVG terbaru di rangkaian bearish |
| Kondisi 3 | Swing high FTM tanpa iFVG, dikonfirmasi SWL FTM | Swing low FTM tanpa iFVG, dikonfirmasi SWH FTM |
| Rangkaian candle | Bullish | Bearish |
| Level CISD | Open candle bullish pertama | Open candle bearish pertama |
| Konfirmasi | Body bearish menembus level (close di bawah) | Body bullish menembus level (close di atas) |
| Arah setelahnya | Naik berubah ke turun | Turun berubah ke naik |

# Aturan Penting

- CISD hanya untuk **LTF**, dan key level yang dipakai **wajib ERL**.
- Konfirmasi memakai **close body**, bukan wick.
- Level CISD selalu **open candle pertama** dalam rangkaian.
- Pada kondisi 3, **dua syarat wajib bersama**: CISD dan swing berlawanan yang FTM.
- Hanya **swing sejenis yang paling baru** yang bisa dipakai. Swing lama yang digantikan bukan FTM.
- iFVG yang dipakai di kondisi 2 adalah iFVG **terbaru**.

# Panduan Output untuk AI

1. **Timeframe**: sebutkan timeframe LTF (dan HTF jika ada). Jika tidak terlihat di chart, tanyakan kepada pengguna.
2. **Arah**: bearish (-CISD) atau bullish (+CISD).
3. **Key level ERL LTF**: sebutkan swing yang dipakai (swing terbaru), levelnya, dan statusnya (valid dan terkena wick, atau FTM).
4. **Kondisi key level**: sebutkan kondisi 1, 2, atau 3, dan dasarnya (misalnya "swing high FTM, ada iFVG terbaru di rangkaian bullish").
5. **Rangkaian candle dan level CISD**: sebutkan candle pertama dalam rangkaian dan open-nya.
6. **Konfirmasi**: sebutkan apakah ada body yang menembus level CISD (close body), dan pada kondisi 3 juga status swing berlawanan yang FTM.
7. **Status akhir**: salah satu dari:
   - `-CISD / +CISD terkonfirmasi (kondisi X)`
   - `Belum terkonfirmasi` (sebutkan syarat yang belum terpenuhi)
   - `Tidak berlaku` (sebutkan alasannya)
8. Jika candle masih berjalan (belum close), nyatakan bahwa status belum final karena close body belum terbentuk.
9. Jika data di chart tidak terbaca jelas, sebutkan nilai yang diperkirakan dan beri tahu bahwa itu perkiraan.
