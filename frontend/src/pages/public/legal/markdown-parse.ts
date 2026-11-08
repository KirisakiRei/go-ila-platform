/**
 * Parser markdown untuk dokumen legal.
 *
 * Dokumen resmi disimpan di basis data sebagai markdown dan boleh diubah admin
 * kapan saja lewat CMS, sehingga perendernya sengaja dibuat sempit dan dapat
 * diprediksi: judul, paragraf, daftar, kutipan, tabel, serta penanda blok khusus.
 *
 * Penanda blok khusus ditulis sebagai blockquote berlabel:
 *
 *   > [!penting] Judul blok
 *   > Isi peringatan
 *
 * Jenis yang dikenali: penting, perhatian, catatan.
 */

export type CalloutTone = 'penting' | 'perhatian' | 'catatan'

export type Block =
  | { kind: 'heading'; level: 1 | 2 | 3; text: string }
  | { kind: 'paragraph'; text: string }
  | { kind: 'list'; ordered: boolean; items: string[] }
  | { kind: 'quote'; lines: string[] }
  | { kind: 'callout'; tone: CalloutTone; title: string; lines: string[] }
  | { kind: 'table'; head: string[]; rows: string[][] }
  | { kind: 'rule' }

export const CALLOUT_LABEL: Record<CalloutTone, string> = {
  penting: 'Penting',
  perhatian: 'Perhatian',
  catatan: 'Catatan',
}

/** slugHeading membuat id stabil yang bisa dipakai tautan langsung ke pasal. */
export function slugHeading(text: string): string {
  return text
    .toLowerCase()
    .normalize('NFD')
    .replace(/[\u0300-\u036f]/g, '')
    .replace(/[^a-z0-9\s-]/g, '')
    .trim()
    .replace(/\s+/g, '-')
}

/** splitRow memecah satu baris tabel markdown menjadi sel. */
function splitRow(line: string): string[] {
  return line
    .trim()
    .replace(/^\|/, '')
    .replace(/\|$/, '')
    .split('|')
    .map((cell) => cell.trim())
}

/** isBlockStart menandai baris yang memulai blok baru, bukan lanjutan paragraf. */
function isBlockStart(line: string): boolean {
  const trimmed = line.trim()
  return (
    /^#{1,3}\s+/.test(trimmed) ||
    trimmed.startsWith('|') ||
    trimmed.startsWith('>') ||
    /^\d+[.)]\s+/.test(trimmed) ||
    /^[-*+]\s+/.test(trimmed) ||
    /^(---|\*\*\*|___)$/.test(trimmed)
  )
}

/** collectList mengumpulkan item daftar sampai menemui baris yang bukan item. */
function collectList(
  lines: string[],
  start: number,
  pattern: RegExp,
): { items: string[]; next: number } {
  const items: string[] = []
  let i = start

  while (i < lines.length) {
    const current = lines[i] ?? ''
    const item = current.match(pattern)
    if (item) {
      items.push(item[1] ?? '')
      i += 1
      continue
    }
    // Baris menjorok melanjutkan item sebelumnya, bukan item baru.
    if (current.trim() !== '' && /^\s{2,}\S/.test(current) && items.length > 0) {
      const last = items.length - 1
      items[last] = `${items[last] ?? ''} ${current.trim()}`.trim()
      i += 1
      continue
    }
    break
  }

  return { items, next: i }
}

/** parseMarkdown memecah markdown menjadi blok yang siap dirender. */
export function parseMarkdown(source: string): Block[] {
  const lines = source.replace(/\r\n/g, '\n').split('\n')
  const blocks: Block[] = []
  let i = 0

  /** lineAt mengembalikan baris atau string kosong bila indeks melewati akhir. */
  const lineAt = (index: number): string => lines[index] ?? ''

  while (i < lines.length) {
    const line = lineAt(i)

    if (line.trim() === '') {
      i += 1
      continue
    }

    if (/^(---|\*\*\*|___)\s*$/.test(line.trim())) {
      blocks.push({ kind: 'rule' })
      i += 1
      continue
    }

    const heading = line.match(/^(#{1,3})\s+(.*)$/)
    if (heading) {
      const level = (heading[1] ?? '#').length
      blocks.push({
        kind: 'heading',
        level: (level >= 1 && level <= 3 ? level : 2) as 1 | 2 | 3,
        text: heading[2]?.trim() ?? '',
      })
      i += 1
      continue
    }

    if (line.trim().startsWith('|') && lineAt(i + 1).includes('---')) {
      const head = splitRow(line)
      i += 2
      const rows: string[][] = []
      while (i < lines.length && lineAt(i).trim().startsWith('|')) {
        rows.push(splitRow(lineAt(i)))
        i += 1
      }
      blocks.push({ kind: 'table', head, rows })
      continue
    }

    if (line.trim().startsWith('>')) {
      const raw: string[] = []
      while (i < lines.length && lineAt(i).trim().startsWith('>')) {
        raw.push(lineAt(i).trim().replace(/^>\s?/, ''))
        i += 1
      }
      const marker = (raw[0] ?? '').match(/^\[!(penting|perhatian|catatan)\]\s*(.*)$/i)
      if (marker) {
        const tone = (marker[1] ?? 'catatan').toLowerCase() as CalloutTone
        const rest = [marker[2] ?? '', ...raw.slice(1)].filter((text) => text !== '')
        blocks.push({
          kind: 'callout',
          tone,
          title: rest.shift() ?? CALLOUT_LABEL[tone],
          lines: rest,
        })
      } else {
        blocks.push({ kind: 'quote', lines: raw })
      }
      continue
    }

    if (/^\s*\d+[.)]\s+/.test(line)) {
      const { items, next } = collectList(lines, i, /^\s*\d+[.)]\s+(.*)$/)
      blocks.push({ kind: 'list', ordered: true, items })
      i = next
      continue
    }

    if (/^\s*[-*+]\s+/.test(line)) {
      const { items, next } = collectList(lines, i, /^\s*[-*+]\s+(.*)$/)
      blocks.push({ kind: 'list', ordered: false, items })
      i = next
      continue
    }

    const paragraph: string[] = []
    while (i < lines.length && lineAt(i).trim() !== '' && !isBlockStart(lineAt(i))) {
      paragraph.push(lineAt(i).trim())
      i += 1
    }
    if (paragraph.length > 0) blocks.push({ kind: 'paragraph', text: paragraph.join(' ') })
  }

  return blocks
}

/**
 * outlineFromBlocks mengambil daftar pasal (heading level 2) untuk sidebar.
 *
 * Daftar isi selalu dibangun dari hasil parse yang sama dengan isi halaman,
 * sehingga sidebar tidak mungkin menunjuk pasal yang tidak ada.
 */
export function outlineFromBlocks(blocks: Block[]): { id: string; label: string }[] {
  const seen = new Map<string, number>()

  return blocks
    .filter((block): block is Extract<Block, { kind: 'heading' }> => block.kind === 'heading')
    .filter((block) => block.level === 2)
    .map((block) => {
      const base = slugHeading(block.text)
      const count = seen.get(base) ?? 0
      seen.set(base, count + 1)
      return { id: count === 0 ? base : `${base}-${count + 1}`, label: block.text }
    })
}
