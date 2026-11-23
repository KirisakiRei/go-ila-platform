# C2 Closure pada Key Level ERL (Swing High / Swing Low)

Dokumen ini mendefinisikan logika **C2 Closure** untuk key level **ERL (External Range Liquidity)**:

- **Swing Low** (Bagian A), sisi **bullish**
- **Swing High** (Bagian B), yang merupakan **kebalikan (mirror)** dari swing low (sisi **bearish**)

Gunakan dokumen ini sebagai aturan baku saat mengevaluasi chart candlestick. Jangan menambah aturan di luar dokumen ini. Definisi pembentukan dan invalidasi swing high/low ada di `key-level-erl.md`.

**Konteks:** Setiap timeframe memiliki key level ERL-nya sendiri, jadi swing high/low diidentifikasi dan dievaluasi menggunakan candle dari timeframe yang sedang dianalisis. Selalu sebutkan timeframe yang dipakai saat mengevaluasi.

**Gambaran C2 closure pada ERL:** harga menyapu level swing dengan wick, ditolak, lalu close kembali di sisi semula.

**Konsep kunci:** status C2 **tidak permanen**. C2 bisa **berpindah** ke candle berikutnya, atau **gugur**, selama swing belum invalid. C2 yang berlaku adalah C2 **terakhir** yang masih aktif.

---

# Istilah Umum

| Istilah | Definisi |
|---|---|
| **Level swing** | Low candle tengah (swing low) atau high candle tengah (swing high) pembentuk swing. Level ini juga batas invalidasi swing. |
| **Level acuan (reference)** | Level yang dipakai untuk menguji candle berikutnya. Awalnya sama dengan level swing. Setelah ada C2, level acuan menjadi low C2 (swing low) atau high C2 (swing high). |
| **Menyapu level** | Wick candle menyentuh atau melewati level acuan. |
| **Close** | Harga penutupan candle (body close). Wick tidak dihitung sebagai close. |
| **C2** | Candle yang memenuhi semua syarat C2 Closure pada swing. |
| **C2 aktif** | C2 terakhir yang belum gugur. C2 aktif bisa berpindah ke candle lain atau gugur. |
| **FTM** (Failure to Manipulate) | Swing high/low yang invalid karena close di luar level. Lihat `key-level-erl.md`. |

Candle yang dievaluasi adalah candle **setelah candle kanan** pembentuk swing. Candle tengah dan candle kanan pembentuk swing tidak bisa menjadi C2.

---

# BAGIAN A: SWING LOW (Bullish)

## A1. Syarat C2 Closure

Sebuah candle disebut **C2** jika **semua** kondisi berikut terpenuhi:

1. Swing low **belum invalid** (belum ada close di bawah level swing low, dan belum terbentuk swing low baru).
2. Saat candle berjalan, **wick menyentuh atau melewati level acuan** (menyapu level).
3. **Close berada di atas level acuan.**

Intinya: harga menyapu level dengan wick, ditolak, lalu close kembali di atas.

Hal yang diperbolehkan:
- Wick boleh menembus jauh di bawah level swing low, selama close tidak di bawah level swing low.
- C2 tidak harus candle yang langsung setelah swing terbentuk. C2 bisa muncul beberapa candle kemudian, selama swing low belum invalid.

## A2. Evaluasi Setiap Candle

Evaluasi candle satu per satu, berurutan setelah candle kanan pembentuk swing, **dengan urutan pengecekan berikut**:

| Urutan | Kondisi candle | Hasil |
|---|---|---|
| 1 | **Terbentuk swing low baru** (swing sejenis) setelah swing ini, dengan candle ini sebagai candle kanan swing baru | **SWING LOW INVALID.** Hentikan evaluasi. C2 aktif (jika ada) gugur. Tidak ada C2 lagi untuk swing ini. |
| 2 | **Close di bawah level swing low** | **SWING LOW INVALID.** Hentikan evaluasi. C2 aktif (jika ada) gugur. Tidak ada C2 lagi untuk swing ini. |
| 3 | **Low candle tidak menyentuh level acuan** | **Diabaikan.** Level acuan tidak berubah. C2 aktif (jika ada) tetap. |
| 4 | Low menyentuh/melewati level acuan **dan** close di atas level acuan | **C2 CLOSURE.** Candle ini menjadi C2 aktif. Jika sebelumnya sudah ada C2 aktif, status C2 **berpindah** ke candle ini. **Level acuan baru = low candle ini.** |

