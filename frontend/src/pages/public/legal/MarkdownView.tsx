import { cn } from '@/lib/utils'
import { CALLOUT_LABEL, parseMarkdown, slugHeading, type CalloutTone } from './markdown-parse'
import { renderInlineBlock } from './markdown-inline'

const CALLOUT_STYLE: Record<CalloutTone, { wrapper: string; label: string; dot: string }> = {
  penting: {
    wrapper: 'border-[#09090b]/90 bg-[#09090b] text-zinc-200',
    label: 'text-white',
    dot: 'bg-white',
  },
  perhatian: {
    wrapper: 'border-zinc-300 bg-[#f4f4f5] text-[#27272a]',
    label: 'text-[#09090b]',
    dot: 'bg-[#09090b]',
  },
  catatan: {
    wrapper: 'border-zinc-200 bg-transparent text-[#3f3f46]',
    label: 'text-[#09090b]',
    dot: 'bg-zinc-400',
  },
}

/**
 * MarkdownView merender blok dokumen dengan gaya editorial: judul pasal yang
 * bisa ditautkan, blok peringatan, dan tabel klasifikasi yang mudah dipindai.
 */
export function MarkdownView({ source, className }: { source: string; className?: string }) {
  const blocks = parseMarkdown(source)

  return (
    <div className={cn('text-[15px] leading-[1.8] text-[#3f3f46]', className)}>
      {blocks.map((block, index) => {
        const key = `blok-${index}`

        switch (block.kind) {
          case 'heading': {
            if (block.level === 1) return null

            if (block.level === 2) {
              return (
                <h2
                  key={key}
                  id={slugHeading(block.text)}
                  className="mt-12 scroll-mt-28 border-t border-zinc-200 pt-8 text-[19px] font-semibold tracking-tight text-[#09090b] first:mt-0 first:border-0 first:pt-0 sm:text-[21px]"
                >
                  {renderInlineBlock(block.text, key)}
                </h2>
              )
            }

            return (
              <h3
                key={key}
                className="mt-7 scroll-mt-28 text-[15px] font-semibold tracking-tight text-[#09090b]"
              >
                {renderInlineBlock(block.text, key)}
              </h3>
            )
          }

          case 'paragraph':
            return (
              <p key={key} className="mt-4">
                {renderInlineBlock(block.text, key)}
              </p>
            )

          case 'list': {
            const ListTag = block.ordered ? 'ol' : 'ul'
            return (
              <ListTag
                key={key}
                className={cn(
                  'mt-4 space-y-2 pl-5 marker:text-zinc-400',
                  block.ordered ? 'list-decimal' : 'list-disc',
                )}
              >
                {block.items.map((item, itemIndex) => (
                  <li key={`${key}-${itemIndex}`} className="pl-1">
                    {renderInlineBlock(item, `${key}-${itemIndex}`)}
                  </li>
                ))}
              </ListTag>
            )
          }

          case 'quote':
            return (
              <blockquote
                key={key}
                className="mt-5 border-l-2 border-zinc-300 pl-4 text-[15px] italic text-[#27272a]"
              >
                {block.lines.map((line, lineIndex) => (
                  <p key={`${key}-${lineIndex}`} className={lineIndex === 0 ? '' : 'mt-2'}>
                    {renderInlineBlock(line, `${key}-${lineIndex}`)}
                  </p>
                ))}
              </blockquote>
            )

          case 'callout': {
            const style = CALLOUT_STYLE[block.tone]
            return (
              <aside
                key={key}
                aria-label={CALLOUT_LABEL[block.tone]}
                className={cn('mt-6 rounded-xl border p-5 sm:p-6', style.wrapper)}
              >
                <p
                  className={cn(
                    'flex items-center gap-2 text-[11px] font-semibold uppercase tracking-[0.16em]',
                    style.label,
                  )}
                >
                  <span className={cn('h-1.5 w-1.5 rounded-full', style.dot)} aria-hidden="true" />
                  {CALLOUT_LABEL[block.tone]}
                </p>
                {block.title && block.title !== CALLOUT_LABEL[block.tone] && (
                  <p className="mt-2 text-[14px] font-semibold">
                    {renderInlineBlock(block.title, `${key}-judul`)}
                  </p>
                )}
                <div className="mt-2 space-y-2 text-[14px] leading-relaxed">
                  {block.lines.map((line, lineIndex) => (
                    <p key={`${key}-${lineIndex}`}>
                      {renderInlineBlock(line, `${key}-${lineIndex}`)}
                    </p>
                  ))}
                </div>
              </aside>
            )
          }

          case 'table':
            return (
              <div key={key} className="mt-6 overflow-x-auto rounded-xl border border-zinc-200">
                <table className="w-full min-w-[520px] border-collapse text-left text-[14px]">
                  <thead>
                    <tr className="bg-[#f4f4f5]">
                      {block.head.map((cell, cellIndex) => (
                        <th
                          key={`${key}-h-${cellIndex}`}
                          scope="col"
                          className="border-b border-zinc-200 px-4 py-3 text-[11px] font-semibold uppercase tracking-[0.12em] text-[#3f3f46]"
                        >
                          {renderInlineBlock(cell, `${key}-h-${cellIndex}`)}
                        </th>
                      ))}
                    </tr>
                  </thead>
                  <tbody>
                    {block.rows.map((row, rowIndex) => (
                      <tr key={`${key}-r-${rowIndex}`} className="align-top even:bg-[#fafafa]">
                        {row.map((cell, cellIndex) => (
                          <td
                            key={`${key}-c-${rowIndex}-${cellIndex}`}
                            className={cn(
                              'border-b border-zinc-100 px-4 py-3',
                              cellIndex === 0 && 'font-medium text-[#09090b]',
                            )}
                          >
                            {renderInlineBlock(cell, `${key}-${rowIndex}-${cellIndex}`)}
                          </td>
                        ))}
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )

          case 'rule':
            return <hr key={key} className="mt-10 border-zinc-200" />

          default:
            return null
        }
      })}
    </div>
  )
}
