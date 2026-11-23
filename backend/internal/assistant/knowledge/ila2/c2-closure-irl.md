# C2 Closure pada Key Level IRL (SIBI / BISI)

Dokumen ini mendefinisikan logika **C2 Closure** untuk key level **IRL (Internal Range Liquidity)**:

- **Bearish FVG / SIBI** (Bagian A)
- **Bullish FVG / BISI** (Bagian B), yang merupakan **kebalikan (mirror)** dari bearish

Gunakan dokumen ini sebagai aturan baku saat mengevaluasi chart candlestick. Jangan menambah aturan di luar dokumen ini. Definisi pembentukan dan invalidasi SIBI/BISI ada di `key-level-irl.md`.

**Konteks:** Setiap timeframe memiliki key level IRL-nya sendiri, jadi SIBI/BISI diidentifikasi dan dievaluasi menggunakan candle dari timeframe yang sedang dianalisis. Selalu sebutkan timeframe yang dipakai saat mengevaluasi.

**Konsep kunci:** status C2 **tidak permanen**. C2 bisa **berpindah** ke candle berikutnya, atau **gugur**, selama FVG belum invalid. C2 yang berlaku adalah C2 **terakhir** yang masih aktif.

---

# Istilah Umum

| Istilah | Definisi |
|---|---|
| **FVG** | Fair Value Gap, gap harga dari 3 candle berurutan yang rentang harganya tidak bertemu karena candle 2 bergerak terlalu cepat. |
| **Zona FVG** | Area harga di dalam gap. |
| **Level acuan (reference)** | Level yang dipakai untuk menguji candle berikutnya. Nilai awalnya ditentukan saat FVG terbentuk, lalu terus diperbarui (lihat "Evaluasi Setiap Candle" di tiap bagian). |
| **Close** | Harga penutupan candle (body close). Wick tidak dihitung sebagai close. |
| **C2** | Candle yang memenuhi semua syarat C2 Closure (lihat "Syarat C2 Closure" di tiap bagian). |
| **C2 aktif** | C2 terakhir yang belum gugur. C2 aktif bisa berpindah ke candle lain atau gugur. |
| **iFVG** | FVG (SIBI/BISI) yang invalid karena close di luar batas zona. FVG yang invalid karena digantikan FVG sejenis baru bukan iFVG, hanya invalid dan diabaikan. Lihat `key-level-irl.md`. |

---

# BAGIAN A: BEARISH FVG / SIBI

## A1. Pembentukan FVG

1. Ambil 3 candle berurutan: candle 1, candle 2, candle 3.
2. Candle 2 bergerak turun cepat sehingga **low candle 1 > high candle 3** (tidak ada rentang harga yang bertemu).
3. Zona FVG:
   - Batas atas = **low candle 1**
   - Batas bawah = **high candle 3**
4. Level acuan awal = **high candle 3**.

```
        candle 1  candle 2  candle 3
low c1  ---------------------------  <- batas atas FVG (invalid bila close di atasnya)
          (zona FVG / area pink)
high c3 ---------------------------  <- batas bawah FVG = level acuan awal
```

## A2. Syarat C2 Closure

Sebuah candle disebut **C2** jika **semua** kondisi berikut terpenuhi:

1. FVG **belum invalid** (belum ada close di atas batas atas FVG, dan belum terbentuk SIBI baru).
2. Saat candle berjalan, **harga menyentuh atau melewati level acuan** (cukup lewat wick).
3. Harga **masuk ke area FVG** (cukup lewat wick, body tidak harus masuk).
4. **Close berada di bawah level acuan.** Close yang tepat sama dengan level acuan **bukan C2**.

Intinya: harga menusuk naik ke dalam FVG, ditolak, lalu close kembali di bawah level acuan.

Hal yang diperbolehkan:
- Wick boleh masuk jauh ke dalam zona FVG.
- Wick boleh menembus **melewati batas atas FVG**, selama close tetap di bawah level acuan.
- C2 tidak harus candle yang langsung setelah candle 3. C2 bisa muncul beberapa candle kemudian, selama FVG belum invalid.
- Body candle boleh seluruhnya berada di bawah zona FVG, selama wick-nya memenuhi syarat 2 dan 3.

## A3. Evaluasi Setiap Candle

