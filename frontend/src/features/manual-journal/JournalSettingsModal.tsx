import { useState } from 'react'
import { Download } from 'lucide-react'
import { useIsMutating } from '@tanstack/react-query'
import { Alert, Button, Modal } from '@/components/ui'
import { AccountsPanel, CapitalPanel } from './AccountManager'
import { journalApi } from './api'
import { CSVImport } from './CSVImport'
import { journalKey } from './hooks'
import { JournalAIPreferences } from './JournalReview'
import type { JournalAccount, JournalFilters } from './types'
import { SegmentedControl } from './ui'

export type SettingsTab = 'accounts' | 'capital' | 'ai' | 'csv'

const TABS = [
  { value: 'accounts', label: 'Akun' },
  { value: 'capital', label: 'Modal' },
  { value: 'ai', label: 'AI & Privasi' },
  { value: 'csv', label: 'CSV' },
] as const

export function JournalSettingsModal({
  open,
  initialTab = 'accounts',
  accounts,
  activeAccountId,
  exportFilters,
  onClose,
  onImportDone,
  onExportSuccess,
}: {
  open: boolean
  initialTab?: SettingsTab
  accounts: JournalAccount[]
  activeAccountId: string
  exportFilters: JournalFilters
  onClose: () => void
  onImportDone: () => void
  onExportSuccess: () => void
}) {
  const mutating = useIsMutating({ mutationKey: journalKey }) > 0
  const [tab, setTab] = useState<SettingsTab>(initialTab)
  const [capitalAccountOverride, setCapitalAccountOverride] = useState<string | null>(null)
  const [exporting, setExporting] = useState(false)
  const [exportError, setExportError] = useState('')
  const busy = mutating || exporting

  return (
    <Modal
      open={open}
      onClose={() => {
        if (!busy) onClose()
      }}
      title="Pengaturan jurnal"
      description="Kelola akun, modal, preferensi insight AI, serta impor/ekspor CSV."
      size="2xl"
    >
      <div className="space-y-5">
        <div className="overflow-x-auto pb-0.5 scrollbar-none">
          <SegmentedControl
            ariaLabel="Tab pengaturan jurnal"
            value={tab}
            onChange={(next) => {
              setCapitalAccountOverride(null)
              setTab(next)
            }}
            options={TABS}
            disabled={busy}
            fullWidth
          />
        </div>

        {tab === 'accounts' && (
          <AccountsPanel
            accounts={accounts}
            onSelectCapital={(id) => {
              setCapitalAccountOverride(id)
              setTab('capital')
            }}
          />
        )}

        {tab === 'capital' && (
          <CapitalPanel
            accounts={accounts}
            initialAccountId={capitalAccountOverride ?? activeAccountId}
          />
        )}

        {tab === 'ai' && <JournalAIPreferences accountId={activeAccountId || undefined} />}

        {tab === 'csv' && (
          <div className="space-y-5">
            <div className="flex flex-wrap items-center justify-between gap-3 rounded-xl border border-border/80 bg-secondary/25 p-3.5">
              <div className="min-w-0">
                <p className="text-sm font-semibold">Ekspor CSV</p>
                <p className="mt-0.5 text-xs text-muted-foreground">
                  Mengikuti filter aktif di halaman utama · maks 500 baris / 1 MiB.
                </p>
              </div>
              <Button
                variant="outline"
                size="sm"
                disabled={!activeAccountId}
                isLoading={exporting}
                onClick={async () => {
                  setExporting(true)
                  setExportError('')
                  try {
                    await journalApi.export(exportFilters)
                    onExportSuccess()
                  } catch (err) {
                    setExportError(err instanceof Error ? err.message : 'Ekspor gagal.')
                  } finally {
                    setExporting(false)
                  }
                }}
              >
                <Download className="h-3.5 w-3.5" aria-hidden="true" />
                Unduh CSV
              </Button>
            </div>
            {exportError && <Alert variant="destructive">{exportError}</Alert>}
            <div className="border-t border-border pt-4">
              <h3 className="mb-2 text-sm font-semibold">Impor CSV</h3>
              <CSVImport accounts={accounts} onDone={onImportDone} />
            </div>
          </div>
        )}
      </div>
    </Modal>
  )
}
