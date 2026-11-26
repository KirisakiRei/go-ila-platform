import { useMemo, useState } from 'react'
import { Plus, RotateCcw, Search, Settings, SlidersHorizontal } from 'lucide-react'
import { useQueryClient } from '@tanstack/react-query'
import { Alert, Button, Card, Input, Modal, PageHeader, Select, Skeleton } from '@/components/ui'
import { journalApi } from '@/features/manual-journal/api'
import {
  journalKey,
  useJournalAccounts,
  useJournalEntries,
  useJournalInsights,
  useJournalMutation,
  useJournalStats,
} from '@/features/manual-journal/hooks'
import { emptyTrade, wibDay } from '@/features/manual-journal/model'
import { AccountBalance } from '@/features/manual-journal/AccountBalance'
import { JournalAnalytics, JournalKpiStrip } from '@/features/manual-journal/JournalAnalytics'
import { JournalReview } from '@/features/manual-journal/JournalReview'
import {
  JournalSettingsModal,
  type SettingsTab,
} from '@/features/manual-journal/JournalSettingsModal'
import { CalendarView, JournalTable, TradeDetail } from '@/features/manual-journal/JournalViews'
import { Screenshots } from '@/features/manual-journal/Screenshots'
import { TradeForm } from '@/features/manual-journal/TradeForm'
import { SegmentedControl } from '@/features/manual-journal/ui'
import type { JournalFilters, TradeInput } from '@/features/manual-journal/types'

type PeriodPreset = 'all' | 'month' | '30d' | 'custom'

const WORKSPACE_TABS = [
  { value: 'calendar', label: 'Kalender' },
  { value: 'table', label: 'Tabel' },
  { value: 'analytics', label: 'Analitik' },
] as const

function shiftDaysWib(days: number) {
  return wibDay(new Date(Date.now() + days * 86400000).toISOString())
}

function monthBounds(ym: string) {
  const y = Number(ym.slice(0, 4))
  const m = Number(ym.slice(5, 7))
  const last = new Date(Date.UTC(y, m, 0)).getUTCDate()
  return { from: `${ym}-01`, to: `${ym}-${String(last).padStart(2, '0')}` }
}

const EMPTY_FILTERS: Omit<JournalFilters, 'account_id'> = {
  search: '',
  from: '',
  to: '',
  status: '',
  asset: '',
  setup: '',
  tag: '',
  emotion: '',
  mistake: '',
}

