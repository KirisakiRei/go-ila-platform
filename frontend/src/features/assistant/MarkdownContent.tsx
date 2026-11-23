import type { ReactNode } from 'react'

type MarkdownContentProps = {
  content: string
}

/**
 * Perender Markdown ringkas bebas dangerouslySetInnerHTML:
 * mendukung heading, paragraf, daftar, blockquote, tabel harga (.tabular),
 * serta format inline (**tebal**, *miring*, `kode`).
 */
export function MarkdownContent({ content }: MarkdownContentProps) {
  const lines = content.replace(/\r\n/g, '\n').split('\n')
  const lineAt = (idx: number): string => lines[idx] ?? ''
  const blocks: ReactNode[] = []

  let i = 0
  let key = 0
  while (i < lines.length) {
    const trimmed = lineAt(i).trim()

    if (!trimmed) {
      i++
      continue
    }

    // Code block ```
    if (trimmed.startsWith('```')) {
      const codeLines: string[] = []
      i++
      while (i < lines.length && !lineAt(i).trim().startsWith('```')) {
        codeLines.push(lineAt(i))
        i++
      }
      if (i < lines.length) i++
      blocks.push(
        <pre
          key={key++}
          className="max-w-full overflow-x-auto rounded-lg border border-border bg-secondary/60 px-3 py-2.5 font-mono text-[12px] leading-relaxed text-foreground"
        >
          <code>{codeLines.join('\n')}</code>
        </pre>,
      )
      continue
    }

    // Heading (#, ##, ###)
    const headingMatch = trimmed.match(/^(#{1,3})\s+(.+)$/)
    if (headingMatch) {
      const level = (headingMatch[1] ?? '').length
      const text = headingMatch[2] ?? ''
      const cls =
        level === 1
          ? 'text-[15px] font-semibold tracking-tight text-foreground mt-2'
          : level === 2
            ? 'text-[14px] font-semibold text-foreground mt-2'
            : 'text-[13px] font-semibold text-foreground mt-1.5'
      blocks.push(
        <p key={key++} className={cls}>
          {renderInline(text)}
        </p>,
      )
      i++
      continue
    }

    // Horizontal rule
    if (/^(-{3,}|\*{3,})$/.test(trimmed)) {
      blocks.push(<hr key={key++} className="my-2 border-border" />)
      i++
      continue
    }

    // Markdown table
    if (
      trimmed.includes('|') &&
      i + 1 < lines.length &&
      /^\s*\|?[\s:-]+\|[\s|:-]*$/.test(lineAt(i + 1))
    ) {
      const headerCells = splitTableRow(trimmed)
      i += 2
      const bodyRows: string[][] = []
      while (i < lines.length && lineAt(i).trim().includes('|') && lineAt(i).trim() !== '') {
        bodyRows.push(splitTableRow(lineAt(i)))
        i++
      }
      blocks.push(
        <div key={key++} className="max-w-full overflow-x-auto rounded-lg border border-border">
          <table className="w-full border-collapse text-left text-[12.5px]">
            <thead>
              <tr className="border-b border-border bg-secondary/50">
                {headerCells.map((cell, idx) => (
                  <th key={idx} className="px-3 py-2 font-medium text-muted-foreground">
                    {renderInline(cell)}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody className="divide-y divide-border/70">
              {bodyRows.map((row, rIdx) => (
                <tr key={rIdx}>
                  {headerCells.map((_, cIdx) => (
                    <td key={cIdx} className="tabular px-3 py-2 text-foreground">
                      {renderInline(row[cIdx] ?? '')}
                    </td>
                  ))}
                </tr>
              ))}
            </tbody>
          </table>
        </div>,
      )
      continue
    }

    // Blockquote
    if (trimmed.startsWith('>')) {
      const quoteLines: string[] = []
      while (i < lines.length && lineAt(i).trim().startsWith('>')) {
        quoteLines.push(lineAt(i).trim().replace(/^>\s?/, ''))
        i++
      }
      blocks.push(
        <blockquote
          key={key++}
          className="border-l-2 border-border pl-3 text-[13px] text-muted-foreground"
        >
          {renderInline(quoteLines.join(' '))}
        </blockquote>,
      )
      continue
    }

    // Bullet list
    if (/^[-*]\s+/.test(trimmed)) {
      const items: string[] = []
      while (i < lines.length && /^\s*[-*]\s+/.test(lineAt(i))) {
        items.push(lineAt(i).trim().replace(/^[-*]\s+/, ''))
        i++
      }
      blocks.push(
        <ul key={key++} className="list-disc space-y-1 pl-5 text-[13.5px] leading-relaxed">
          {items.map((item, idx) => (
            <li key={idx}>{renderInline(item)}</li>
          ))}
        </ul>,
      )
      continue
    }

    // Ordered list
    if (/^\d+\.\s+/.test(trimmed)) {
      const items: string[] = []
      while (i < lines.length && /^\s*\d+\.\s+/.test(lineAt(i))) {
        items.push(lineAt(i).trim().replace(/^\d+\.\s+/, ''))
        i++
      }
      blocks.push(
        <ol key={key++} className="list-decimal space-y-1 pl-5 text-[13.5px] leading-relaxed">
          {items.map((item, idx) => (
            <li key={idx}>{renderInline(item)}</li>
          ))}
        </ol>,
      )
      continue
    }

    // Paragraph (gabung baris berturut-turut yang bukan blok khusus)
    const paraLines: string[] = []
    while (i < lines.length) {
      const cur = lineAt(i).trim()
      if (
        !cur ||
        cur.startsWith('#') ||
        cur.startsWith('```') ||
        cur.startsWith('>') ||
        /^[-*]\s+/.test(cur) ||
        /^\d+\.\s+/.test(cur) ||
        /^(-{3,}|\*{3,})$/.test(cur)
      ) {
        break
      }
      paraLines.push(cur)
      i++
    }
    blocks.push(
      <p key={key++} className="text-[13.5px] leading-relaxed text-foreground">
        {renderInline(paraLines.join(' '))}
      </p>,
    )
  }

  return <div className="min-w-0 max-w-full space-y-2.5 break-words">{blocks}</div>
}

function splitTableRow(line: string): string[] {
  return line
    .trim()
    .replace(/^\|/, '')
    .replace(/\|$/, '')
    .split('|')
    .map((cell) => cell.trim())
}

function renderInline(text: string): ReactNode[] {
  const tokens: ReactNode[] = []
  const pattern = /(\*\*[^*]+\*\*|`[^`]+`|\*[^*]+\*)/g
  let lastIndex = 0
  let match: RegExpExecArray | null
  let k = 0

  while ((match = pattern.exec(text)) !== null) {
    if (match.index > lastIndex) {
      tokens.push(text.slice(lastIndex, match.index))
    }
    const token = match[0]
    if (token.startsWith('**') && token.endsWith('**')) {
      tokens.push(
        <strong key={k++} className="font-semibold text-foreground">
          {token.slice(2, -2)}
        </strong>,
      )
    } else if (token.startsWith('`') && token.endsWith('`')) {
      tokens.push(
        <code
          key={k++}
          className="rounded bg-secondary px-1.5 py-0.5 font-mono text-[12px] text-foreground"
        >
          {token.slice(1, -1)}
        </code>,
      )
    } else if (token.startsWith('*') && token.endsWith('*')) {
      tokens.push(
        <em key={k++} className="italic">
          {token.slice(1, -1)}
        </em>,
      )
    }
    lastIndex = pattern.lastIndex
  }

  if (lastIndex < text.length) {
    tokens.push(text.slice(lastIndex))
  }
  return tokens
}