Evaluasi candle satu per satu, berurutan setelah candle 3, **dengan urutan pengecekan berikut**:

| Urutan | Kondisi candle | Hasil |
|---|---|---|
| 1 | **Terbentuk SIBI baru** (FVG sejenis) setelah FVG ini, dengan candle ini sebagai candle 3 FVG baru | **FVG INVALID.** Hentikan evaluasi. C2 aktif (jika ada) gugur. Tidak ada C2 lagi untuk FVG ini. |
| 2 | **Close di atas batas atas FVG** (low candle 1) | **FVG INVALID.** Hentikan evaluasi. C2 aktif (jika ada) gugur. Tidak ada C2 lagi untuk FVG ini. |
| 3 | **High candle tidak menyentuh level acuan** | **Diabaikan.** Level acuan tidak berubah. C2 aktif (jika ada) tetap. |
| 4 | High menyentuh/melewati level acuan **dan** close di bawah level acuan | **C2 CLOSURE.** Candle ini menjadi C2 aktif. Jika sebelumnya sudah ada C2 aktif, status C2 **berpindah** ke candle ini. **Level acuan baru = high candle ini.** |
| 5 | High menyentuh/melewati level acuan **dan** close di atas level acuan, tetapi masih di dalam FVG (tidak di atas batas atas) | **Bukan C2.** C2 aktif (jika ada) **gugur**. **Level acuan bergeser ke high candle ini.** FVG tetap hidup. |

### Pseudocode (Bearish)

```text
reference = high(candle3)
upper     = low(candle1)
c2        = None

for candle in candles_after(candle3):
    if new_sibi_formed(candle):       # candle ini = candle 3 dari SIBI baru
        c2 = None
        return "FVG INVALID (digantikan SIBI baru)"

    if candle.close > upper:
        c2 = None
        return "FVG INVALID"

    if candle.high < reference:
        continue                      # tidak menyentuh acuan, diabaikan (c2 tetap)

    if candle.close < reference:
        c2 = candle                   # C2 baru / C2 berpindah ke candle ini
        reference = candle.high       # acuan berikutnya = high C2
        continue

    # menyentuh acuan, close di atas acuan tapi masih di dalam FVG
    c2 = None                         # C2 aktif gugur
    reference = candle.high           # acuan bergeser
    continue

if c2 is not None:
    return "C2 CLOSURE aktif di candle " + c2
return "BELUM ADA C2 (FVG masih hidup)"
```

## A4. Contoh Kasus (Bearish)

Zona FVG dengan batas atas = `U` dan batas bawah = `L` (level acuan awal).

- **Kasus A, C2 langsung:** Candle setelah candle 3 punya wick naik melewati `L` dan masuk zona FVG, tetapi close di bawah `L`. **Hasil: C2 Closure.**
- **Kasus B, candle tidak menyentuh acuan lalu C2:** Candle X naik tetapi high-nya di bawah `L` (diabaikan, FVG tetap hidup). Candle Y wick naik melewati `L` dan masuk zona, close di bawah `L`. **Hasil: Candle Y adalah C2 Closure.**
- **Kasus C, acuan bergeser:** Candle X menyentuh `L` dan close di atas `L` tetapi masih di bawah `U` (bukan C2, acuan baru = high candle X). Candle Y menyentuh atau melewati high candle X, tetapi close di bawah high candle X. **Hasil: Candle Y adalah C2 Closure.**
- **Kasus D, invalidasi langsung:** Candle naik dan close di atas `U`. **Hasil: FVG invalid, tidak ada C2.**
- **Kasus E, invalidasi lalu candle turun kembali:** Candle X close di atas `U` (FVG invalid). Candle berikutnya turun kembali masuk zona FVG dan close di bawah acuan. **Hasil: Bukan C2 Closure**, karena FVG sudah invalid sebelum candle tersebut terbentuk.
- **Kasus F, wick menembus batas atas FVG:** Candle wick-nya naik melewati `U`, tetapi close di bawah level acuan. **Hasil: Tetap C2 Closure.** Invalidasi hanya dihitung dari close, bukan wick.
- **Kasus G, C2 berpindah:** Candle X adalah C2 (wick melewati acuan, close di bawahnya). Candle Y wick-nya melewati high candle X, tetapi close di bawah high candle X. **Hasil: Status C2 berpindah dari candle X ke candle Y.**
- **Kasus H, C2 gugur lalu C2 baru:**
  1. Candle X adalah C2 (acuan berikutnya = high candle X).
  2. Candle Y wick-nya melewati high candle X, tetapi close **di atas** high candle X dan masih di bawah `U`. Candle Y bukan C2, **C2 candle X gugur**, acuan bergeser ke high candle Y. FVG tetap hidup.
  3. Candle Z wick-nya melewati high candle Y, tetapi close di bawah high candle Y.
  **Hasil: Candle Z adalah C2 Closure aktif.**
  (Wick candle Y dan candle Z boleh melewati `U`, selama close tidak di atas `U`.)

