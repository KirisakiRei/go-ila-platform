# Key Level ERL (Swing High / Swing Low)

Dokumen ini mendefinisikan key level **ERL (External Range Liquidity)**, yaitu **Swing High** dan **Swing Low**. Gunakan dokumen ini sebagai aturan baku saat mengevaluasi chart candlestick. Jangan menambah aturan di luar dokumen ini.

**Konteks:** setiap timeframe memiliki key level ERL-nya sendiri. Swing high/low diidentifikasi menggunakan candle dari timeframe yang sedang dianalisis. Selalu sebutkan timeframe yang dipakai saat mengevaluasi.

---

## 1. Swing Low

Swing low adalah pola **3 candle berurutan** dengan candle tengah sebagai titiknya.

**Syarat:** **low candle tengah lebih rendah** daripada **low candle sebelumnya** (kiri) **dan** **low candle sesudahnya** (kanan).

```
Candle kiri     Candle tengah     Candle kanan
    |                                 |
  low kiri      low tengah (paling    low kanan
  (lebih tinggi)   rendah)            (lebih tinggi)
                    ^
              SWING LOW
```

- Level swing low = **low candle tengah**.
- Garis swing low ditarik **horizontal dari low candle tengah ke arah kanan**.

---

## 2. Swing High

Swing high adalah pola **3 candle berurutan** dengan candle tengah sebagai titiknya.

**Syarat:** **high candle tengah lebih tinggi** daripada **high candle sebelumnya** (kiri) **dan** **high candle sesudahnya** (kanan).

```
                    SWING HIGH
                        v
                   high tengah (paling tinggi)
  high kiri                                  high kanan
  (lebih rendah)                             (lebih rendah)
Candle kiri     Candle tengah     Candle kanan
```

- Level swing high = **high candle tengah**.
- Garis swing high ditarik **horizontal dari high candle tengah ke arah kanan**.

---

## 3. Aturan Penting

- Yang dibandingkan hanya **wick (high dan low)**, bukan body atau close.
- Hanya **satu candle di kiri dan satu candle di kanan** yang dibandingkan. Candle yang lebih jauh tidak ikut menentukan.
- **Warna candle tidak berpengaruh.** Candle tengah boleh bullish atau bearish.
- Swing baru bisa dipastikan **setelah candle di sebelah kanan terbentuk (close)**.

### Pseudocode

```text
for i in range(1, len(candles) - 1):
    left, mid, right = candles[i-1], candles[i], candles[i+1]

    if mid.low < left.low and mid.low < right.low:
        swing_low  = mid.low     # tarik garis horizontal ke kanan

    if mid.high > left.high and mid.high > right.high:
        swing_high = mid.high    # tarik garis horizontal ke kanan
```

---

## 4. Invalidasi Swing High dan Swing Low

Invalidasi ERL hanya dihitung dari **close**, bukan wick. Wick boleh menembus level, tetapi close tidak boleh menembus.

| Key level | Invalid jika | Tidak invalid jika |
|---|---|---|
| **Swing Low** | **Close di bawah** swing low | Wick menembus ke bawah swing low, tetapi close kembali di atasnya |
| **Swing High** | **Close di atas** swing high | Wick menembus ke atas swing high, tetapi close kembali di bawahnya |

### Pseudocode

```text
# Swing Low
if candle.close < swing_low:
    return "SWING LOW INVALID"

# Swing High
if candle.close > swing_high:
    return "SWING HIGH INVALID"
```

### Terbentuk swing sejenis yang baru

Sama seperti FVG, jika ada beberapa swing **sejenis**, swing sebelumnya **langsung invalid** begitu swing sejenis yang baru terbentuk, tanpa menunggu close di luar level.

- **Swing high** invalid saat **swing high baru** terbentuk. **Swing low** invalid saat **swing low baru** terbentuk.
- Berlaku untuk swing sejenis mana pun yang terbentuk setelahnya.
- Hanya berlaku untuk swing **sejenis**, dan dievaluasi **per timeframe**.
- Swing baru dianggap terbentuk saat **candle kanan**-nya selesai (close).
- Invalid bersifat **permanen**. Swing lama tidak hidup kembali, meskipun swing baru yang menggantikannya kemudian invalid.
- Jika ada beberapa swing sejenis, hanya **swing sejenis yang paling baru** yang bisa valid (selama belum invalid karena close di luar levelnya).

Contoh: ada swing high di candle 7 dan swing high di candle 9. Saat swing high candle 9 terbentuk, **swing high candle 7 langsung invalid**, dan yang valid adalah swing high candle 9.

### Pseudocode

```text
# Swing High
if new_swing_high_formed_after(this_swing_high):
    return "SWING HIGH INVALID (digantikan swing high baru)"

# Swing Low
if new_swing_low_formed_after(this_swing_low):
    return "SWING LOW INVALID (digantikan swing low baru)"
```

### Penamaan: FTM

Swing high atau swing low yang invalid karena **close** di luar level disebut **FTM (Failure to Manipulate)**.

| Kondisi | Nama |
|---|---|
| Swing high / swing low masih valid | Swing high / swing low |
| Swing high / swing low invalid karena close di luar level | **FTM** |
| Swing high / swing low digantikan swing sejenis yang baru | **Bukan FTM.** Hanya invalid, abaikan (tidak perlu dihiraukan). |

## 5. Panduan Output untuk AI

Saat diminta mengidentifikasi swing, jawab dengan format berikut:

1. **Timeframe**: sebutkan timeframe chart. Jika tidak terlihat di chart, tanyakan kepada pengguna.
2. **Daftar swing**: untuk tiap swing, sebutkan jenisnya (swing high atau swing low), candle tengahnya, dan levelnya (high atau low candle tengah).
3. **Dasar penentuan**: sebutkan perbandingan dengan candle kiri dan kanan (misalnya "low tengah 3.870 lebih rendah dari low kiri 3.900 dan low kanan 3.890").
4. Jika candle kanan belum terbentuk (masih berjalan), nyatakan bahwa swing tersebut belum bisa dipastikan.
5. **Status swing**: untuk tiap swing, nyatakan masih valid atau invalid (swing yang invalid karena close disebut **FTM**, sedangkan swing yang digantikan swing sejenis baru hanya invalid dan diabaikan) berdasarkan close candle setelahnya **dan ada tidaknya swing sejenis yang terbentuk setelahnya** (lihat Bagian 4 di atas), dan sebutkan candle yang membuatnya invalid jika ada.
6. Jika data di chart tidak terbaca jelas, sebutkan nilai yang diperkirakan dan beri tahu bahwa itu perkiraan.
