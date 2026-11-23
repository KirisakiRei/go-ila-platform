# Key Level IRL (SIBI / BISI)

Dokumen ini mendefinisikan key level **IRL (Internal Range Liquidity)**, yaitu **SIBI** (Bearish FVG) dan **BISI** (Bullish FVG). Gunakan dokumen ini sebagai aturan baku saat mengevaluasi chart candlestick. Jangan menambah aturan di luar dokumen ini.

**Konteks:** setiap timeframe memiliki key level IRL-nya sendiri. SIBI/BISI diidentifikasi menggunakan candle dari timeframe yang sedang dianalisis. Selalu sebutkan timeframe yang dipakai saat mengevaluasi.

---

SIBI dan BISI adalah **FVG (Fair Value Gap)**, yaitu gap harga dari 3 candle berurutan yang rentang harganya tidak bertemu karena candle 2 bergerak terlalu cepat. Area gap disebut **zona FVG**.

| Nama | Jenis FVG | Arah candle 2 |
|---|---|---|
| **SIBI** | Bearish FVG | Turun cepat |
| **BISI** | Bullish FVG | Naik cepat |

## 1. Bearish FVG / SIBI

1. Ambil 3 candle berurutan: candle 1, candle 2, candle 3.
2. Candle 2 bergerak turun cepat sehingga **low candle 1 tidak bertemu high candle 3** (low candle 1 lebih tinggi dari high candle 3).
3. Zona FVG:
   - Batas atas = **low candle 1**
   - Batas bawah = **high candle 3**

```
        candle 1  candle 2  candle 3
low c1  ---------------------------  <- batas atas zona FVG
          (zona FVG)
high c3 ---------------------------  <- batas bawah zona FVG
```

## 2. Bullish FVG / BISI

Kebalikan dari bearish FVG / SIBI.

1. Ambil 3 candle berurutan: candle 1, candle 2, candle 3.
2. Candle 2 bergerak naik cepat sehingga **high candle 1 tidak bertemu low candle 3** (high candle 1 lebih rendah dari low candle 3).
3. Zona FVG:
   - Batas bawah = **high candle 1**
   - Batas atas = **low candle 3**

```
low c3  ---------------------------  <- batas atas zona FVG
          (zona FVG)
high c1 ---------------------------  <- batas bawah zona FVG
        candle 1  candle 2  candle 3
```

## 3. Invalidasi SIBI dan BISI

SIBI/BISI invalid jika **salah satu** dari dua kondisi berikut terpenuhi.

### 3.1 Close di luar batas zona

Invalidasi ini hanya dihitung dari **close**, bukan wick. Wick boleh menembus batas zona, tetapi close tidak boleh menembus.

| Key level | Invalid jika | Tidak invalid jika |
|---|---|---|
| **SIBI** (Bearish FVG) | **Close di atas batas atas zona** (low candle 1) | Wick menembus ke atas batas atas zona, tetapi close tidak di atasnya |
| **BISI** (Bullish FVG) | **Close di bawah batas bawah zona** (high candle 1) | Wick menembus ke bawah batas bawah zona, tetapi close tidak di bawahnya |

### 3.2 Terbentuk FVG sejenis yang baru

Begitu FVG sejenis terbentuk setelahnya, FVG sebelumnya **langsung invalid**, tanpa menunggu close di luar zona.

- **SIBI** invalid saat **SIBI baru** terbentuk. **BISI** invalid saat **BISI baru** terbentuk.
- Berlaku untuk **FVG sejenis mana pun** yang terbentuk setelahnya, baik yang berurutan (berbagi candle dengan FVG lama) maupun yang terpisah beberapa candle.
- Hanya berlaku untuk FVG **sejenis**, dan dievaluasi **per timeframe** (FVG baru di timeframe yang sama).
- FVG baru dianggap terbentuk saat **candle 3**-nya selesai (close).
- Invalid bersifat **permanen**. FVG lama tidak hidup kembali, meskipun FVG baru yang menggantikannya kemudian invalid.
- Jika ada beberapa FVG sejenis, hanya **FVG sejenis yang paling baru** yang bisa valid (selama belum invalid karena close di luar zonanya).