- **Kasus I, digantikan SIBI baru:** Setelah SIBI ini, terbentuk SIBI baru (bisa berbagi candle dengan SIBI lama, atau terpisah beberapa candle). **Hasil: SIBI lama langsung invalid sejak SIBI baru terbentuk**, tanpa menunggu close di atas `U`. C2 aktif pada SIBI lama (jika ada) gugur. Yang dievaluasi selanjutnya adalah SIBI baru.
---

# BAGIAN B: BULLISH FVG / BISI

Bullish FVG / BISI adalah **kebalikan dari bearish FVG / SIBI**. Semua arah dibalik: high menjadi low, atas menjadi bawah, naik menjadi turun.

## B1. Pembentukan FVG

1. Ambil 3 candle berurutan: candle 1, candle 2, candle 3.
2. Candle 2 bergerak naik cepat sehingga **high candle 1 < low candle 3** (tidak ada rentang harga yang bertemu).
3. Zona FVG:
   - Batas bawah = **high candle 1**
   - Batas atas = **low candle 3**
4. Level acuan awal = **low candle 3**.

```
low c3  ---------------------------  <- batas atas FVG = level acuan awal
          (zona FVG)
high c1 ---------------------------  <- batas bawah FVG (invalid bila close di bawahnya)
        candle 1  candle 2  candle 3
```

## B2. Syarat C2 Closure

Sebuah candle disebut **C2** jika **semua** kondisi berikut terpenuhi:

1. FVG **belum invalid** (belum ada close di bawah batas bawah FVG, dan belum terbentuk BISI baru).
2. Saat candle berjalan, **harga menyentuh atau melewati level acuan** (cukup lewat wick).
3. Harga **masuk ke area FVG** (cukup lewat wick, body tidak harus masuk).
4. **Close berada di atas level acuan.** Close yang tepat sama dengan level acuan **bukan C2**.

Intinya: harga menusuk turun ke dalam FVG, ditolak, lalu close kembali di atas level acuan.

Hal yang diperbolehkan:
- Wick boleh masuk jauh ke dalam zona FVG.
- Wick boleh menembus **melewati batas bawah FVG**, selama close tetap di atas level acuan.
- C2 tidak harus candle yang langsung setelah candle 3. C2 bisa muncul beberapa candle kemudian, selama FVG belum invalid.
- Body candle boleh seluruhnya berada di atas zona FVG, selama wick-nya memenuhi syarat 2 dan 3.

## B3. Evaluasi Setiap Candle

Evaluasi candle satu per satu, berurutan setelah candle 3, **dengan urutan pengecekan berikut**:

| Urutan | Kondisi candle | Hasil |
|---|---|---|
| 1 | **Terbentuk BISI baru** (FVG sejenis) setelah FVG ini, dengan candle ini sebagai candle 3 FVG baru | **FVG INVALID.** Hentikan evaluasi. C2 aktif (jika ada) gugur. Tidak ada C2 lagi untuk FVG ini. |
| 2 | **Close di bawah batas bawah FVG** (high candle 1) | **FVG INVALID.** Hentikan evaluasi. C2 aktif (jika ada) gugur. Tidak ada C2 lagi untuk FVG ini. |
| 3 | **Low candle tidak menyentuh level acuan** | **Diabaikan.** Level acuan tidak berubah. C2 aktif (jika ada) tetap. |
| 4 | Low menyentuh/melewati level acuan **dan** close di atas level acuan | **C2 CLOSURE.** Candle ini menjadi C2 aktif. Jika sebelumnya sudah ada C2 aktif, status C2 **berpindah** ke candle ini. **Level acuan baru = low candle ini.** |
| 5 | Low menyentuh/melewati level acuan **dan** close di bawah level acuan, tetapi masih di dalam FVG (tidak di bawah batas bawah) | **Bukan C2.** C2 aktif (jika ada) **gugur**. **Level acuan bergeser ke low candle ini.** FVG tetap hidup. |