**Catatan:** level acuan setelah ada C2 berada di level swing low atau di bawahnya. Jadi candle yang close di bawah level acuan juga close di bawah level swing low, dan itu masuk ke urutan 2 (swing invalid dan C2 aktif gugur).

### Pseudocode (Swing Low)

```text
swing_low = level swing low
reference = swing_low
c2        = None

for candle in candles_after(right_candle):
    if new_swing_low_formed(candle):  # candle ini = candle kanan dari swing low baru
        c2 = None
        return "SWING LOW INVALID (digantikan swing low baru)"

    if candle.close < swing_low:
        c2 = None
        return "SWING LOW INVALID"

    if candle.low > reference:
        continue                      # tidak menyentuh acuan, diabaikan (c2 tetap)

    if candle.close > reference:
        c2 = candle                   # C2 baru / C2 berpindah ke candle ini
        reference = candle.low        # acuan berikutnya = low C2
        continue

if c2 is not None:
    return "C2 CLOSURE aktif di candle " + c2
return "BELUM ADA C2 (swing low masih valid)"
```

## A3. Contoh Kasus (Swing Low)

Swing low dengan level `S`.

- **Kasus A, C2 langsung:** Candle setelah swing wick-nya turun melewati `S`, tetapi close di atas `S`. **Hasil: C2 Closure.**
- **Kasus B, candle tidak menyentuh level lalu C2:** Beberapa candle bergerak tanpa low menyentuh `S` (diabaikan). Candle berikutnya wick-nya turun melewati `S`, close di atas `S`. **Hasil: Candle tersebut adalah C2 Closure.** (Seperti pada contoh gambar: C2 muncul beberapa candle setelah swing low terbentuk.)
- **Kasus C, invalidasi:** Candle turun dan close di bawah `S`. **Hasil: Swing low invalid, tidak ada C2.**
- **Kasus D, invalidasi lalu candle naik kembali:** Candle X close di bawah `S` (swing low invalid). Candle berikutnya naik kembali dan close di atas `S`. **Hasil: Bukan C2 Closure**, karena swing low sudah invalid sebelum candle tersebut terbentuk.
- **Kasus E, C2 berpindah:** Candle X adalah C2 (wick melewati `S`, close di atas `S`). Acuan berikutnya = low candle X. Candle Y wick-nya melewati low candle X, tetapi close di atas low candle X (dan tidak di bawah `S`). **Hasil: Status C2 berpindah dari candle X ke candle Y.**
- **Kasus F, C2 tetap karena candle berikutnya tidak menyentuh acuan:** Candle X adalah C2. Candle Y dan Z low-nya tidak menyentuh low candle X. **Hasil: C2 aktif tetap di candle X.**
- **Kasus G, C2 gugur:** Candle X adalah C2. Candle Y close di bawah `S`. **Hasil: C2 candle X gugur dan swing low invalid.**

- **Kasus H, digantikan swing low baru:** Setelah swing low ini, terbentuk swing low baru. **Hasil: swing low lama langsung invalid sejak swing low baru terbentuk**, tanpa menunggu close di bawah `S`. C2 aktif pada swing lama (jika ada) gugur. Ini bukan FTM, hanya invalid dan diabaikan. Yang dievaluasi selanjutnya adalah swing low baru.
---

# BAGIAN B: SWING HIGH (Bearish)

Kebalikan dari swing low.

## B1. Syarat C2 Closure

Sebuah candle disebut **C2** jika **semua** kondisi berikut terpenuhi:

1. Swing high **belum invalid** (belum ada close di atas level swing high, dan belum terbentuk swing high baru).
2. Saat candle berjalan, **wick menyentuh atau melewati level acuan** (menyapu level).
3. **Close berada di bawah level acuan.**

Intinya: harga menyapu level dengan wick, ditolak, lalu close kembali di bawah.

Hal yang diperbolehkan:
- Wick boleh menembus jauh di atas level swing high, selama close tidak di atas level swing high.
- C2 tidak harus candle yang langsung setelah swing terbentuk. C2 bisa muncul beberapa candle kemudian, selama swing high belum invalid.

## B2. Evaluasi Setiap Candle

| Urutan | Kondisi candle | Hasil |
|---|---|---|
| 1 | **Terbentuk swing high baru** (swing sejenis) setelah swing ini, dengan candle ini sebagai candle kanan swing baru | **SWING HIGH INVALID.** Hentikan evaluasi. C2 aktif (jika ada) gugur. Tidak ada C2 lagi untuk swing ini. |
| 2 | **Close di atas level swing high** | **SWING HIGH INVALID.** Hentikan evaluasi. C2 aktif (jika ada) gugur. Tidak ada C2 lagi untuk swing ini. |
| 3 | **High candle tidak menyentuh level acuan** | **Diabaikan.** Level acuan tidak berubah. C2 aktif (jika ada) tetap. |
| 4 | High menyentuh/melewati level acuan **dan** close di bawah level acuan | **C2 CLOSURE.** Candle ini menjadi C2 aktif. Jika sebelumnya sudah ada C2 aktif, status C2 **berpindah** ke candle ini. **Level acuan baru = high candle ini.** |

**Catatan:** level acuan setelah ada C2 berada di level swing high atau di atasnya. Jadi candle yang close di atas level acuan juga close di atas level swing high, dan itu masuk ke urutan 2 (swing invalid dan C2 aktif gugur).

### Pseudocode (Swing High)

```text
swing_high = level swing high
reference  = swing_high
c2         = None

for candle in candles_after(right_candle):
    if new_swing_high_formed(candle):  # candle ini = candle kanan dari swing high baru
        c2 = None
        return "SWING HIGH INVALID (digantikan swing high baru)"

    if candle.close > swing_high:
        c2 = None
        return "SWING HIGH INVALID"

    if candle.high < reference:
        continue                      # tidak menyentuh acuan, diabaikan (c2 tetap)

    if candle.close < reference:
        c2 = candle                   # C2 baru / C2 berpindah ke candle ini
        reference = candle.high       # acuan berikutnya = high C2
        continue

if c2 is not None:
    return "C2 CLOSURE aktif di candle " + c2
return "BELUM ADA C2 (swing high masih valid)"
```

## B3. Contoh Kasus (Swing High)

Swing high dengan level `S`.

- **Kasus A, C2 langsung:** Candle setelah swing wick-nya naik melewati `S`, tetapi close di bawah `S`. **Hasil: C2 Closure.**
- **Kasus B, candle tidak menyentuh level lalu C2:** Beberapa candle bergerak tanpa high menyentuh `S` (diabaikan). Candle berikutnya wick-nya naik melewati `S`, close di bawah `S`. **Hasil: Candle tersebut adalah C2 Closure.**
- **Kasus C, invalidasi:** Candle naik dan close di atas `S`. **Hasil: Swing high invalid, tidak ada C2.**
- **Kasus D, invalidasi lalu candle turun kembali:** Candle X close di atas `S` (swing high invalid). Candle berikutnya turun kembali dan close di bawah `S`. **Hasil: Bukan C2 Closure**, karena swing high sudah invalid sebelum candle tersebut terbentuk.
- **Kasus E, C2 berpindah:** Candle X adalah C2 (wick melewati `S`, close di bawah `S`). Acuan berikutnya = high candle X. Candle Y wick-nya melewati high candle X, tetapi close di bawah high candle X (dan tidak di atas `S`). **Hasil: Status C2 berpindah dari candle X ke candle Y.**
- **Kasus F, C2 tetap karena candle berikutnya tidak menyentuh acuan:** Candle X adalah C2. Candle Y dan Z high-nya tidak menyentuh high candle X. **Hasil: C2 aktif tetap di candle X.**
- **Kasus G, C2 gugur:** Candle X adalah C2. Candle Y close di atas `S`. **Hasil: C2 candle X gugur dan swing high invalid.**

