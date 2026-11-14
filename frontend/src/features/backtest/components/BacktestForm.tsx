/**
 * BacktestForm menyusun satu backtest: pair, pasangan timeframe, rentang, dan
 * modal.
 *
 * Dua keputusan membentuk seluruh isi komponen ini:
 *
 * 1. Pemilih tanggal dikunci ke cakupan data yang benar-benar tersedia. Rentang
 *    yang mustahil dipenuhi tidak pernah ditawarkan, karena pengguna yang
 *    memilihnya hanya akan menerima galat setelah menunggu.
 * 2. Pasangan timeframe tampil sebagai tombol, bukan dropdown. Hanya ada empat
 *    pilihan, dan cakupannya berbeda-beda sehingga perlu terlihat sekaligus.
 *
 * State di sini sengaja hanya menyimpan PILIHAN pengguna. Rentang tanggal
 * diturunkan dari pilihan itu dan dari cakupan, sehingga tidak ada dua sumber
 * kebenaran yang bisa berselisih atau api state-di-dalam-effect.
 */
import { useId, useMemo, useState } from 'react'
import { AlertTriangle, Play, RotateCw } from 'lucide-react'
import { Button, Input, Label, Select, Skeleton } from '@/components/ui'
import { useBacktestCoverage, useBacktestQuota, useCreateBacktest } from '@/lib/queries'
import { ApiError } from '@/lib/api-error'
import type { BacktestPairOption } from '@/types/api'
import {
  dariTanggalInput,
  formatTanggalInput,
  formatTanggalPanjang,
  hitungBulan,
} from '../lib/tanggal'

type Props = {
  symbols: string[]
  symbolTerpilih: string
  onPilihSymbol: (symbol: string) => void
  onDibuat: (runID: string) => void
}

/** Pasangan bawaan mengikuti yang biasa dipakai di dashboard. */
const PASANGAN_BAWAAN = { htf: 'D1', ltf: 'H1' }