### Pseudocode (Bullish)

```text
reference = low(candle3)
lower     = high(candle1)
c2        = None

for candle in candles_after(candle3):
    if new_bisi_formed(candle):       # candle ini = candle 3 dari BISI baru
        c2 = None
        return "FVG INVALID (digantikan BISI baru)"

    if candle.close < lower:
        c2 = None
        return "FVG INVALID"

    if candle.low > reference:
        continue                      # tidak menyentuh acuan, diabaikan (c2 tetap)

    if candle.close > reference:
        c2 = candle                   # C2 baru / C2 berpindah ke candle ini
        reference = candle.low        # acuan berikutnya = low C2
        continue

    # menyentuh acuan, close di bawah acuan tapi masih di dalam FVG
    c2 = None                         # C2 aktif gugur
    reference = candle.low            # acuan bergeser
    continue

if c2 is not None:
    return "C2 CLOSURE aktif di candle " + c2
return "BELUM ADA C2 (FVG masih hidup)"
```

## B4. Contoh Kasus (Bullish)

Zona FVG dengan batas bawah = `D` dan batas atas = `H` (level acuan awal).

- **Kasus A, C2 langsung:** Candle setelah candle 3 punya wick turun melewati `H` dan masuk zona FVG, tetapi close di atas `H`. **Hasil: C2 Closure.**
- **Kasus B, candle tidak menyentuh acuan lalu C2:** Candle X turun tetapi low-nya di atas `H` (diabaikan, FVG tetap hidup). Candle Y wick turun melewati `H` dan masuk zona, close di atas `H`. **Hasil: Candle Y adalah C2 Closure.**
- **Kasus C, acuan bergeser:** Candle X menyentuh `H` dan close di bawah `H` tetapi masih di atas `D` (bukan C2, acuan baru = low candle X). Candle Y menyentuh atau melewati low candle X, tetapi close di atas low candle X. **Hasil: Candle Y adalah C2 Closure.**
- **Kasus D, invalidasi langsung:** Candle turun dan close di bawah `D`. **Hasil: FVG invalid, tidak ada C2.**
- **Kasus E, invalidasi lalu candle naik kembali:** Candle X close di bawah `D` (FVG invalid). Candle berikutnya naik kembali masuk zona FVG dan close di atas acuan. **Hasil: Bukan C2 Closure**, karena FVG sudah invalid sebelum candle tersebut terbentuk.
- **Kasus F, wick menembus batas bawah FVG:** Candle wick-nya turun melewati `D`, tetapi close di atas level acuan. **Hasil: Tetap C2 Closure.** Invalidasi hanya dihitung dari close, bukan wick.
- **Kasus G, C2 berpindah:** Candle X adalah C2 (wick melewati acuan, close di atasnya). Candle Y wick-nya melewati low candle X, tetapi close di atas low candle X. **Hasil: Status C2 berpindah dari candle X ke candle Y.**
- **Kasus H, C2 gugur lalu C2 baru:**
  1. Candle X adalah C2 (acuan berikutnya = low candle X).
  2. Candle Y wick-nya melewati low candle X, tetapi close **di bawah** low candle X dan masih di atas `D`. Candle Y bukan C2, **C2 candle X gugur**, acuan bergeser ke low candle Y. FVG tetap hidup.
  3. Candle Z wick-nya melewati low candle Y, tetapi close di atas low candle Y.
  **Hasil: Candle Z adalah C2 Closure aktif.**
  (Wick candle Y dan candle Z boleh melewati `D`, selama close tidak di bawah `D`.)

- **Kasus I, digantikan BISI baru:** Setelah BISI ini, terbentuk BISI baru (bisa berbagi candle dengan BISI lama, atau terpisah beberapa candle). **Hasil: BISI lama langsung invalid sejak BISI baru terbentuk**, tanpa menunggu close di bawah `D`. C2 aktif pada BISI lama (jika ada) gugur. Yang dievaluasi selanjutnya adalah BISI baru.
---

