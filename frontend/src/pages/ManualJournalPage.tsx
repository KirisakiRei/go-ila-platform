import { useState } from 'react'
import { Download, Plus, Upload, Wallet } from 'lucide-react'
import { Alert, Button, Card, Input, Modal, PageHeader, Select, Skeleton } from '@/components/ui'
import { journalApi } from '@/features/manual-journal/api'
import {
  useJournalAccounts,
  useJournalEntries,
  useJournalInsights,
  useJournalMutation,
  useJournalStats,
} from '@/features/manual-journal/hooks'
import { emptyTrade, wibDay } from '@/features/manual-journal/model'
import { AccountManager } from '@/features/manual-journal/AccountManager'
import { AccountBalance } from '@/features/manual-journal/AccountBalance'
import { CSVImport } from '@/features/manual-journal/CSVImport'
import { JournalAnalytics } from '@/features/manual-journal/JournalAnalytics'
import { JournalReview } from '@/features/manual-journal/JournalReview'
import { CalendarView, JournalTable, TradeDetail } from '@/features/manual-journal/JournalViews'
import { Screenshots } from '@/features/manual-journal/Screenshots'
import { TradeForm } from '@/features/manual-journal/TradeForm'
import type { JournalFilters, JournalTrade, TradeInput } from '@/features/manual-journal/types'

