import { AlertCircle, AlertTriangle, Info } from 'lucide-react'
import { cn } from '@/lib/utils'
import { CALLOUT_LABEL, parseMarkdown, slugHeading, type CalloutTone } from './markdown-parse'
import { renderInlineBlock } from './markdown-inline'

const CALLOUT_CONFIG: Record<
  CalloutTone,
  { wrapper: string; label: string; icon: typeof Info }
> = {
  penting: {
    wrapper: 'border-zinc-800 bg-[#18181b] text-zinc-100 shadow-sm',
    label: 'text-zinc-300',
    icon: AlertCircle,
  },
  perhatian: {
    wrapper: 'border-amber-500/30 bg-amber-500/[0.06] text-amber-950 dark:text-amber-200',
    label: 'text-amber-800 dark:text-amber-300',
    icon: AlertTriangle,
  },
  catatan: {
    wrapper: 'border-zinc-200 bg-[#f7f7f8] text-[#3f3f46]',
    label: 'text-[#09090b]',
    icon: Info,
  },
}

/**
 * MarkdownView merender blok dokumen dengan gaya editorial: judul pasal yang
 * bisa ditautkan, blok peringatan, dan tabel klasifikasi yang mudah dipindai.
 */
export function MarkdownView({ source, className }: { source: string; className?: string }) {
  const blocks = parseMarkdown(source)

  return (
    <div className={cn('max-w-[56ch] text-[15px] leading-[1.8] text-[#3f3f46]', className)}>
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
                className="mt-5 rounded-r-xl border-l-[3.5px] border-[#09090b] bg-[#f8f8fa] py-3 pr-4 pl-4.5 text-[14.5px] text-[#27272a] shadow-xs"
              >
                {block.lines.map((line, lineIndex) => (
                  <p key={`${key}-${lineIndex}`} className={lineIndex === 0 ? '' : 'mt-2'}>
                    {renderInlineBlock(line, `${key}-${lineIndex}`)}
                  </p>
                ))}
              </blockquote>
            )

          case 'callout': {
            const config = CALLOUT_CONFIG[block.tone]
            const Icon = config.icon
            return (
              <aside
                key={key}
                aria-label={CALLOUT_LABEL[block.tone]}
                className={cn('mt-6 rounded-2xl border p-4.5 sm:p-5.5 transition-shadow', config.wrapper)}
              >
                <div className="flex items-center gap-2">
                  <Icon className="h-4 w-4 shrink-0" aria-hidden="true" />
                  <p
                    className={cn(
                      'text-[11px] font-semibold uppercase tracking-[0.14em]',
                      config.label,
                    )}
                  >
                    {CALLOUT_LABEL[block.tone]}
                  </p>
                </div>
                {block.title && block.title !== CALLOUT_LABEL[block.tone] && (
                  <p className="mt-2 text-[14.5px] font-semibold">
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
              <div key={key} className="mt-6 overflow-x-auto rounded-xl border border-zinc-200/90 bg-white shadow-xs">
                <table className="w-full min-w-[520px] border-collapse text-left text-[13.5px]">
                  <thead>
                    <tr className="border-b border-zinc-200/90 bg-[#f8f8fa]">
                      {block.head.map((cell, cellIndex) => (
                        <th
                          key={`${key}-h-${cellIndex}`}
                          scope="col"
                          className="px-4 py-3 text-[11px] font-semibold uppercase tracking-[0.12em] text-[#3f3f46]"
                        >
                          {renderInlineBlock(cell, `${key}-h-${cellIndex}`)}
                        </th>
                      ))}
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-zinc-100">
                    {block.rows.map((row, rowIndex) => (
                      <tr key={`${key}-r-${rowIndex}`} className="align-top hover:bg-zinc-50/60 transition-colors">
                        {row.map((cell, cellIndex) => (
                          <td
                            key={`${key}-c-${rowIndex}-${cellIndex}`}
                            className={cn(
                              'px-4 py-3',
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
