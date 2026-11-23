# Time Cycle

Dokumen ini mendefinisikan **time cycle**, yaitu penentu **HTF (High Timeframe)** dan **LTF (Low Timeframe)** yang dipakai dalam setup.

Gunakan dokumen ini sebagai aturan baku. Jangan menambah aturan di luar dokumen ini. Alur setup setelah HTF dan LTF ditentukan ada di `alur-entry.md`. Aturan key level ada di `key-level-irl.md` dan `key-level-erl.md`, C2 closure di `c2-closure-irl.md` dan `c2-closure-erl.md`, dan CISD di `cisd.md`.

---

# Konsep

- **Time cycle** menentukan kapan kita memakai HTF dan LTF yang mana.
- **Semua time cycle aktif bersamaan**, dari **Yearly** sampai **Micro**. Pada satu waktu, ketujuh cycle sedang berjalan.
- Kita **memilih** cycle yang mau dipakai untuk setup. Pilihan cycle menentukan HTF (tempat key level dan C2 closure dicari) dan LTF (tempat entry dicari).
- Setiap cycle adalah satu **putaran** yang berulang. Dalam satu putaran, key level di HTF-nya terkena dan membentuk **C2 closure**. Itu sebabnya disebut time cycle.
  - Contoh: key level **4H** membentuk C2 closure **setiap hari**. Key level **Daily** biasanya membentuk C2 closure **setiap minggu**.

# Tabel Time Cycle

| Time cycle | Durasi satu putaran | HTF | LTF | C2 closure di key level HTF |
|---|---|---|---|---|
| **Yearly** | 1 tahun (week 1 bulan Januari sampai week 4 bulan Desember) | Quarterly | Weekly | Setiap tahun |
| **Quarterly** | 1 kuartal (week 1 bulan pertama sampai week 4 bulan ketiga) | Monthly | Daily | Setiap kuartal |
| **Monthly** | 1 bulan (week 1 Minggu 18:00 sampai week 4 Jumat 18:00 NY) | Weekly | 4H | Setiap bulan |
| **Weekly** | 1 minggu (Minggu 18:00 sampai Jumat 18:00 NY) | Daily | 1H | Setiap minggu |
| **Daily** | 1 hari (24 jam) | 4H | 15m | Setiap hari |
| **90M** | 6 jam | 1H | 5m | Setiap 6 jam |
| **Micro** | 90 menit | 15m | 1m | Setiap 90 menit |

Catatan: nama **90M** tidak sama dengan durasinya. **90M cycle berdurasi 6 jam**, dan **Micro cycle berdurasi 90 menit**.

Pasangan HTF > LTF di tabel ini sama dengan tabel pasangan timeframe di `alur-entry.md`.

# Waktu (Acuan NY Time)

- Waktu acuan adalah **NY time**.
- **Hari trading dimulai jam 18:00 NY.**
- **Semua putaran cycle dimulai jam 18:00 NY**, baik Daily, Weekly, Monthly, Quarterly, maupun Yearly. Jam mulainya sama untuk semua cycle.
- **Daily cycle:** 24 jam, dari 18:00 NY sampai 18:00 NY berikutnya.
- **90M cycle (6 jam):** 24 jam dibagi 4, sehingga ada empat blok per hari:

| Blok 90M cycle |
|---|
| 18:00 sampai 00:00 |
| 00:00 sampai 06:00 |
| 06:00 sampai 12:00 |
| 12:00 sampai 18:00 |

- **Micro cycle (90 menit):** empat putaran di dalam tiap blok 6 jam. Contoh di blok 00:00 sampai 06:00: 00:00, 01:30, 03:00, dan 04:30.
- **Weekly cycle:** mulai **hari Minggu 18:00 NY**, berakhir **hari Jumat 18:00 NY**. Di dalamnya ada hari-hari dalam minggu. Contoh: hari Kamis berarti kita sedang berada di Kamis dalam weekly cycle.
- **Monthly cycle:** mulai di **week 1 pada bulan tersebut, hari Minggu 18:00 NY**, berakhir di **week 4, hari Jumat 18:00 NY**.
  - **Week 1:** jika tanggal 1 jatuh di hari Minggu, minggu itu adalah week 1. Jika tanggal 1 bukan Minggu, minggu itu tidak dihitung week 1, dan week 1 dimulai di **Minggu pertama setelah tanggal 1**.
  - **Minggu ke-5** (jika ada) berada **di luar monthly cycle**, tetapi tetap masuk **weekly, daily, 90M, dan micro cycle**.