export function BacktestForm({ symbols, symbolTerpilih, onPilihSymbol, onDibuat }: Props) {
  const idSymbol = useId()
  const idDari = useId()
  const idSampai = useId()
  const idEquity = useId()
  const idRisk = useId()

  const { data: cakupan, isLoading, isFetching, error, refetch } = useBacktestCoverage(symbolTerpilih)
  const { data: kuota } = useBacktestQuota()
  const buat = useCreateBacktest()

  const [pasanganDiminta, setPasanganDiminta] = useState(PASANGAN_BAWAAN)
  const [rentangDiminta, setRentangDiminta] = useState<{ dari: string; sampai: string } | null>(null)
  const [equity, setEquity] = useState('10000')
  const [risk, setRisk] = useState('0.5')
  const [pesan, setPesan] = useState<string | null>(null)

  const opsi = useMemo(() => cakupan?.pasangan ?? [], [cakupan])

  // Pasangan yang dipakai: permintaan pengguna bila datanya ada, jika tidak
  // pasangan pertama yang tersedia. Diturunkan, bukan diset lewat effect.
  const pasanganAktif = useMemo(() => {
    const diminta = opsi.find(
      (item) => item.htf === pasanganDiminta.htf && item.ltf === pasanganDiminta.ltf,
    )
    if (diminta?.tersedia) return diminta
    return opsi.find((item) => item.tersedia) ?? null
  }, [opsi, pasanganDiminta])

  // Rentang aktif: pilihan pengguna bila masih di dalam cakupan, jika tidak
  // seluruh cakupan. Begitu pasangan berganti, rentang menyesuaikan sendiri —
  // cakupan H1 dan M5 hanya sampai beberapa bulan terakhir, jadi rentang yang
  // tadinya sah bisa berada di luar data.
  const rentangAktif = useMemo(() => {
    if (!pasanganAktif) return { dari: '', sampai: '' }
    const minDari = formatTanggalInput(new Date(pasanganAktif.mulai * 1000))
    const maksSampai = formatTanggalInput(new Date(pasanganAktif.selesai * 1000))
    if (!rentangDiminta) return { dari: minDari, sampai: maksSampai }
    const dari = rentangDiminta.dari < minDari ? minDari : rentangDiminta.dari
    const sampai = rentangDiminta.sampai > maksSampai ? maksSampai : rentangDiminta.sampai
    return { dari, sampai }
  }, [pasanganAktif, rentangDiminta])

  const bisaJalan =
    Boolean(pasanganAktif?.tersedia) &&
    Boolean(rentangAktif.dari) &&
    Boolean(rentangAktif.sampai) &&
    (kuota?.bisa_jalankan ?? true)

  const kirim = async () => {
    setPesan(null)
    if (!symbolTerpilih) {
      setPesan('Pilih pair terlebih dahulu.')
      return
    }
    if (!pasanganAktif) {
      setPesan('Pasangan timeframe ini belum punya data yang cukup.')
      return
    }
    const dariUnix = dariTanggalInput(rentangAktif.dari)
    const sampaiUnix = dariTanggalInput(rentangAktif.sampai)
    if (!dariUnix || !sampaiUnix) {
      setPesan('Lengkapi tanggal mulai dan tanggal akhir.')
      return
    }
    if (sampaiUnix <= dariUnix) {
      setPesan('Tanggal akhir harus setelah tanggal mulai.')
      return
    }

    try {
      const run = await buat.mutateAsync({
        symbol: symbolTerpilih,
        htf: pasanganAktif.htf,
        ltf: pasanganAktif.ltf,
        dari: dariUnix,
        sampai: sampaiUnix,
        equity: Number(equity) || undefined,
        risk: Number(risk) || undefined,
      })
      onDibuat(run.id)
    } catch (galat) {
      setPesan(galat instanceof ApiError ? galat.message : 'Backtest gagal dijalankan.')
    }
  }

  return (
    <div className="flex flex-col gap-5 border border-border p-4 sm:p-5">
      <div className="grid gap-4 sm:grid-cols-2">
        <div className="flex flex-col gap-1.5">
          <Label htmlFor={idSymbol}>Pair</Label>
          <Select
            id={idSymbol}
            value={symbolTerpilih}
            onChange={(e) => onPilihSymbol(e.target.value)}
            disabled={symbols.length === 0}
          >
            {symbols.length === 0 && <option value="">Belum ada pair di katalog</option>}
            {symbols.map((simbol) => (
              <option key={simbol} value={simbol}>
                {simbol}
              </option>
            ))}
          </Select>
        </div>

        <div className="flex flex-col gap-1.5">
          <Label>Metode</Label>
          {/* Satu metode, tetapi tetap ditampilkan: pengguna berhak tahu aturan
              mana yang sedang diuji, dan kelak pilihan ini akan bertambah. */}
          <div className="flex h-10 items-center justify-between rounded-md border border-input px-3 text-sm">
            <span>ILA 2.0</span>
            <span className="text-[11px] text-muted-foreground">satu-satunya yang bisa diuji</span>
          </div>
        </div>
      </div>

      <div className="flex flex-col gap-2">
        <Label>Pasangan timeframe</Label>
        {isLoading ? (
          <Skeleton className="h-12 w-full" />
        ) : (
          <div className="grid grid-cols-2 gap-2 sm:grid-cols-4">
            {opsi.map((item) => (
              <TombolPasangan
                key={`${item.htf}-${item.ltf}`}
                opsi={item}
                aktif={item.htf === pasanganAktif?.htf && item.ltf === pasanganAktif?.ltf}
                onPilih={() => setPasanganDiminta({ htf: item.htf, ltf: item.ltf })}
              />
            ))}
          </div>
        )}
        {pasanganAktif && (
          <p className="text-[12px] text-muted-foreground">
            {pasanganAktif.ltf} tersedia {formatTanggalPanjang(pasanganAktif.mulai)} sampai{' '}
            {formatTanggalPanjang(pasanganAktif.selesai)}.
          </p>
        )}
      </div>

      <div className="grid gap-4 sm:grid-cols-2">
        <div className="flex flex-col gap-1.5">
          <Label htmlFor={idDari}>Tanggal mulai</Label>
          <Input
            id={idDari}
            type="date"
            value={rentangAktif.dari}
            min={pasanganAktif ? formatTanggalInput(new Date(pasanganAktif.mulai * 1000)) : undefined}
            max={rentangAktif.sampai || undefined}
            onChange={(e) => setRentangDiminta({ ...rentangAktif, dari: e.target.value })}
            disabled={!pasanganAktif}
          />
        </div>
        <div className="flex flex-col gap-1.5">
          <Label htmlFor={idSampai}>Tanggal akhir</Label>
          <Input
            id={idSampai}
            type="date"
            value={rentangAktif.sampai}
            min={rentangAktif.dari || undefined}
            max={pasanganAktif ? formatTanggalInput(new Date(pasanganAktif.selesai * 1000)) : undefined}
            onChange={(e) => setRentangDiminta({ ...rentangAktif, sampai: e.target.value })}
            disabled={!pasanganAktif}
          />
        </div>
      </div>

      <div className="grid gap-4 sm:grid-cols-2">
        <div className="flex flex-col gap-1.5">
          <Label htmlFor={idEquity}>Modal awal</Label>
          <Input
            id={idEquity}
            type="number"
            inputMode="decimal"
            min={100}
            step={100}
            value={equity}
            onChange={(e) => setEquity(e.target.value)}
            className="tabular"
          />
        </div>
        <div className="flex flex-col gap-1.5">
          <Label htmlFor={idRisk}>Risiko per trade (persen)</Label>
          <Input
            id={idRisk}
            type="number"
            inputMode="decimal"
            min={0.1}
            max={5}
            step={0.1}
            value={risk}
            onChange={(e) => setRisk(e.target.value)}
            className="tabular"
          />
        </div>
      </div>

      <p className="text-[12px] leading-relaxed text-muted-foreground">
        Aturan harian mengikuti mesin live: paling banyak dua trade sehari, berhenti setelah menang,
        dan paling banyak dua kerugian. Target laba ditentukan metode, bukan diatur di sini.
      </p>

      {error && (
        <div className="flex items-start gap-2 border border-border bg-secondary/40 p-3 text-[13px]">
          <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0" aria-hidden="true" />
          <span>Cakupan data gagal dibaca. {error instanceof ApiError ? error.message : ''}</span>
        </div>
      )}

      {pesan && (
        <div
          role="alert"
          className="flex items-start gap-2 border border-destructive/40 bg-destructive/5 p-3 text-[13px] text-destructive"
        >
          <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0" aria-hidden="true" />
          <span>{pesan}</span>
        </div>
      )}

      <div className="flex flex-wrap items-center justify-between gap-3 border-t border-border pt-4">
        <div className="flex flex-col gap-0.5 text-[12px] text-muted-foreground">
          <span>
            {kuota
              ? kuota.sisa < 0
                ? 'Tanpa batas harian'
                : `Sisa jatah hari ini ${kuota.sisa} dari ${kuota.batas_harian}`
              : 'Menghitung jatah…'}
          </span>
          <span>
            {kuota
              ? `Rentang paling panjang ${kuota.rentang_maks_bulan} bulan untuk paket ${kuota.tier}`
              : '—'}
          </span>
        </div>

        <div className="flex items-center gap-2">
          <Button
            variant="ghost"
            size="sm"
            onClick={() => void refetch()}
            disabled={isFetching || !symbolTerpilih}
          >
            <RotateCw
              className={isFetching ? 'h-4 w-4 animate-spin' : 'h-4 w-4'}
              aria-hidden="true"
            />
            Periksa ulang data
          </Button>
          <Button onClick={() => void kirim()} isLoading={buat.isPending} disabled={!bisaJalan}>
            <Play className="h-4 w-4" aria-hidden="true" />
            {buat.isPending ? 'Menyiapkan…' : 'Jalankan backtest'}
          </Button>
        </div>
      </div>

      {kuota && !kuota.bisa_jalankan && kuota.alasan && (
        <p className="text-[12px] text-muted-foreground">{kuota.alasan}</p>
      )}
    </div>
  )
}

