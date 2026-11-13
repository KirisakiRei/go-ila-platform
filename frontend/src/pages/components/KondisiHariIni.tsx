import { Link } from 'react-router-dom'
import { ArrowRight, Radio } from 'lucide-react'
import { Card, CardContent, Skeleton, StatusDot } from '@/components/ui'
import { usePositions, useSystemStatus } from '@/lib/queries'
import { publicContentApi } from '@/lib/system-api'
import { useQuery } from '@tanstack/react-query'
import { formatMoney } from '@/lib/format'
import { cn } from '@/lib/utils'
import { kondisiPasar, jamTutupMenyusul, tanggalWIB, POSISI_LIMIT_DASHBOARD } from '@/features/signals/lib/kondisi-pasar'

/**
 * KondisiHariIni merangkum keadaan yang menentukan boleh-tidaknya trading hari ini.
 *
 * Komponen ini menggantikan tiga kartu terpisah (Batas harian, Status Sistem,
 * dan Sesi pasar) yang menampilkan angka yang sama berulang kali. Dashboard
 * adalah ringkasan: setiap informasi cukup muncul sekali.
 */
export function KondisiHariIni({ className }: { className?: string }) {
  const { data: status, isLoading, isError, error } = useSystemStatus()
  const { data: posisi } = usePositions(POSISI_LIMIT_DASHBOARD)

  // Jadwal sesi hanya tersedia di endpoint publik; dipakai agar status pasar
  // tidak pernah tampil "belum diketahui" padahal jadwalnya jelas.
  const { data: publik } = useQuery({
    queryKey: ['public', 'status'],
    queryFn: publicContentApi.status,
  })

  if (isLoading) return <Skeleton className={cn('h-64 w-full', className)} />
  if (isError) {
    return (
      <Card className={className}>
        <CardContent className="p-4 text-[13px] text-muted-foreground">
          Ringkasan gagal dimuat: {(error as Error).message}
        </CardContent>
      </Card>
    )
  }
  if (!status) return null

  const { daily, brokers, engines } = status
  const pasar = kondisiPasar(status.market, publik?.market)
  const bolehTrading = !daily.locked && daily.total_trades < daily.max_trades
  const sisa = Math.max(0, daily.max_trades - daily.total_trades)
  const persen = Math.min(100, (daily.total_trades / Math.max(1, daily.max_trades)) * 100)

  // P/L hari ini dihitung dari posisi yang ditutup pada hari akuntansi yang
  // sama dengan `daily.day`. Posisi lama tidak dihitung agar angkanya benar-benar
  // mencerminkan hari ini, bukan akumulasi sepanjang masa.
  //
  // Tanggal diturunkan dalam WIB, bukan dari potongan string UTC: `daily.day`
  // memakai batas tengah malam WIB, sedangkan `closed_at` dikirim dalam UTC.
  // Posisi yang ditutup 17:43 UTC tanggal 12 sebenarnya masuk hari akuntansi
  // 13 Nov WIB; membandingkan potongan UTC membuatnya hilang dari hitungan.
  const closedHariIni = (posisi ?? []).filter(
    (p) => p.closed_at && tanggalWIB(p.closed_at) === daily.day,
  )
  const plHariIni = closedHariIni.reduce((total, p) => total + (p.profit_loss ?? 0), 0)
  const totalSelesai = daily.win_count + daily.loss_count
  const winRate = totalSelesai > 0 ? Math.round((daily.win_count / totalSelesai) * 100) : null

  const brokerSehat = brokers.connected > 0 && !brokers.any_error

  const jadwalBerikut = publik?.market?.next_change_pair?.wib
    ? `${jamTutupMenyusul(publik.market) ? 'Tutup' : publik.market.next_label} ${publik.market.next_change_pair.wib} WIB`
    : null

  return (
    <Card className={className}>
      <CardContent className="flex flex-col gap-4 p-4">
        {/* Baris 1: kondisi pasar & jadwal sesi */}
        <div className="flex items-start justify-between gap-3">
          <div className="min-w-0">
            <p className="text-micro font-bold uppercase tracking-wider text-muted-foreground">
              Kondisi Pasar
            </p>
            <div className="mt-1 flex items-center gap-2">
              <StatusDot
                state={pasar.state === 'OPEN' ? 'online' : pasar.state === 'ROLLOVER' ? 'pending' : 'offline'}
                label={pasar.label}
              />
              <span className="text-sm font-semibold text-foreground">{pasar.label}</span>
            </div>
            <p className="mt-0.5 text-caption text-muted-foreground">
              {jadwalBerikut ?? 'Jadwal sesi belum tersedia'}
            </p>
          </div>
          <div className="shrink-0 text-right text-micro text-muted-foreground">
            <p className="font-mono">WIB {publik?.time_pair?.wib ?? '—'}</p>
            <p className="font-mono">NY {publik?.time_pair?.ny ?? '—'}</p>
          </div>
        </div>

        {/* Baris 2: kuota trading hari ini */}
        <div className="border-t border-border pt-3">
          <div className="flex items-baseline justify-between">
            <p className="text-micro font-bold uppercase tracking-wider text-muted-foreground">
              Kuota Trading
            </p>
            <p className="font-mono text-[12.5px] font-semibold text-foreground">
              {daily.total_trades} / {daily.max_trades}
            </p>
          </div>

          <div className="mt-1.5 h-1.5 w-full overflow-hidden rounded-full bg-secondary" role="presentation">
            <div
              className={cn(
                'h-full transition-all',
                daily.locked ? 'bg-rose-500' : bolehTrading ? 'bg-foreground' : 'bg-amber-500',
              )}
              style={{ width: `${persen}%` }}
            />
          </div>

          <p className="mt-1.5 text-caption text-muted-foreground">
            {daily.locked
              ? daily.lock_reason || 'Entry baru ditahan sampai besok.'
              : bolehTrading
                ? `Sisa ${sisa} eksekusi hari ini`
                : 'Kuota hari ini sudah terpakai penuh'}
          </p>
        </div>

        {/* Baris 3: hasil hari ini */}
        <div className="grid grid-cols-3 gap-2 border-t border-border pt-3">
          <div>
            <p className="text-micro uppercase tracking-wide text-muted-foreground">P/L Hari Ini</p>
            <p
              className={cn(
                'font-mono text-[13px] font-semibold tabular-nums',
                closedHariIni.length === 0
                  ? 'text-muted-foreground'
                  : plHariIni > 0
                    ? 'text-emerald-600 dark:text-emerald-400'
                    : plHariIni < 0
                      ? 'text-rose-600 dark:text-rose-400'
                      : 'text-foreground',
              )}
            >
              {closedHariIni.length === 0 ? '—' : formatMoney(plHariIni)}
            </p>
          </div>
          <div>
            <p className="text-micro uppercase tracking-wide text-muted-foreground">Win / Loss</p>
            <p className="font-mono text-[13px] font-semibold tabular-nums text-foreground">
              {daily.win_count} / {daily.loss_count}
              {winRate !== null && (
                <span className="ml-1 text-micro font-normal text-muted-foreground">({winRate}%)</span>
              )}
            </p>
          </div>
          <div>
            <p className="text-micro uppercase tracking-wide text-muted-foreground">Posisi Aktif</p>
            <p className="font-mono text-[13px] font-semibold tabular-nums text-foreground">
              {daily.open_count}
            </p>
          </div>
        </div>

        {/* Baris 4: kesiapan sistem */}
        <div className="flex flex-wrap items-center gap-x-4 gap-y-1.5 border-t border-border pt-3 text-caption">
          {engines.length === 0 ? (
            <span className="flex items-center gap-1.5 text-muted-foreground">
              <StatusDot state="idle" label="Engine belum terhubung" />
              Engine menunggu koneksi
            </span>
          ) : (
            engines.map((engine) => (
              <span key={engine.strategy} className="flex items-center gap-1.5">
                <StatusDot
                  state={engine.connected ? 'online' : 'offline'}
                  label={`${engine.strategy} ${engine.connected ? 'aktif' : 'nonaktif'}`}
                />
                <span className="font-mono text-muted-foreground">
                  {engine.strategy}
                  {engine.engine_version ? ` v${engine.engine_version}` : ''}
                </span>
              </span>
            ))
          )}

          <Link
            to="/akun"
            className="ml-auto flex items-center gap-1.5 font-medium text-foreground hover:underline"
          >
            <StatusDot
              state={brokerSehat ? 'online' : brokers.any_error ? 'pending' : 'idle'}
              label={`${brokers.connected} dari ${brokers.total} akun tersambung`}
            />
            {brokers.connected}/{brokers.total} akun
            <ArrowRight className="h-3 w-3" />
          </Link>
        </div>
      </CardContent>
    </Card>
  )
}

