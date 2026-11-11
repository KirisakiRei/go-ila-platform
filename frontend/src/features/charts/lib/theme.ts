/**
 * Palet chart.
 *
 * Chart mengikuti tema terang/gelap aplikasi. Warna semantik (naik/turun,
 * risiko/target) tetap dipakai karena trader membacanya sebagai konvensi, bukan
 * sebagai hiasan.
 */
export type ChartTheme = {
  background: string
  text: string
  grid: string
  border: string
  crosshair: string
  up: string
  down: string
  upWick: string
  downWick: string
  entry: string
  stopLoss: string
  takeProfit: string
  sweep: string
  cisd: string
  fvg: string
  fvgBorder: string
  profit: string
  loss: string
  neutral: string
}

const LIGHT: ChartTheme = {
  background: '#ffffff',
  text: '#52525b',
  grid: '#f1f1f4',
  border: '#e4e4e7',
  crosshair: '#a1a1aa',
  up: '#059669',
  down: '#e11d48',
  upWick: '#059669',
  downWick: '#e11d48',
  entry: '#2563eb',
  stopLoss: '#e11d48',
  takeProfit: '#059669',
  sweep: '#d97706',
  cisd: '#7c3aed',
  fvg: 'rgba(37, 99, 235, 0.12)',
  fvgBorder: 'rgba(37, 99, 235, 0.45)',
  profit: '#059669',
  loss: '#e11d48',
  neutral: '#71717a',
}

const DARK: ChartTheme = {
  background: '#0b0b0d',
  text: '#a1a1aa',
  grid: '#1c1c20',
  border: '#27272a',
  crosshair: '#71717a',
  up: '#34d399',
  down: '#fb7185',
  upWick: '#34d399',
  downWick: '#fb7185',
  entry: '#60a5fa',
  stopLoss: '#fb7185',
  takeProfit: '#34d399',
  sweep: '#fbbf24',
  cisd: '#a78bfa',
  fvg: 'rgba(96, 165, 250, 0.16)',
  fvgBorder: 'rgba(96, 165, 250, 0.5)',
  profit: '#34d399',
  loss: '#fb7185',
  neutral: '#a1a1aa',
}

/** isDarkMode membaca kelas tema pada elemen html. */
export function isDarkMode(): boolean {
  if (typeof document === 'undefined') return false
  return document.documentElement.classList.contains('dark')
}

export function chartTheme(dark = isDarkMode()): ChartTheme {
  return dark ? DARK : LIGHT
}