/**
 * TombolPasangan adalah satu pilihan pasangan timeframe.
 *
 * Pasangan yang datanya tidak mencukupi tetap ditampilkan, tetapi tidak bisa
 * dipilih. Menyembunyikannya membuat pengguna mengira pasangannya tidak ada,
 * padahal masalahnya hanya cakupan data.
 */
function TombolPasangan({
  opsi,
  aktif,
  onPilih,
}: {
  opsi: BacktestPairOption
  aktif: boolean
  onPilih: () => void
}) {
  return (
    <button
      type="button"
      onClick={onPilih}
      disabled={!opsi.tersedia}
      aria-pressed={aktif}
      className={[
        'flex flex-col gap-0.5 rounded-md border px-3 py-2 text-left transition-colors',
        'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background',
        aktif ? 'border-foreground bg-foreground text-background' : 'border-input hover:bg-secondary',
        !opsi.tersedia && 'cursor-not-allowed opacity-40 hover:bg-transparent',
      ]
        .filter(Boolean)
        .join(' ')}
    >
      <span className="text-[13px] font-semibold">
        {opsi.htf} ke {opsi.ltf}
      </span>
      <span className={['text-[11px]', aktif ? 'text-background/70' : 'text-muted-foreground'].join(' ')}>
        {opsi.tersedia ? `${hitungBulan(opsi.mulai, opsi.selesai)} bulan data` : 'data belum cukup'}
      </span>
    </button>
  )
}
