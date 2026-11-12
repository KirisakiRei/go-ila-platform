/**
 * Presisi harga per pair.
 *
 * Setiap angka pada chart penting: XAUUSD di 4051.31 dan 4051.3 adalah level
 * yang berbeda, sedangkan EURUSD hanya bergerak di digit ke-5 (0.00001). Memakai
 * satu jumlah desimal untuk semua pair akan menyembunyikan detail yang justru
 * dipakai trader untuk memutuskan entry — mis. sweep di 4051.31 tampil 4051.31,
 * tapi EURUSD 1.08467 tampil 1.08 sehingga FVG dan CISD-nya tidak terlihat
 * berjarak sama sekali.
 *
 * Kunci utama adalah `symbol`. Nilai `digits` mengikuti konvensi broker:
 *  - Forex 5 desimal, pair JPY 3 desimal.
 *  - Logam: XAU 2, XAG 3 (perak bergerak lebih halus daripada emas).
 *  - Indeks 2, energi/komoditas 2.
 *  - Kripto 2 untuk BTC/ETH (harga ribuan dolar), 3 untuk yang lebih murah.
 *
 * Bila pair baru ditambahkan admin dan belum terdaftar di sini, `digits` untuk
 * pair itu diturunkan dari besaran harganya (`fallbackDigits`) supaya chart tetap
 * menampilkan angka yang masuk akal alih-alih 0 desimal atau NaN.
 */

/** Jumlah desimal bawaan per simbol kanonik. */
const DIGITS_BY_SYMBOL: Record<string, number> = {
  // Forex mayor & minor
  EURUSD: 5,
  GBPUSD: 5,
  AUDUSD: 5,
  NZDUSD: 5,
  USDCAD: 5,
  USDCHF: 5,
  EURGBP: 5,
  EURCHF: 5,
  EURAUD: 5,
  GBPAUD: 5,
  AUDCAD: 5,
  AUDCHF: 5,
  // Pair yen dikutip 3 desimal (pip = 0.01).
  USDJPY: 3,
  EURJPY: 3,
  GBPJPY: 3,
  AUDJPY: 3,
  CADJPY: 3,
  CHFJPY: 3,
  NZDJPY: 3,

  // Indeks
  USTEC: 2,
  US30: 2,
  US500: 2,
  GER40: 2,
  UK100: 2,
  JP225: 2,
  HK50: 2,

  // Logam & energi
  XAUUSD: 2,
  XAUEUR: 2,
  XAGUSD: 3,
  XAGEUR: 3,
  USOIL: 3,
  UKOIL: 3,
  NATGAS: 4,

  // Kripto
  BTCUSD: 2,
  ETHUSD: 2,
  SOLUSD: 3,
  XRPUSD: 4,
  LTCUSD: 2,
}

/** Alias nama dari engine/broker ke simbol kanonik. */
const ALIAS_TO_SYMBOL: Record<string, string> = {
  GOLD: 'XAUUSD',
  SILVER: 'XAGUSD',
  NAS100: 'USTEC',
  US100: 'USTEC',
  NDX: 'USTEC',
  SPX500: 'US500',
  SP500: 'US500',
  DOW30: 'US30',
  DJ30: 'US30',
  DAX40: 'GER40',
  DE40: 'GER40',
  WTI: 'USOIL',
  CRUDEOIL: 'USOIL',
  XBTUSD: 'BTCUSD',
}

/**
 * normalisasiSymbol membuang pemisah dan sufiks broker agar "EURUSD.a",
 * "EUR/USD", "EURUSDm", dan "eurusd" menunjuk entri yang sama.
 *
 * Urutannya penting: simbol utuh dicocokkan lebih dulu. Tanpa itu "USTEC"
 * akan terpotong menjadi "USTE" hanya karena berakhiran C — dan sufiks satu
 * huruf memang dipakai broker (akun mikro sering menambah "m" atau "c").
 */
function normalisasiSymbol(symbol: string): string {
  const dasar = symbol.toUpperCase().replace(/[^A-Z0-9]/g, '')

  // 1. Simbol utuh atau aliasnya.
  if (DIGITS_BY_SYMBOL[dasar] !== undefined) return dasar
  if (ALIAS_TO_SYMBOL[dasar] !== undefined) return ALIAS_TO_SYMBOL[dasar]

  // 2. Buang sufiks broker terpanjang lebih dulu agar "EURUSDmicro" menjadi
  //    "EURUSD", bukan "EURUSDmicr".
  const SUFIKS = ['MICROM', 'MICRO', 'MINI', 'STANDARD', 'RAW', 'ECN', 'SPOT', 'CASH', 'PRO', 'IDX', 'INDEX', 'M', 'C', 'A']
  for (const sufiks of SUFIKS) {
    if (!dasar.endsWith(sufiks) || dasar.length <= sufiks.length) continue
    const tanpa = dasar.slice(0, -sufiks.length)
    if (DIGITS_BY_SYMBOL[tanpa] !== undefined) return tanpa
    if (ALIAS_TO_SYMBOL[tanpa] !== undefined) return ALIAS_TO_SYMBOL[tanpa]
  }

  return dasar
}

/**
 * fallbackDigits menurunkan desimal dari besaran harga untuk pair yang belum
 * terdaftar. Harga ribuan butuh 2 desimal; harga di bawah 1 butuh 5.
 */
function fallbackDigits(hargaSampler: number): number {
  if (!Number.isFinite(hargaSampler) || hargaSampler <= 0) return 2
  if (hargaSampler >= 1000) return 2
  if (hargaSampler >= 100) return 2
  if (hargaSampler >= 10) return 3
  if (hargaSampler >= 1) return 4
  if (hargaSampler >= 0.1) return 5
  if (hargaSampler >= 0.01) return 6
  return 8
}

/**
 * priceDigits mengembalikan jumlah desimal untuk sebuah pair.
 *
 * `hargaSampler` opsional: dipakai hanya bila simbol belum terdaftar, sehingga
 * pair baru tetap tampil wajar tanpa perlu mengubah berkas ini.
 */
export function priceDigits(symbol: string | null | undefined, hargaSampler?: number): number {
  if (!symbol) return fallbackDigits(hargaSampler ?? 0)
  const kanonik = normalisasiSymbol(symbol)
  const terdaftar = DIGITS_BY_SYMBOL[kanonik]
  if (terdaftar !== undefined) return terdaftar
  return fallbackDigits(hargaSampler ?? 0)
}

/**
 * priceStep adalah kelipatan harga terkecil yang ditampilkan chart.
 *
 * Lightweight-charts memakai `minMove` untuk menentukan seberapa halus sumbu
 * harga digambar; nilainya harus 10^-digits agar label dan pergerakan lilin
 * selaras dengan presisi pair.
 */
export function priceStep(digits: number): number {
  const aman = Math.max(0, Math.min(8, Math.trunc(digits)))
  return Number(`1e-${aman}`)
}
