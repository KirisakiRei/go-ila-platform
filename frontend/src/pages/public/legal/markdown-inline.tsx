import type { ReactNode } from 'react'

/**
 * renderInlineBlock menebalkan dan memiringkan teks tanpa mengizinkan HTML mentah.
 *
 * Hanya `**tebal**` dan `*miring*` yang dikenali, karena itu yang dipakai
 * dokumen legal pada umumnya.
 */
export function renderInlineBlock(text: string, keyPrefix: string): ReactNode[] {
  const nodes: ReactNode[] = []
  const pattern = /\*\*([^*]+)\*\*|\*([^*]+)\*/g
  let cursor = 0
  let match: RegExpExecArray | null

  while ((match = pattern.exec(text)) !== null) {
    if (match.index > cursor) nodes.push(text.slice(cursor, match.index))
    if (match[1] !== undefined) {
      nodes.push(
        <strong key={`${keyPrefix}-b-${match.index}`} className="font-semibold">
          {match[1]}
        </strong>,
      )
    } else {
      nodes.push(<em key={`${keyPrefix}-i-${match.index}`}>{match[2]}</em>)
    }
    cursor = match.index + match[0].length
  }

  if (cursor < text.length) nodes.push(text.slice(cursor))
  return nodes
}
