# MEMORY.md — Aturan Operasional ILA Platform

Fakta tetap mengenai cara kerja sistem ILA Platform yang harus dipahami asisten:

1. **Strategi di Asisten Ini**:
   - Pengetahuan analisis di asisten ini hanya mencakup **New ILA 2.0**. Pasangan timeframe resmi New ILA 2.0 adalah: `Quarterly→W`, `Monthly→D1`, `W→H4`, `D1→H1`, `H4→M15`, `H1→M5`, `M15→M1` (acuan batas hari/cycle mengikuti 18:00 NY).
   - Bila akun pengguna memiliki posisi atau sinyal dari strategi lain (`ila-3.0`), Anda tetap bisa membacakan data faktualnya dari tool, namun jelaskan bahwa analisis teori mendalam di asisten ini khusus untuk ILA 2.0.

2. **Posisi Akun vs Sinyal Tidak Tereksekusi (Simulasi)**:
   - **Tabel `position`** mencatat order nyata/dry-run yang dieksekusi di akun broker (`OPEN`, `CLOSED_TP`, `CLOSED_SL`, `CLOSED_MANUAL`).
   - **Sinyal tidak tereksekusi** (`user_signal` dengan status ditolak/dilewati/kedaluwarsa sebelum entry) tetap dipantau oleh sistem simulasi platform untuk melihat apakah harga menyentuh Entry lalu TP atau SL (`eval_status`: `RUNNING`, `TP`, `SL`, `NO_ENTRY`, `PENDING`, `UNKNOWN`). Jangan mencampuradukkan hasil simulasi ini dengan saldo/PnL akun broker.

3. **Aturan Pasar & Rollover**:
   - Sistem menggunakan waktu harian default **WIB (`Asia/Jakarta`)** untuk batas trade harian dan laporan dashboard, serta waktu **New York (`America/New_York`)** untuk Time Cycle ILA 2.0.
   - Jam **05:00–07:00 WIB** (sekitar 22:00–00:00 UTC) merupakan jendela *rollover* antar-sesi pasar di mana spread broker sering melebar dan eksekusi baru dibatasi.

4. **Sumber Candle**:
   - Setiap sinyal menyimpan `feed_source` (mis. `FOREXCOM`, `ICMARKETS`, `BINANCE`) agar candle yang dilihat di chart dashboard identik dengan candle yang dibaca oleh engine.
