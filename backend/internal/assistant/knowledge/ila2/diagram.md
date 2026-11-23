# Diagram Mermaid

Kumpulan diagram dari dokumen-dokumen metode ini. Setiap diagram merujuk ke dokumen sumbernya. Jika ada perbedaan, **dokumen sumber yang berlaku**.

| No | Diagram | Sumber |
|---|---|---|
| 1 | Alur setup dari time cycle sampai entry | `alur-entry.md`, `time-cycle.md` |
| 2 | Jenis key level | `key-level-irl.md`, `key-level-erl.md` |
| 3 | Status key level: valid, iFVG, FTM, atau digantikan | `key-level-irl.md`, `key-level-erl.md` |
| 4 | Evaluasi C2 closure pada IRL (SIBI/BISI) | `c2-closure-irl.md` |
| 5 | Evaluasi C2 closure pada ERL (swing high/low) | `c2-closure-erl.md` |
| 6 | Bearish CISD (-CISD) | `cisd.md` |
| 7 | Hierarki time cycle | `time-cycle.md` |

---

## 1. Alur setup dari time cycle sampai entry

```mermaid
flowchart TD
    A["Pilih time cycle<br/>Yearly sampai Micro"] --> B["Tentukan HTF dan LTF<br/>dari tabel time cycle"]
    B --> C["HTF: key level ERL atau IRL<br/>yang valid"]
    C --> D{"C2 closure terjadi<br/>di dalam cycle?"}
    D -- "Belum" --> D1["Tunggu"]
    D -- "Ya" --> E["C3 = candle HTF<br/>setelah C2 aktif"]
    E --> F{"C3 di dalam cycle?"}
    F -- "Tidak" --> X["Setup tidak berlaku"]
    F -- "Ya" --> G["Arah setup<br/>C2 di swing high atau SIBI = bearish, cari -CISD<br/>C2 di swing low atau BISI = bullish, cari +CISD"]
    G --> H["LTF: cari CISD selama C3 berjalan<br/>key level wajib ERL"]
    H --> I{"CISD terkonfirmasi<br/>sebelum C3 close?"}
    I -- "Tidak" --> J["Setup invalid"]
    I -- "Ya" --> K["Entry di close candle<br/>yang menembus level CISD"]
    K --> L["Stop loss: swing yang terbentuk<br/>saat harga menyapu ERL"]
    L --> M["Target: RR 1:2"]
```

---

## 2. Jenis key level

```mermaid
flowchart LR
    KL["Key level<br/>dievaluasi per timeframe"] --> IRL["IRL<br/>Internal Range Liquidity"]
    KL --> ERL["ERL<br/>External Range Liquidity"]
    IRL --> SIBI["SIBI<br/>bearish FVG"]
    IRL --> BISI["BISI<br/>bullish FVG"]
    ERL --> SH["Swing high"]
    ERL --> SL["Swing low"]
    SIBI -. "invalid karena close" .-> IFVG["iFVG"]
    BISI -. "invalid karena close" .-> IFVG
    SH -. "invalid karena close" .-> FTM["FTM<br/>Failure to Manipulate"]
    SL -. "invalid karena close" .-> FTM
```

---

## 3. Status key level

Berlaku untuk SIBI, BISI, swing high, dan swing low. Invalid bersifat permanen, dan hanya key level sejenis yang paling baru yang bisa valid.

```mermaid
flowchart TD
    V["Key level valid"] --> Q["Peristiwa mana yang terjadi lebih dulu?"]
    Q --> Q1["Close body di luar level atau zona<br/>SIBI: di atas batas atas<br/>BISI: di bawah batas bawah<br/>Swing high: di atas level<br/>Swing low: di bawah level"]
    Q --> Q2["Terbentuk key level sejenis<br/>yang lebih baru"]
    Q1 --> N1["SIBI atau BISI menjadi iFVG<br/>Swing high atau swing low menjadi FTM"]
    Q2 --> N2["Invalid saja, diabaikan<br/>bukan iFVG, bukan FTM"]
    N1 --> P["Invalid permanen<br/>tidak hidup kembali"]
    N2 --> P
    V --> OK["Tidak ada peristiwa di atas<br/>masih valid"]
```

---

## 4. Evaluasi C2 closure pada IRL (SIBI/BISI)

Bearish dan bullish adalah kebalikan. Evaluasi candle satu per satu setelah candle 3, dengan urutan ini.