export default function ManualJournalPage() {
  const accountsQuery = useJournalAccounts()
  const accounts = accountsQuery.data?.data ?? []
  const today = wibDay(new Date().toISOString())
  const [accountId, setAccountId] = useState<string | null>(null)
  const [month, setMonth] = useState(today.slice(0, 7))
  const [filters, setFilters] = useState<Omit<JournalFilters, 'account_id'>>({
    from: '',
    to: '',
    status: '',
    asset: '',
    setup: '',
    tag: '',
    emotion: '',
    mistake: '',
  })
  const activeId = accountId ?? accounts.find((a) => !a.archived)?.id ?? accounts[0]?.id ?? ''
  const active = accounts.find((a) => a.id === activeId)
  const queryFilters: JournalFilters = { ...filters, account_id: activeId }
  const validRange = !filters.from || !filters.to || filters.from <= filters.to
  const enabled = !!activeId && validRange
  const entriesQuery = useJournalEntries(queryFilters, enabled)
  const statsQuery = useJournalStats(queryFilters, enabled)
  const insightsQuery = useJournalInsights(queryFilters, enabled)
  const entries = entriesQuery.data?.data ?? []
  const [view, setView] = useState<'calendar' | 'table'>('calendar')
  const [panel, setPanel] = useState<'accounts' | 'import' | null>(null)
  const [selected, setSelected] = useState<JournalTrade | null>(null)
  const [editor, setEditor] = useState<{ id?: string; input: TradeInput } | null>(null)
  const [message, setMessage] = useState('')
  const [error, setError] = useState('')
  const [exporting, setExporting] = useState(false)
  const save = useJournalMutation(({ input, id }: { input: TradeInput; id?: string }) =>
    journalApi.saveEntry(input, id),
  )
  const remove = useJournalMutation(journalApi.deleteEntry)
  const dataError =
    accountsQuery.error || entriesQuery.error || statsQuery.error || insightsQuery.error
  function newTrade() {
    setEditor({
      input: {
        ...emptyTrade(activeId),
        checklist: (active?.checklist ?? []).map((label) => ({ label, checked: false })),
      },
    })
    setSelected(null)
  }
  return (
    <>
      <PageHeader
        title="Jurnal manual"
        description="Catat keputusan, ukur hasil, perbaiki proses. Terpisah dari posisi dan koneksi broker · seluruh waktu WIB."
        actions={
          <>
            <Button variant="outline" size="sm" onClick={() => setPanel('accounts')}>
              <Wallet className="h-4 w-4" aria-hidden="true" />
              Akun & modal
            </Button>
            <Button size="sm" disabled={!active || active.archived} onClick={newTrade}>
              <Plus className="h-4 w-4" aria-hidden="true" />
              Tambah trade
            </Button>
          </>
        }
      />
      {(error || dataError) && (
        <Alert variant="destructive">
          <div className="flex flex-wrap items-center justify-between gap-2">
            <span>{error || dataError?.message}</span>
            <Button
              variant="outline"
              size="sm"
              onClick={() => {
                setError('')
                void accountsQuery.refetch()
                if (enabled) {
                  void entriesQuery.refetch()
                  void statsQuery.refetch()
                  void insightsQuery.refetch()
                }
              }}
            >
              Coba lagi
            </Button>
          </div>
        </Alert>
      )}
      {message && (
        <p role="status" className="text-sm text-muted-foreground">
          {message}
        </p>
      )}
      {accountsQuery.isLoading ? (
        <div role="status" aria-label="Memuat akun jurnal">
          <Skeleton className="h-28 w-full" />
        </div>
      ) : !accounts.length && !accountsQuery.isError ? (
        <Card className="px-5 py-12 text-center">
          <h2 className="text-lg font-semibold">Mulai dari akun jurnalmu</h2>
          <p className="mx-auto mt-2 max-w-lg text-sm text-muted-foreground">
            Tentukan nama akun, mata uang, dan modal awal. Tidak perlu menghubungkan broker atau
            menyalakan bot.
          </p>
          <Button className="mt-5" onClick={() => setPanel('accounts')}>
            Buat akun jurnal
          </Button>
        </Card>
      ) : accounts.length > 0 ? (
        <>
          {active && <AccountBalance account={active} />}
          <section className="space-y-4" aria-label="Filter jurnal">
            <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
              <label className="space-y-1 text-xs text-muted-foreground">
                Akun jurnal
                <Select
                  value={activeId}
                  onChange={(e) => {
                    setAccountId(e.target.value)
                    setSelected(null)
                    setMessage('')
                  }}
                >
                  {accounts.map((a) => (
                    <option key={a.id} value={a.id}>
                      {a.name} · {a.currency}
                      {a.archived ? ' (arsip)' : ''}
                    </option>
                  ))}
                </Select>
              </label>
              <label className="space-y-1 text-xs text-muted-foreground">
                Dari tanggal (WIB)
                <Input
                  type="date"
                  value={filters.from}
                  onChange={(e) => setFilters({ ...filters, from: e.target.value })}
                />
              </label>
              <label className="space-y-1 text-xs text-muted-foreground">
                Sampai tanggal (WIB)
                <Input
                  type="date"
                  value={filters.to}
                  onChange={(e) => setFilters({ ...filters, to: e.target.value })}
                />
              </label>
              <label className="space-y-1 text-xs text-muted-foreground">
                Status
                <Select
                  value={filters.status}
                  onChange={(e) =>
                    setFilters({ ...filters, status: e.target.value as JournalFilters['status'] })
                  }
                >
                  <option value="">Semua status</option>
                  <option value="draft">Draft</option>
                  <option value="open">Open</option>
                  <option value="closed">Closed</option>
                </Select>
              </label>
            </div>
            <details className="text-sm">
              <summary className="w-fit cursor-pointer rounded text-muted-foreground focus-visible:ring-2 focus-visible:ring-ring">
                Filter aset, setup, tag, emosi & kesalahan
              </summary>
              <div className="mt-3 grid gap-3 sm:grid-cols-3">
                {(
                  [
                    ['asset', 'Aset'],
                    ['setup', 'Setup'],
                    ['tag', 'Tag'],
                    ['emotion', 'Emosi'],
                    ['mistake', 'Kesalahan'],
                  ] as const
                ).map(([key, label]) => (
                  <label key={key} className="space-y-1 text-xs text-muted-foreground">
                    {label}
                    <Input
                      maxLength={100}
                      value={filters[key]}
                      onChange={(e) => setFilters({ ...filters, [key]: e.target.value })}
                    />
                  </label>
                ))}
              </div>
            </details>
            {!validRange && (
              <Alert variant="warning">Tanggal akhir harus sama atau setelah tanggal awal.</Alert>
            )}
            {active?.archived && (
              <Alert variant="info">
                Akun diarsipkan. Pulihkan lewat Akun & modal untuk mencatat trade baru.
              </Alert>
            )}
          </section>
          {!!activeId && <JournalReview key={activeId} accountId={activeId} />}
          {enabled && (
            <>
              {statsQuery.isLoading ? (
                <div role="status" aria-label="Memuat statistik jurnal">
                  <Skeleton className="h-32 w-full" />
                </div>
              ) : (
                statsQuery.data && (
                  <JournalAnalytics
                    stats={{
                      ...statsQuery.data,
                      currency: statsQuery.data.currency || active?.currency || '',
                    }}
                    insights={insightsQuery.data ?? []}
                  />
                )
              )}
              <section className="space-y-4" aria-label="Catatan trade">
                <div className="flex flex-wrap items-center justify-between gap-3">
                  <div className="flex gap-1" aria-label="Tampilan jurnal">
                    <Button
                      size="sm"
                      variant={view === 'calendar' ? 'secondary' : 'ghost'}
                      aria-pressed={view === 'calendar'}
                      onClick={() => setView('calendar')}
                    >
                      Kalender
                    </Button>
                    <Button
                      size="sm"
                      variant={view === 'table' ? 'secondary' : 'ghost'}
                      aria-pressed={view === 'table'}
                      onClick={() => setView('table')}
                    >
                      Tabel
                    </Button>
                  </div>
                  <div className="flex flex-wrap gap-2">
                    <Button variant="outline" size="sm" onClick={() => setPanel('import')}>
                      <Upload className="h-4 w-4" aria-hidden="true" />
                      Impor CSV
                    </Button>
                    <Button
                      variant="outline"
                      size="sm"
                      isLoading={exporting}
                      onClick={async () => {
                        setExporting(true)
                        setError('')
                        try {
                          await journalApi.export(queryFilters)
                          setMessage('CSV jurnal berhasil diekspor sesuai filter aktif.')
                        } catch (err) {
                          setError(err instanceof Error ? err.message : 'Ekspor gagal.')
                        } finally {
                          setExporting(false)
                        }
                      }}
                    >
                      <Download className="h-4 w-4" aria-hidden="true" />
                      Ekspor CSV
                    </Button>
                  </div>
                </div>
                <p className="text-xs text-muted-foreground">
                  Ekspor CSV maksimal 500 baris / 1 MiB; persempit filter jika melebihi batas.
                  Kalender, tabel, KPI, dan ekspor memakai filter yang sama. Klik tanggal untuk
                  rekap harian. Angka kalender dalam {active?.currency}.
                </p>
                {entriesQuery.isLoading ? (
                  <div role="status" aria-label="Memuat trade">
                    <Skeleton className="h-72 w-full" />
                  </div>
                ) : (
                  !entriesQuery.isError && (
                    <>
                      {!entries.length && (
                        <div className="rounded-lg border border-dashed border-border px-4 py-8 text-center">
                          <h2 className="text-sm font-semibold">Belum ada trade pada filter ini</h2>
                          <p className="mt-1 text-xs text-muted-foreground">
                            Ubah filter atau tambahkan trade pertama. Draft juga bisa disimpan.
                          </p>
                        </div>
                      )}
                      {view === 'calendar' ? (
                        <CalendarView
                          trades={entries}
                          accounts={accounts}
                          month={month}
                          onMonth={setMonth}
                          onSelect={setSelected}
                          calendar={statsQuery.data?.calendar ?? {}}
                        />
                      ) : (
                        <JournalTable
                          key={JSON.stringify(queryFilters)}
                          trades={entries}
                          accounts={accounts}
                          onSelect={setSelected}
                        />
                      )}
                    </>
                  )
                )}
              </section>
            </>
          )}
        </>
      ) : null}
      <Modal
        open={panel !== null}
        onClose={() => setPanel(null)}
        title={panel === 'accounts' ? 'Akun jurnal & modal' : 'Impor CSV jurnal'}
        size="2xl"
      >
        {panel === 'accounts' ? (
          <AccountManager accounts={accounts} />
        ) : panel === 'import' ? (
          <CSVImport
            onDone={() => {
              setPanel(null)
              setMessage('Impor CSV selesai. Data jurnal sudah diperbarui.')
            }}
          />
        ) : null}
      </Modal>
      <Modal
        open={!!editor}
        onClose={() => {
          if (!save.isPending) setEditor(null)
        }}
        title={editor?.id ? 'Edit trade jurnal' : 'Tambah trade jurnal'}
        size="2xl"
      >
        {editor && (
          <TradeForm
            key={editor.id ?? 'new'}
            initial={editor.input}
            editing={!!editor.id}
            accounts={accounts}
            pending={save.isPending}
            onCancel={() => setEditor(null)}
            onSave={async (input) => {
              const trade = await save.mutateAsync({ input, id: editor.id })
              setEditor(null)
              setSelected(trade)
              setMessage('Trade berhasil disimpan.')
            }}
          />
        )}
      </Modal>
      <Modal
        open={!!selected}
        onClose={() => {
          if (!remove.isPending) setSelected(null)
        }}
        title="Detail trade"
        size="2xl"
      >
        {selected && (
          <div className="space-y-5">
            <TradeDetail
              trade={selected}
              account={accounts.find((a) => a.id === selected.account_id)}
              deleting={remove.isPending}
              onEdit={() => {
                setEditor({ id: selected.id, input: selected })
                setSelected(null)
              }}
              onDelete={async () => {
                if (!window.confirm('Hapus trade ini secara permanen beserta screenshot-nya?'))
                  return
                try {
                  await remove.mutateAsync(selected.id)
                  setSelected(null)
                  setMessage('Trade dihapus.')
                } catch (err) {
                  setError(err instanceof Error ? err.message : 'Gagal menghapus trade.')
                }
              }}
            />
            <Screenshots
              readOnly={accounts.find((a) => a.id === selected.account_id)?.archived}
              entryId={selected.id}
            />
          </div>
        )}
      </Modal>
    </>
  )
}