export default function ManualJournalPage() {
  const client = useQueryClient()
  const accountsQuery = useJournalAccounts()
  const accounts = accountsQuery.data?.data ?? []
  const today = wibDay(new Date().toISOString())

  const [accountId, setAccountId] = useState<string | null>(null)
  const [month, setMonth] = useState(today.slice(0, 7))
  const [period, setPeriod] = useState<PeriodPreset>('all')
  const [filters, setFilters] = useState<Omit<JournalFilters, 'account_id'>>(EMPTY_FILTERS)
  const [filtersOpen, setFiltersOpen] = useState(false)

  const active =
    accounts.find((a) => a.id === accountId) ??
    accounts.find((a) => !a.archived) ??
    accounts[0]
  const activeId = active?.id ?? ''
  const queryFilters: JournalFilters = { ...filters, account_id: activeId }
  const validRange = !filters.from || !filters.to || filters.from <= filters.to
  const enabled = !!activeId && validRange

  const entriesQuery = useJournalEntries(queryFilters, enabled)
  const statsQuery = useJournalStats(queryFilters, enabled)
  const insightsQuery = useJournalInsights(queryFilters, enabled)
  const entries = useMemo(() => entriesQuery.data?.data ?? [], [entriesQuery.data])

  const recentAssets = useMemo(() => {
    const seen = new Set<string>()
    for (const item of entries) {
      if (item.asset) seen.add(item.asset)
      if (seen.size >= 6) break
    }
    return [...seen]
  }, [entries])

  const [view, setView] = useState<'calendar' | 'table' | 'analytics'>('calendar')
  const [settingsModal, setSettingsModal] = useState<{ open: boolean; tab: SettingsTab }>({
    open: false,
    tab: 'accounts',
  })
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const selected = useMemo(
    () => (selectedId ? entries.find((e) => e.id === selectedId) ?? null : null),
    [entries, selectedId],
  )
  const [editor, setEditor] = useState<{ id?: string; input: TradeInput } | null>(null)
  const [saving, setSaving] = useState(false)
  const [message, setMessage] = useState('')
  const [error, setError] = useState('')

  const save = useJournalMutation(({ input, id }: { input: TradeInput; id?: string }) =>
    journalApi.saveEntry(input, id),
  )
  const remove = useJournalMutation(journalApi.deleteEntry)
  const dataError =
    accountsQuery.error || entriesQuery.error || statsQuery.error || insightsQuery.error

  const activeFilterCount = [
    filters.status,
    filters.asset,
    filters.setup,
    filters.tag,
    filters.emotion,
    filters.mistake,
  ].filter(Boolean).length

  function applyPeriod(next: PeriodPreset) {
    setPeriod(next)
    if (next === 'all') {
      setFilters((f) => ({ ...f, from: '', to: '' }))
    } else if (next === 'month') {
      const b = monthBounds(month)
      setFilters((f) => ({ ...f, ...b }))
    } else if (next === '30d') {
      setFilters((f) => ({ ...f, from: shiftDaysWib(-29), to: today }))
    }
  }

  function handleCalendarMonth(nextMonth: string) {
    setMonth(nextMonth)
    if (period === 'month') {
      setFilters((f) => ({ ...f, ...monthBounds(nextMonth) }))
    }
  }

  function newTrade() {
    setEditor({
      input: {
        ...emptyTrade(activeId),
        status: 'closed',
        opened_at: new Date().toISOString(),
        closed_at: new Date().toISOString(),
        size_unit: 'lot',
        checklist: (active?.checklist ?? []).map((label) => ({ label, checked: false })),
      },
    })
    setSelectedId(null)
  }

  function openSettings(tab: SettingsTab = 'accounts') {
    setSettingsModal({ open: true, tab })
  }

  return (
    <div className="space-y-4">
      <PageHeader
        title="Jurnal"
        description="Catatan trade manual & kalender P&L · WIB"
        actions={
          <div className="flex flex-wrap items-center gap-2">
            <Button
              variant="outline"
              size="sm"
              onClick={() => openSettings('accounts')}
              aria-label="Pengaturan jurnal"
            >
              <Settings className="h-4 w-4" aria-hidden="true" />
              Pengaturan
            </Button>
            <Button size="sm" disabled={!active || active.archived} onClick={newTrade}>
              <Plus className="h-4 w-4" aria-hidden="true" />
              Catat trade
            </Button>
          </div>
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
        <div
          role="status"
          className="flex items-center justify-between gap-2 rounded-lg border border-border bg-secondary/40 px-3.5 py-2 text-xs text-foreground"
        >
          <span>{message}</span>
          <button
            type="button"
            onClick={() => setMessage('')}
            className="text-muted-foreground hover:text-foreground"
          >
            Tutup
          </button>
        </div>
      )}

      {accountsQuery.isLoading ? (
        <div role="status" aria-label="Memuat daftar jurnal" className="space-y-3">
          <Skeleton className="h-20 w-full rounded-xl" />
          <Skeleton className="h-64 w-full rounded-xl" />
        </div>
      ) : !accounts.length && !accountsQuery.isError ? (
        <Card className="px-6 py-12 text-center">
          <h2 className="text-base font-semibold">Buat jurnal pertamamu</h2>
          <p className="mx-auto mt-1.5 max-w-md text-sm text-muted-foreground">
            Isi nama jurnal, mata uang, dan modal awal.
          </p>
          <Button className="mt-5" onClick={() => openSettings('accounts')}>
            <Plus className="h-4 w-4" aria-hidden="true" />
            Tambah jurnal
          </Button>
        </Card>
      ) : accounts.length > 0 ? (
        <>
          {/* Bar identitas jurnal ringkas */}
          <div className="flex flex-wrap items-center justify-between gap-2.5">
            <div className="flex flex-wrap items-center gap-2">
              <Select
                aria-label="Pilih jurnal"
                className="h-9 w-auto min-w-[180px] text-xs font-medium sm:text-sm"
                value={activeId}
                onChange={(e) => {
                  setAccountId(e.target.value)
                  setSelectedId(null)
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
              {active && <AccountBalance account={active} compact />}
            </div>

            {active?.archived && (
              <span className="text-xs text-muted-foreground">
                Jurnal diarsipkan · pulihkan di Pengaturan untuk mencatat trade
              </span>
            )}
          </div>

          {/* 4 KPI inti dengan hover tooltip */}
          {enabled &&
            (statsQuery.isLoading ? (
              <div role="status" aria-label="Memuat statistik jurnal">
                <Skeleton className="h-24 w-full rounded-xl" />
              </div>
            ) : (
              statsQuery.data && (
                <JournalKpiStrip
                  stats={{
                    ...statsQuery.data,
                    currency: statsQuery.data.currency || active?.currency || '',
                  }}
                />
              )
            ))}

          {/* Insight AI / Aturan ringkas 1 baris ala info tip */}
          {!!activeId && (
            <JournalReview
              key={activeId}
              accountId={activeId}
              onOpenSettings={() => openSettings('ai')}
            />
          )}

          {/* Toolbar ruang kerja: Tabs + Search + Periode + Filter */}
          <section className="space-y-3" aria-label="Ruang kerja jurnal">
            <div className="flex flex-col gap-2.5 lg:flex-row lg:items-center lg:justify-between">
              <SegmentedControl
                ariaLabel="Tampilan jurnal"
                value={view}
                onChange={setView}
                options={WORKSPACE_TABS}
                className="w-full sm:w-auto"
              />

              <div className="flex flex-wrap items-center gap-2">
                <div className="relative min-w-[170px] flex-1 sm:w-52 sm:flex-none">
                  <Search
                    className="pointer-events-none absolute left-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-muted-foreground"
                    aria-hidden="true"
                  />
                  <Input
                    aria-label="Cari trade"
                    value={filters.search ?? ''}
                    onChange={(e) => setFilters({ ...filters, search: e.target.value })}
                    placeholder="Cari pair, setup, catatan"
                    className="h-9 pl-8 text-xs sm:text-sm"
                  />
                </div>

                <Select
                  aria-label="Periode jurnal"
                  value={period}
                  onChange={(e) => applyPeriod(e.target.value as PeriodPreset)}
                  className="h-9 w-auto text-xs sm:text-sm"
                >
                  <option value="all">Semua waktu</option>
                  <option value="month">Bulan terpilih</option>
                  <option value="30d">30 hari terakhir</option>
                  <option value="custom">Rentang tanggal</option>
                </Select>

                <Button
                  variant={filtersOpen || activeFilterCount > 0 ? 'secondary' : 'outline'}
                  size="sm"
                  className="h-9 gap-1.5"
                  aria-expanded={filtersOpen}
                  onClick={() => setFiltersOpen((v) => !v)}
                >
                  <SlidersHorizontal className="h-3.5 w-3.5" aria-hidden="true" />
                  Filter
                  {activeFilterCount > 0 && (
                    <span className="rounded bg-foreground px-1.5 py-0.5 text-[10px] font-semibold tabular-nums text-background">
                      {activeFilterCount}
                    </span>
                  )}
                </Button>
              </div>
            </div>

            {/* Rentang tanggal kustom (muncul hanya saat dipilih) */}
            {period === 'custom' && (
              <div className="grid gap-3 rounded-xl border border-border/80 bg-secondary/20 p-3 sm:grid-cols-2">
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
              </div>
            )}

            {/* Panel filter lanjutan (tersembunyi secara default agar halaman lega) */}
            {filtersOpen && (
              <div className="space-y-3 rounded-xl border border-border/80 bg-secondary/20 p-3.5">
                <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
                  <label className="space-y-1 text-xs text-muted-foreground">
                    Status
                    <Select
                      value={filters.status}
                      onChange={(e) =>
                        setFilters({
                          ...filters,
                          status: e.target.value as JournalFilters['status'],
                        })
                      }
                    >
                      <option value="">Semua status</option>
                      <option value="closed">Closed</option>
                      <option value="open">Open</option>
                      <option value="draft">Draft</option>
                    </Select>
                  </label>
                  {(
                    [
                      ['asset', 'Pair'],
                      ['setup', 'Setup'],
                      ['tag', 'Tag'],
                      ['emotion', 'Emosi'],
                      ['mistake', 'Mistake'],
                    ] as const
                  ).map(([key, label]) => (
                    <label key={key} className="space-y-1 text-xs text-muted-foreground">
                      {label}
                      <Input
                        maxLength={100}
                        value={filters[key]}
                        onChange={(e) => setFilters({ ...filters, [key]: e.target.value })}
                        placeholder={`Filter ${label.toLowerCase()}`}
                      />
                    </label>
                  ))}
                </div>
                {activeFilterCount > 0 && (
                  <div className="flex justify-end">
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() =>
                        setFilters((f) => ({
                          ...EMPTY_FILTERS,
                          from: f.from,
                          to: f.to,
                          search: f.search,
                        }))
                      }
                    >
                      <RotateCcw className="h-3.5 w-3.5" aria-hidden="true" />
                      Reset filter
                    </Button>
                  </div>
                )}
              </div>
            )}

            {!validRange && (
              <Alert variant="warning">Tanggal akhir harus sama atau setelah tanggal awal.</Alert>
            )}

            {/* Isi Tab Aktif */}
            {enabled && (
              <>
                {view === 'analytics' ? (
                  statsQuery.isLoading ? (
                    <Skeleton className="h-72 w-full rounded-xl" />
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
                  )
                ) : entriesQuery.isLoading ? (
                  <div role="status" aria-label="Memuat trade">
                    <Skeleton className="h-72 w-full rounded-xl" />
                  </div>
                ) : (
                  !entriesQuery.isError && (
                    <>
                      {!entries.length && view === 'table' ? (
                        <div className="rounded-xl border border-dashed border-border px-4 py-10 text-center">
                          <h2 className="text-sm font-semibold">Belum ada trade pada filter ini</h2>
                          <p className="mt-1 text-xs text-muted-foreground">
                            Ubah filter atau catat trade pertamamu.
                          </p>
                          <Button
                            size="sm"
                            className="mt-4"
                            disabled={!active || active.archived}
                            onClick={newTrade}
                          >
                            <Plus className="h-4 w-4" aria-hidden="true" />
                            Catat trade
                          </Button>
                        </div>
                      ) : view === 'calendar' ? (
                        <CalendarView
                          key={activeId}
                          trades={entries}
                          month={month}
                          accounts={accounts}
                          currency={active?.currency}
                          calendar={statsQuery.data?.calendar ?? {}}
                          onMonth={handleCalendarMonth}
                          onSelect={(trade) => setSelectedId(trade.id)}
                        />
                      ) : (
                        <JournalTable
                          key={JSON.stringify(queryFilters)}
                          trades={entries}
                          accounts={accounts}
                          onSelect={(trade) => setSelectedId(trade.id)}
                        />
                      )}
                    </>
                  )
                )}
              </>
            )}
          </section>
        </>
      ) : null}

      {/* 1 Modal Pengaturan Terpadu */}
      <JournalSettingsModal
        key={settingsModal.open ? settingsModal.tab : 'closed'}
        open={settingsModal.open}
        initialTab={settingsModal.tab}
        accounts={accounts}
        activeAccountId={activeId}
        exportFilters={queryFilters}
        onClose={() => setSettingsModal((s) => ({ ...s, open: false }))}
        onImportDone={() => {
          setSettingsModal((s) => ({ ...s, open: false }))
          setMessage('Impor CSV selesai. Data jurnal sudah diperbarui.')
        }}
        onExportSuccess={() => {
          setMessage('CSV jurnal berhasil diunduh sesuai filter aktif.')
        }}
      />

      {/* Modal Tambah / Edit Trade */}
      <Modal
        open={!!editor}
        onClose={() => {
          if (!saving) setEditor(null)
        }}
        title={editor?.id ? 'Edit trade' : 'Catat trade'}
        size="2xl"
      >
        {editor && (
          <TradeForm
            key={editor.id ?? 'new'}
            initial={editor.input}
            editing={!!editor.id}
            entryId={editor.id}
            accounts={accounts}
            recentAssets={recentAssets}
            pending={saving}
            onCancel={() => {
              if (!saving) setEditor(null)
            }}
            onSave={async (input, queuedFiles) => {
              setSaving(true)
              try {
                const saved = await save.mutateAsync({ input, id: editor.id })
                if (!editor.id && queuedFiles && queuedFiles.length > 0) {
                  const failed: string[] = []
                  for (const file of queuedFiles) {
                    try {
                      await journalApi.uploadScreenshot(saved.id, file)
                    } catch {
                      failed.push(file.name)
                    }
                  }
                  await client.invalidateQueries({ queryKey: journalKey })
                  if (failed.length > 0) {
                    setError(
                      `Trade tersimpan, tetapi ${failed.length} screenshot gagal diunggah (${failed.join(', ')}).`,
                    )
                  }
                }
                setEditor(null)
                setMessage(editor.id ? 'Trade diperbarui.' : 'Trade tersimpan ke jurnal.')
              } finally {
                setSaving(false)
              }
            }}
          />
        )}
      </Modal>

      {/* Modal Detail Trade */}
      <Modal
        open={!!selected}
        onClose={() => {
          if (!remove.isPending) setSelectedId(null)
        }}
        title="Detail trade"
        size="xl"
      >
        {selected && (
          <TradeDetail
            trade={selected}
            account={accounts.find((a) => a.id === selected.account_id)}
            deleting={remove.isPending}
            onEdit={() => {
              setEditor({ id: selected.id, input: selected })
              setSelectedId(null)
            }}
            onDelete={async () => {
              setError('')
              try {
                await remove.mutateAsync(selected.id)
                setSelectedId(null)
                setMessage('Trade dihapus.')
              } catch (err) {
                setError(err instanceof Error ? err.message : 'Gagal menghapus trade.')
              }
            }}
          >
            <Screenshots
              entryId={selected.id}
              readOnly={accounts.find((a) => a.id === selected.account_id)?.archived}
            />
          </TradeDetail>
        )}
      </Modal>
    </div>
  )
}