/**
 * KondisiHariIniRingkas adalah versi satu baris untuk layar sempit.
 *
 * Mobile tetap memakai urutan blok yang sudah dikenal pengguna, jadi yang
 * dibutuhkan di sana bukan kartu penuh melainkan pengganti angka-angka yang
 * dulu terpisah. Detail lengkap tetap tersedia lewat halaman masing-masing.
 */
export function KondisiHariIniRingkas({ className }: { className?: string }) {
  const { data: status, isLoading } = useSystemStatus()
  const { data: publik } = useQuery({
    queryKey: ['public', 'status'],
    queryFn: publicContentApi.status,
  })

  // Kegagalan muat harus tetap terlihat: baris kosong di tengah dashboard
  // membuat pengguna mengira tidak ada kuota sama sekali.
  if (isLoading) return <Skeleton className={cn('h-16 w-full', className)} />
  if (!status) {
    return (
      <div className={cn('rounded-xl border border-border bg-card px-3 py-2.5 text-caption text-muted-foreground', className)}>
        Ringkasan hari ini gagal dimuat. Tarik ke bawah untuk mencoba lagi.
      </div>
    )
  }

  const pasar = kondisiPasar(status.market, publik?.market)
  const { daily, brokers } = status
  const sisa = Math.max(0, daily.max_trades - daily.total_trades)

  return (
    <div
      className={cn(
        'flex items-center justify-between gap-3 rounded-xl border border-border bg-card px-3 py-2.5',
        className,
      )}
    >
      <div className="flex items-center gap-2">
        <StatusDot
          state={pasar.state === 'OPEN' ? 'online' : pasar.state === 'ROLLOVER' ? 'pending' : 'offline'}
          label={pasar.label}
        />
        <div className="leading-tight">
          <p className="text-[12px] font-semibold text-foreground">{pasar.label}</p>
          <p className="text-micro text-muted-foreground">
            {daily.locked
              ? 'Batas harian tercapai'
              : `Sisa ${sisa} dari ${daily.max_trades} eksekusi`}
          </p>
        </div>
      </div>

      <div className="flex items-center gap-3 text-micro">
        <div className="text-right leading-tight">
          <p className="font-mono font-semibold text-foreground">
            {daily.win_count}/{daily.loss_count}
          </p>
          <p className="text-micro text-muted-foreground">W/L</p>
        </div>
        <div className="flex items-center gap-1 text-muted-foreground" title="Akun broker tersambung">
          <Radio className="h-3.5 w-3.5" aria-hidden="true" />
          <span className="font-mono">
            {brokers.connected}/{brokers.total}
          </span>
        </div>
      </div>
    </div>
  )
}