- **Kasus H, digantikan swing high baru:** Setelah swing high ini, terbentuk swing high baru (misalnya swing high di candle 7, lalu swing high di candle 9). **Hasil: swing high lama langsung invalid sejak swing high baru terbentuk**, tanpa menunggu close di atas `S`. C2 aktif pada swing lama (jika ada) gugur. Ini bukan FTM, hanya invalid dan diabaikan. Yang dievaluasi selanjutnya adalah swing high baru.
---

# Perbandingan Swing Low vs Swing High

| Komponen | Swing Low (bullish) | Swing High (bearish) |
|---|---|---|
| Level swing | Low candle tengah | High candle tengah |
| Level acuan awal | Level swing low | Level swing high |
| Wick harus | Turun menyentuh/melewati acuan | Naik menyentuh/melewati acuan |
| Close C2 | **Di atas** acuan | **Di bawah** acuan |
| Acuan setelah ada C2 | Low candle C2 | High candle C2 |
| C2 berpindah | Menyentuh acuan, close di atasnya | Menyentuh acuan, close di bawahnya |
| Swing invalid (C2 aktif gugur) | Close **di bawah** level swing low | Close **di atas** level swing high |
| Invalid juga jika (C2 aktif gugur) | Terbentuk **swing low baru** | Terbentuk **swing high baru** |

# Aturan Penting

- Invalidasi **hanya** berdasarkan **close**, bukan wick.
- C2 menggunakan **wick** untuk menyapu level acuan, dan **close** untuk penolakan.
- Status C2 **bisa berpindah** ke candle berikutnya yang menyentuh acuan (low/high C2) dan close di sisi penolakan.
- Status C2 **gugur** saat swing invalid (close di luar level swing).
- Candle yang tidak menyentuh level acuan **diabaikan**, tidak mengubah apa pun (C2 aktif tetap).
- Swing juga **langsung invalid** saat terbentuk swing sejenis yang baru (swing high baru untuk swing high, swing low baru untuk swing low) di timeframe yang sama, tanpa menunggu close di luar level. Saat itu C2 aktif (jika ada) gugur. Invalid bersifat **permanen**, dan hanya swing sejenis yang paling baru yang bisa valid.
- Evaluasi dimulai dari candle setelah candle kanan pembentuk swing, satu per satu secara berurutan.

# Panduan Output untuk AI

1. **Timeframe dan jenis swing**: sebutkan timeframe chart, lalu swing high atau swing low. Jika timeframe tidak terlihat di chart, tanyakan kepada pengguna.
2. **Identifikasi swing**: sebutkan candle tengahnya dan levelnya.
3. **Evaluasi per candle**: untuk tiap candle setelah candle kanan, sebutkan kondisi (menyentuh acuan atau tidak, posisi close) dan hasilnya (diabaikan / C2 terbentuk atau berpindah / invalid), serta level acuan setelah candle tersebut.
4. **Status akhir**: salah satu dari:
   - `C2 CLOSURE aktif` (sebutkan candle C2 aktif dan level acuan berikutnya)
   - `SWING INVALID` (sebutkan candle yang membuatnya invalid, atau swing sejenis baru yang menggantikannya)
   - `Swing masih valid, belum ada C2` (sebutkan level swing dan batas invalidasi)
5. Jika candle masih berjalan (belum close), nyatakan bahwa status belum final karena close belum terbentuk. Karena C2 bisa berpindah atau gugur, status C2 aktif juga bersifat sementara sampai candle berikutnya dievaluasi.
6. Jika data di chart tidak terbaca jelas, sebutkan nilai yang diperkirakan dan beri tahu bahwa itu perkiraan.
