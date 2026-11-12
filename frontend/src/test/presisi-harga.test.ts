import { describe, expect, it } from 'vitest'
import { formatPrice } from '@/lib/format'
import { priceDigits, priceStep } from '@/lib/pricePrecision'

/**
 * Presisi harga adalah hal kecil yang salahnya mahal: level yang terbaca
 * berbeda dari broker akan membuat trader mengira chart meleset.
 */
describe('priceDigits', () => {
  it('memberi 5 desimal untuk forex non-JPY', () => {
    expect(priceDigits('EURUSD')).toBe(5)
    expect(priceDigits('GBPUSD')).toBe(5)
    expect(priceDigits('AUDUSD')).toBe(5)
  })

  it('memberi 3 desimal untuk pair JPY', () => {
    expect(priceDigits('USDJPY')).toBe(3)
    expect(priceDigits('EURJPY')).toBe(3)
    expect(priceDigits('GBPJPY')).toBe(3)
  })

  it('memberi 2 desimal untuk emas dan indeks', () => {
    expect(priceDigits('XAUUSD')).toBe(2)
    expect(priceDigits('XAUEUR')).toBe(2)
    expect(priceDigits('USTEC')).toBe(2)
    expect(priceDigits('US30')).toBe(2)
  })

  it('memberi 3 desimal untuk perak', () => {
    expect(priceDigits('XAGUSD')).toBe(3)
  })

  it('mengenali alias engine dan sufiks broker', () => {
    // Engine kadang mengirim GOLD/NAS100; akun broker menambah sufiks.
    expect(priceDigits('GOLD')).toBe(2)
    expect(priceDigits('NAS100')).toBe(2)
    expect(priceDigits('EURUSDm')).toBe(5)
    expect(priceDigits('EUR/USD')).toBe(5)
    expect(priceDigits('xauusd')).toBe(2)
  })

  it('tidak memotong simbol yang kebetulan berakhiran huruf sufiks', () => {
    // USTEC berakhir dengan C. Bila sufiks dibuang terlalu cepat, simbol ini
    // jatuh ke perkiraan dari harga dan presisinya berubah tanpa sebab.
    expect(priceDigits('USTEC')).toBe(2)
    expect(priceDigits('GER40')).toBe(2)
  })

  it('menurunkan desimal dari harga untuk pair yang belum terdaftar', () => {
    // Pair baru dari admin tetap tampil wajar, bukan 0 desimal.
    expect(priceDigits('ZZZUSD', 4500)).toBe(2)
    expect(priceDigits('ZZZUSD', 1.2345)).toBe(4)
    // Harga di bawah 0,1 butuh digit lebih banyak; 6 bukan kekeliruan.
    expect(priceDigits('ZZZUSD', 0.023456)).toBe(6)
  })

  it('memakai jatuhan aman saat simbol kosong', () => {
    expect(priceDigits('')).toBe(2)
    expect(priceDigits(null)).toBe(2)
    expect(priceDigits(undefined)).toBe(2)
  })
})

describe('priceStep', () => {
  it('menghasilkan kelipatan terkecil yang sesuai jumlah desimal', () => {
    expect(priceStep(5)).toBeCloseTo(0.00001, 10)
    expect(priceStep(3)).toBeCloseTo(0.001, 10)
    expect(priceStep(2)).toBeCloseTo(0.01, 10)
    expect(priceStep(0)).toBe(1)
  })
})

describe('formatPrice', () => {
  it('menampilkan seluruh digit EURUSD tanpa membulat', () => {
    // Kasus yang dulu rusak: 1.08467 tampil "1,08".
    expect(formatPrice(1.08467, 5)).toBe('1,08467')
    expect(formatPrice(1.08471, 5)).toBe('1,08471')
  })

  it('tetap menunjukkan dua angka berbeda yang sebelumnya tampil sama', () => {
    const bawah = formatPrice(1.08471, 5)
    const atas = formatPrice(1.08476, 5)
    expect(bawah).not.toBe(atas)
  })

  it('tidak memakai pemisah ribuan agar level bisa disalin ke broker', () => {
    expect(formatPrice(4051.31, 2)).toBe('4051,31')
    expect(formatPrice(41250.5, 2)).toBe('41250,50')
  })

  it('menampilkan USDJPY tiga desimal', () => {
    expect(formatPrice(157.842, 3)).toBe('157,842')
  })

  it('memakai tanda pisah untuk nilai tidak sah', () => {
    expect(formatPrice(Number.NaN, 2)).toBe('—')
  })
})