- **Quarterly cycle:** mulai di **week 1 pada bulan pertama kuartal** (aturan week 1 sama seperti monthly cycle, Minggu 18:00 NY), berakhir di **week 4 pada bulan ketiga kuartal, hari Jumat 18:00 NY**.
- **Yearly cycle:** mulai di **week 1 bulan Januari** (Minggu 18:00 NY), berakhir di **week 4 bulan Desember, hari Jumat 18:00 NY**.

---

# Cara Memakai Time Cycle

1. **Pilih time cycle** yang mau dipakai.
2. **Tentukan HTF dan LTF** dari tabel.
3. **HTF:** tunggu key level HTF (ERL atau IRL) mendapat **C2 closure**. C2 closure harus terjadi **di dalam cycle** yang dipilih.
4. **C3 (HTF):** C3 juga harus **di dalam cycle**. Selama C3 berjalan (open sampai close), cari entry di LTF. Jika tidak ada setup di C3, setup **invalid**.
5. **LTF:** cari **CISD** (lihat `cisd.md`), lalu entry, stop loss, dan target sesuai `alur-entry.md`.

## Contoh

| Cycle yang dipilih | HTF | LTF | Gambaran |
|---|---|---|---|
| **Daily cycle** | 4H | 15m | Tunggu key level 4H terkena dan membentuk C2 closure (terjadi setiap hari). Lalu cari CISD di 15m saat C3 (candle 4H) berjalan. |
| **Weekly cycle** | Daily | 1H | Tunggu key level Daily terkena dan membentuk C2 closure (terjadi setiap minggu). Lalu cari CISD di 1H saat C3 (candle Daily) berjalan. |
| **90M cycle** | 1H | 5m | Tunggu key level 1H membentuk C2 closure (setiap 6 jam). Lalu cari CISD di 5m. |

---

# Aturan Penting

- **Semua cycle aktif bersamaan.** Cycle yang dipakai adalah pilihan pengguna.
- Cycle menentukan **HTF dan LTF**. Aturan key level, C2 closure, CISD, entry, stop loss, dan target tetap sama di semua cycle.
- Penamaan C2, C3, C4, dan seterusnya hanya untuk **HTF**, dan dihitung dari C2 aktif.
- **C2 closure dan C3 wajib di dalam cycle** yang dipilih.
- Entry LTF hanya dicari **selama C3 berjalan** (open sampai close). Jika tidak ada setup di C3, setup **invalid**.
- Setup yang sudah berjalan saat cycle berakhir **dibiarkan saja**.
- **90M cycle berdurasi 6 jam** dan **Micro cycle berdurasi 90 menit**. Jangan menyamakan nama dengan durasinya.
- Waktu memakai **NY time**, dan **semua putaran cycle dimulai jam 18:00 NY** (hari trading mulai jam 18:00 NY).

# Panduan Output untuk AI

1. **Cycle yang dipakai**: sebutkan time cycle, HTF, dan LTF-nya. Jika pengguna tidak menyebutkan cycle-nya, tanyakan kepada pengguna. Jangan menebak.
2. **Posisi waktu**: jika relevan, sebutkan posisi waktu saat ini dalam NY time (hari dalam minggu, blok 6 jam, atau putaran 90 menit).
3. Lanjutkan evaluasi sesuai `alur-entry.md`: key level HTF, C2 closure (di dalam cycle), C3 (di dalam cycle), lalu CISD di LTF selama C3 berjalan.
4. Jika timeframe atau waktu di chart tidak terlihat, tanyakan kepada pengguna.
5. Jika data di chart tidak terbaca jelas, sebutkan nilai yang diperkirakan dan beri tahu bahwa itu perkiraan.
