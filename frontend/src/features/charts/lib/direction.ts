/**
 * arahBearish menormalkan penamaan arah dari engine dan dari order.
 *
 * Engine ILA menulis arah struktur sebagai "BEARISH"/"BULLISH" (lihat
 * pkg/ila/evaluator.go), sedangkan order memakai "SELL"/"BUY". Overlay chart
 * harus menerima keduanya: tanpa normalisasi, perbandingan `direction ===
 * 'SELL'` selalu salah dan level sweep digambar di sisi yang keliru.
 */
export function arahBearish(direction?: string | null): boolean {
  const value = (direction ?? '').trim().toUpperCase()
  return value === 'BEARISH' || value === 'SELL' || value === 'SHORT'
}