# Perbandingan Bearish vs Bullish

| Komponen | Bearish FVG / SIBI | Bullish FVG / BISI |
|---|---|---|
| Pembentukan | Low candle 1 > high candle 3 | High candle 1 < low candle 3 |
| Batas FVG untuk invalidasi | Batas atas = low candle 1 | Batas bawah = high candle 1 |
| Level acuan awal | High candle 3 | Low candle 3 |
| Wick harus | Naik menyentuh/melewati acuan | Turun menyentuh/melewati acuan |
| Close C2 | **Di bawah** acuan | **Di atas** acuan |
| Acuan setelah ada C2 | High candle C2 | Low candle C2 |
| C2 berpindah / C2 baru | Menyentuh acuan, close di bawahnya | Menyentuh acuan, close di atasnya |
| C2 gugur | Menyentuh acuan, close di atasnya (masih di dalam FVG) | Menyentuh acuan, close di bawahnya (masih di dalam FVG) |
| FVG invalid | Close **di atas** batas atas | Close **di bawah** batas bawah |
| FVG invalid (kondisi lain) | Terbentuk **SIBI baru** | Terbentuk **BISI baru** |

---

# Aturan Penting (Berlaku untuk Keduanya)

- Invalidasi **hanya** berdasarkan **close**, bukan wick.
- Syarat C2 menggunakan **wick** untuk menyentuh/masuk FVG, dan **close** untuk penolakan.
- Status C2 **bisa berpindah** ke candle berikutnya yang menyentuh acuan (high/low C2) dan close di sisi penolakan.
- Status C2 **gugur** bila candle berikutnya menyentuh acuan tetapi close di sisi yang salah (masih di dalam FVG). Acuan lalu bergeser ke candle tersebut.
- Candle yang tidak menyentuh level acuan **diabaikan**, tidak mengubah apa pun (C2 aktif tetap).
- Evaluasi selalu dimulai dari candle setelah candle 3, satu per satu secara berurutan.

---
- FVG juga **langsung invalid** saat terbentuk FVG sejenis yang baru (SIBI baru untuk SIBI, BISI baru untuk BISI) di timeframe yang sama, tanpa menunggu close di luar zona. Berlaku untuk FVG sejenis mana pun yang terbentuk setelahnya, baik berurutan maupun terpisah. Saat itu C2 aktif (jika ada) gugur.
- Invalid bersifat **permanen**. FVG lama tidak hidup kembali, meskipun FVG baru yang menggantikannya kemudian invalid. Jika ada beberapa FVG sejenis, hanya yang paling baru yang bisa valid.

# Panduan Output untuk AI

Saat diminta mengevaluasi chart, jawab dengan format berikut:

1. **Timeframe dan jenis FVG**: sebutkan timeframe chart, lalu jenis FVG: bearish (SIBI) atau bullish (BISI). Jika timeframe tidak terlihat di chart, tanyakan kepada pengguna.
2. **Identifikasi FVG**: sebutkan candle 1, 2, 3, batas atas, dan batas bawah.
3. **Level acuan awal**: sebutkan nilainya.
4. **Evaluasi per candle**: untuk tiap candle setelah candle 3, sebutkan kondisi (menyentuh acuan atau tidak, posisi close) dan hasilnya (diabaikan / C2 terbentuk atau berpindah / C2 gugur dan acuan bergeser / invalid), serta level acuan setelah candle tersebut.
5. **Status akhir**: salah satu dari:
   - `C2 CLOSURE aktif` (sebutkan candle C2 aktif dan level acuan berikutnya)
   - `FVG INVALID` (sebutkan candle yang membuatnya invalid, atau FVG sejenis baru yang menggantikannya)
   - `FVG masih hidup, belum ada C2` (sebutkan level acuan saat ini dan batas invalidasi)
6. Jika candle masih berjalan (belum close), nyatakan bahwa status belum final karena close belum terbentuk. Karena C2 bisa berpindah atau gugur, status C2 aktif juga bersifat sementara sampai candle berikutnya dievaluasi.
7. Jika data di chart tidak terbaca jelas, sebutkan nilai yang diperkirakan dan beri tahu bahwa itu perkiraan.
