# README: Urutan Membaca Dokumen

Kumpulan dokumen ini mendefinisikan satu metode membaca chart candlestick: **key level**, **C2 closure**, **CISD**, **time cycle**, dan **alur entry**. Dokumen saling merujuk, jadi **baca berurutan** seperti di bawah.

**Aturan utama untuk AI:**
- Gunakan hanya aturan yang ada di dokumen-dokumen ini. Jangan menambah aturan sendiri.
- Jika informasi yang dibutuhkan tidak ada di dokumen atau tidak terlihat di chart (misalnya timeframe atau cycle yang dipakai), **tanyakan kepada pengguna**. Jangan menebak.
- Semua level dan waktu yang dibaca dari gambar adalah perkiraan. Sebutkan itu kepada pengguna.
- Output ini bukan nasihat keuangan.

---

# Urutan Baca

| Urutan | File | Yang didefinisikan | Perlu dibaca lebih dulu |
|---|---|---|---|
| **1** | `key-level-irl.md` | **IRL**: SIBI (bearish FVG) dan BISI (bullish FVG), pembentukan, invalidasi, **iFVG** | Tidak ada |
| **2** | `key-level-erl.md` | **ERL**: swing high dan swing low, pembentukan, invalidasi, **FTM** | Tidak ada |
| **3** | `c2-closure-irl.md` | **C2 closure** pada SIBI/BISI (C2 bisa berpindah atau gugur) | 1 |
| **4** | `c2-closure-erl.md` | **C2 closure** pada swing high/low | 2 |
| **5** | `cisd.md` | **-CISD dan +CISD** di LTF, tiga kondisi key level ERL | 2 (dan 1 untuk iFVG) |
| **6** | `time-cycle.md` | **Time cycle**: penentu HTF dan LTF, jam mulai dan akhir tiap cycle | Tidak ada, tetapi dipakai oleh 7 |
| **7** | `alur-entry.md` | **Alur lengkap** dari HTF ke entry, stop loss, dan target | 1 sampai 6 |

Dokumen 7 (`alur-entry.md`) menyatukan semuanya, jadi baca terakhir.

---

# Gambaran Besar

```text
key-level-irl.md ──┐
                   ├─> c2-closure-irl.md ──┐
key-level-erl.md ──┤                       │
                   ├─> c2-closure-erl.md ──┼─> alur-entry.md
                   └─> cisd.md ────────────┤
time-cycle.md ─────────────────────────────┘
```

**Alur setup singkat:**
1. **Time cycle** menentukan HTF dan LTF.
2. Di **HTF**, key level (ERL atau IRL) yang valid mendapat **C2 closure** di dalam cycle.
3. **C3** (candle HTF berikutnya, di dalam cycle): selama C3 berjalan, cari **CISD** di LTF dengan key level ERL.
4. **Entry** di close candle yang menembus level CISD, **stop loss** di swing yang terbentuk saat harga menyapu ERL, **target** RR 1:2.

---

# Istilah Singkat

| Istilah | Arti singkat |
|---|---|
| **IRL / ERL** | Internal / External Range Liquidity (jenis key level) |
| **SIBI / BISI** | Bearish FVG / bullish FVG (key level IRL) |
| **Swing high / swing low** | Key level ERL |
| **iFVG** | SIBI/BISI yang invalid karena close body di luar zona |
| **FTM** | Swing yang invalid karena close body di luar level (Failure to Manipulate) |
| **C2 closure** | Penolakan di key level HTF: wick menyentuh, close kembali di sisi semula |
| **C3** | Candle HTF tepat setelah C2 aktif, tempat entry LTF dicari |
| **CISD** | Change in State of Delivery, konfirmasi entry di LTF |
| **HTF / LTF** | High / low timeframe, ditentukan oleh time cycle |

Definisi lengkap ada di masing-masing dokumen.

---

# Aturan yang Berlaku di Semua Dokumen

- Key level dievaluasi **per timeframe**.
- Invalidasi **hanya** dari **close body**, bukan wick.
- Hanya key level **sejenis yang paling baru** yang bisa valid. Yang lama langsung invalid, dan itu **bukan** iFVG atau FTM.
- Invalid bersifat **permanen**.
- C2, C3, C4, dan seterusnya **hanya untuk HTF**. Di LTF, key level yang dipakai **wajib ERL**.

---

# Cari Jawaban di Mana

| Pertanyaan | Buka |
|---|---|
| Bagaimana FVG (SIBI/BISI) terbentuk dan kapan invalid? | `key-level-irl.md` |
| Bagaimana swing high/low terbentuk dan kapan invalid? | `key-level-erl.md` |
| Apakah sebuah FVG sudah mendapat C2 closure? | `c2-closure-irl.md` |
| Apakah sebuah swing sudah mendapat C2 closure? | `c2-closure-erl.md` |
| Apakah -CISD atau +CISD sudah terkonfirmasi di LTF? | `cisd.md` |
| HTF dan LTF mana yang dipakai, dan di mana posisi waktunya? | `time-cycle.md` |
| Bagaimana entry, stop loss, dan target ditentukan? | `alur-entry.md` |
| Evaluasi chart dari awal sampai akhir | Baca semuanya, mulai dari `alur-entry.md` sebagai kerangka |

---

# Catatan untuk Pembacaan Chart

- Elemen chart yang **tidak didefinisikan** di dokumen (misalnya panel indikator di bawah chart, garis horizontal tanpa label, atau garis putus-putus tanpa label) **diabaikan**. Jangan ditafsirkan.
- Setiap dokumen punya bagian **Panduan Output untuk AI**. Ikuti format jawaban di bagian itu.
- Jika candle masih berjalan, nyatakan bahwa statusnya belum final.