```mermaid
flowchart TD
    P["Acuan awal<br/>SIBI: high candle 3<br/>BISI: low candle 3"] --> S["Candle berikutnya"]
    S --> A{"Terbentuk FVG sejenis baru<br/>dengan candle ini sebagai candle 3?"}
    A -- "Ya" --> I1["FVG invalid, bukan iFVG<br/>C2 aktif gugur"]
    A -- "Tidak" --> B{"Close body di luar batas zona?<br/>SIBI: di atas batas atas<br/>BISI: di bawah batas bawah"}
    B -- "Ya" --> I2["iFVG<br/>C2 aktif gugur"]
    B -- "Tidak" --> C{"Wick menyentuh atau melewati acuan?"}
    C -- "Tidak" --> D["Diabaikan<br/>C2 aktif dan acuan tetap"]
    C -- "Ya" --> E{"Close di sisi penolakan?<br/>SIBI: di bawah acuan<br/>BISI: di atas acuan"}
    E -- "Ya" --> F["C2 closure<br/>C2 terbentuk atau berpindah<br/>acuan baru = high atau low candle ini"]
    E -- "Tidak" --> G["Bukan C2<br/>C2 aktif gugur<br/>acuan bergeser ke high atau low candle ini"]
```

---

## 5. Evaluasi C2 closure pada ERL (swing high/low)

Swing low adalah sisi bullish, swing high adalah kebalikannya. Evaluasi dimulai dari candle setelah candle kanan pembentuk swing.

```mermaid
flowchart TD
    P["Acuan awal = level swing"] --> S["Candle berikutnya<br/>setelah candle kanan pembentuk swing"]
    S --> A{"Terbentuk swing sejenis baru<br/>dengan candle ini sebagai candle kanan?"}
    A -- "Ya" --> I1["Swing invalid, bukan FTM<br/>C2 aktif gugur"]
    A -- "Tidak" --> B{"Close body di luar level swing?<br/>Swing low: di bawah level<br/>Swing high: di atas level"}
    B -- "Ya" --> I2["FTM<br/>C2 aktif gugur"]
    B -- "Tidak" --> C{"Wick menyentuh atau melewati acuan?"}
    C -- "Tidak" --> D["Diabaikan<br/>C2 aktif tetap"]
    C -- "Ya" --> E{"Close di sisi penolakan?<br/>Swing low: di atas acuan<br/>Swing high: di bawah acuan"}
    E -- "Ya" --> F["C2 closure<br/>C2 terbentuk atau berpindah<br/>acuan baru = low atau high candle ini"]
```

---

## 6. Bearish CISD (-CISD)

Bullish CISD (+CISD) adalah kebalikannya: swing low, rangkaian candle bearish, open candle bearish pertama, dan body bullish yang menembus level.

```mermaid
flowchart TD
    A["LTF: swing high terbaru<br/>key level wajib ERL"] --> B{"Status swing high"}
    B -- "Valid dan terkena wick" --> K1["Kondisi 1"]
    B -- "FTM" --> C{"Ada iFVG terbaru<br/>di rangkaian candle bullish?"}
    C -- "Ya" --> K2["Kondisi 2"]
    C -- "Tidak" --> D{"Swing low terbaru FTM<br/>close body di bawahnya?"}
    D -- "Ya" --> K3["Kondisi 3<br/>CISD dan swing low FTM<br/>wajib keduanya"]
    D -- "Tidak" --> X["Tidak berlaku"]
    K1 --> L["Level CISD =<br/>open candle bullish pertama<br/>dalam rangkaian"]
    K2 --> L
    K3 --> L
    L --> M{"Body candle bearish<br/>close di bawah level CISD?"}
    M -- "Ya" --> OK["-CISD terkonfirmasi"]
    M -- "Tidak" --> W["Belum terkonfirmasi<br/>wick saja tidak dihitung"]
```

---

## 7. Hierarki time cycle

Semua cycle aktif bersamaan. Jam mulai semua putaran adalah 18:00 NY. Kotak menunjukkan HTF dan LTF yang dipakai bila cycle tersebut dipilih.

```mermaid
flowchart TD
    Y["Yearly<br/>HTF Quarterly, LTF Weekly"] --> Q["Quarterly<br/>HTF Monthly, LTF Daily"]
    Q --> M["Monthly<br/>HTF Weekly, LTF 4H"]
    M --> W["Weekly<br/>Minggu 18:00 sampai Jumat 18:00 NY<br/>HTF Daily, LTF 1H"]
    W --> D["Daily<br/>24 jam<br/>HTF 4H, LTF 15m"]
    D --> N["90M cycle<br/>6 jam<br/>HTF 1H, LTF 5m"]
    N --> MI["Micro cycle<br/>90 menit<br/>HTF 15m, LTF 1m"]
```