Contoh:
- **BISI berurutan:** BISI 1 terbentuk dari candle 2, 3, 4, lalu BISI 2 dari candle 3, 4, 5. Saat BISI 2 terbentuk, **BISI 1 langsung invalid**.
- **SIBI terpisah:** SIBI 1 terbentuk dari candle 1, 2, 3, lalu SIBI 2 dari candle 5, 6, 7 (terpisah satu candle dan tidak berbagi candle). Saat SIBI 2 terbentuk, **SIBI 1 langsung invalid**, dan SIBI 2 yang valid (selama SIBI 2 sendiri belum invalid).

### Pseudocode

```text
# SIBI (bearish FVG)
if candle.close > low(candle1):
    return "SIBI INVALID"
if new_sibi_formed_after(this_sibi):
    return "SIBI INVALID (digantikan SIBI baru)"

# BISI (bullish FVG)
if candle.close < high(candle1):
    return "BISI INVALID"
if new_bisi_formed_after(this_bisi):
    return "BISI INVALID (digantikan BISI baru)"
```

### 3.3 Penamaan: iFVG

SIBI atau BISI yang invalid karena **close** di luar batas zona (kondisi 3.1) disebut **iFVG (Inversion Fair Value Gap)**.

| Penyebab invalid | Nama |
|---|---|
| Close di luar batas zona (3.1) | **iFVG** |
| Digantikan FVG sejenis yang baru (3.2) | **Bukan iFVG.** Hanya invalid, abaikan (tidak perlu dihiraukan). |

SIBI/BISI yang masih valid tetap disebut SIBI/BISI.

## 4. Perbandingan SIBI vs BISI

| Komponen | Bearish FVG / SIBI | Bullish FVG / BISI |
|---|---|---|
| Candle 2 | Turun cepat | Naik cepat |
| Pembentukan | Low candle 1 > high candle 3 | High candle 1 < low candle 3 |
| Batas atas zona | Low candle 1 | Low candle 3 |
| Batas bawah zona | High candle 3 | High candle 1 |
| Invalid jika | Close di atas batas atas zona | Close di bawah batas bawah zona |
| Invalid juga jika | Terbentuk SIBI baru setelahnya | Terbentuk BISI baru setelahnya |
| Nama jika invalid karena close | iFVG | iFVG |
| Nama jika digantikan FVG sejenis baru | Bukan iFVG, hanya invalid (abaikan) | Bukan iFVG, hanya invalid (abaikan) |

## 5. Panduan Output untuk AI

Saat diminta mengidentifikasi SIBI/BISI, jawab dengan format berikut:

1. **Timeframe**: sebutkan timeframe chart. Jika tidak terlihat di chart, tanyakan kepada pengguna.
2. **Jenis FVG**: bearish (SIBI) atau bullish (BISI).
3. **Candle pembentuk**: sebutkan candle 1, 2, dan 3.
4. **Zona FVG**: sebutkan batas atas dan batas bawahnya, beserta dasar penentuannya (misalnya "low candle 1 = 3.985, high candle 3 = 3.920").
5. **Status FVG**: nyatakan masih valid atau invalid (FVG yang invalid karena close disebut **iFVG**, sedangkan FVG yang digantikan FVG sejenis baru hanya invalid dan diabaikan) berdasarkan close candle setelah candle 3 **dan ada tidaknya FVG sejenis yang terbentuk setelahnya** (lihat Bagian 3 di atas), dan sebutkan candle atau FVG baru yang membuatnya invalid jika ada.
6. Jika data di chart tidak terbaca jelas, sebutkan nilai yang diperkirakan dan beri tahu bahwa itu perkiraan.
