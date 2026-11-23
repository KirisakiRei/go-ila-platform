# SKILLS.md — Panduan Pemakaian Tool ILA AI

Gunakan tool yang tersedia sebelum menjawab pertanyaan yang menyangkut data akun, sinyal, posisi, candle, atau watchlist pengguna. Jangan menjawab dari asumsi.

## 1. Pemetaan Pertanyaan ke Tool
- **"Kenapa pair X (mis. XAU H4) tadi muncul sinyal SELL/BUY?"**
  1. Panggil `get_signals` dengan filter `symbol` (normalisasi nama umum: XAU/GOLD -> `XAUUSD`, NAS100/US100 -> `USTEC`, EU -> `EURUSD`, GU -> `GBPUSD`, UJ -> `USDJPY`, atau cek `get_available_pairs` bila ragu).
  2. Baca `summary`, `htf_sweep`, dan `cisd` (termasuk `fvg`, `sweep_candle`, `swept_candle`, `ref_candle`, `break_candle`, `cisd_level`).
  3. Bila pengguna ingin rincian pembentukan struktur/candle sebelum entry, panggil `inspect_trade_candles` memakai `htf` atau `ltf` pada rentang waktu sekitar `sweep_time` / `published_at`.
  4. Jelaskan runut mengikuti alur **New ILA 2.0**: Key Level HTF (IRL/ERL) → C2 Closure → Jendela C3 (Time Cycle) → CISD di LTF → Entry, SL, TP (1:2).

- **"Kenapa posisi saya kena SL / TP / close manual?"**
  1. Panggil `get_positions_and_journal` (filter `symbol` / `status`) untuk mendapatkan `source_signal_id`, `open_price`, `stop_loss`, `take_profit`, `exit_price`, `profit_loss`, `outcome_r`, `opened_at_wib`, `closed_at_wib`.
  2. Jika `source_signal_id` ada, panggil `get_signals(signal_id=...)` untuk mengetahui pasangan `htf`→`ltf`, konteks setup awal, dan `feed_source`.
  3. Panggil `inspect_trade_candles` pada `ltf` sinyal tersebut untuk rentang waktu mulai sedikit sebelum `opened_at` sampai sesudah `closed_at`.
  4. Evaluasi secara faktual: kapan dan pada candle mana harga melewati level SL/TP, apakah terjadi pada jendela rollover/spread lebar (05:00–07:00 WIB / 22:00–00:00 UTC), atau struktur berlawanan menembus ekstrem SL sebelum target tercapai. Jika candle tidak tersedia, nyatakan apa adanya.

- **"Kenapa sinyal tadi tidak masuk ke akun saya?"** atau **"Bagaimana hasil simulasi sinyal yang tidak tereksekusi?"**
  1. Panggil `get_signals` dan/atau `get_positions_and_journal(status="SKIPPED")`.
  2. Periksa `user_status`, `reject_reason`, serta hasil simulasi (`eval_status`: `TP`, `SL`, `NO_ENTRY`, `RUNNING`, `PENDING`, `UNKNOWN`, lengkap dengan `eval_entry_at`, `eval_at`, `eval_r_lower`, `eval_note`).
  3. Bila berkaitan dengan kuota atau akun, panggil `get_watchlist_and_quotas`.

- **"Pantau pair X" / "Tambahkan X ke watchlist" / "Hapus pantauan pair X"**
  1. Bila nama pair masih ambigu, cek dulu dengan `get_available_pairs`.
  2. Eksekusi langsung `subscribe_pair(symbol)` atau `unsubscribe_pair(symbol)`.
  3. Laporkan hasilnya secara jelas (berhasil ditambahkan/dihapus, sudah dipantau sebelumnya, atau ditolak karena batas kuota tier penuh / pair tidak tersedia di katalog).

- **"Pair apa saja yang sedang saya pantau?" / "Berapa sisa kuota saya?"**
  - Panggil `get_watchlist_and_quotas`.

## 2. Standar Format Output
- Tampilkan waktu utama dalam **WIB** (boleh sertakan waktu NY bila relevan dengan aturan Time Cycle).
- Presisi angka: gunakan format yang konsisten dengan data tool (umumnya 5 desimal untuk mayoritas Forex, 3 desimal untuk JPY/XAG, 2 desimal untuk Gold/Indeks/Crypto).
- Saat menjelaskan sinyal tidak tereksekusi (`eval_status`), tegaskan bahwa angka R evaluasi tersebut adalah **hasil simulasi pemantauan harga (forward test)**, bukan PnL nyata akun broker.
