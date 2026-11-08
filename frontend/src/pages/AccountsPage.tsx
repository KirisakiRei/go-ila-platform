import { useEffect, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import {
  Alert,
  Button,
  EmptyState,
  Modal,
  PageHeader,
  Section,
  Skeleton,
  StatusDot,
} from '@/components/ui'
import { AccountCard } from '@/features/accounts/components/AccountCard'
import { AccountForm } from '@/features/accounts/components/AccountForm'
import { useAccounts, useCTraderCallback } from '@/lib/queries'
import { useAuthContext } from '@/features/auth/hooks/use-auth'
import { ApiError } from '@/lib/api-error'

/** Batas akun per tier, mengikuti aturan backend. */
const LIMIT: Record<string, number> = { free: 0, basic: 1, plus: 2 }

export default function AccountsPage() {
  const { user } = useAuthContext()
  const { data, isLoading, isError, error } = useAccounts()
  const [menambah, setMenambah] = useState(false)
  const [searchParams, setSearchParams] = useSearchParams()
  const [callbackNotice, setCallbackNotice] = useState<{ type: 'success' | 'error'; message: string } | null>(null)

  const ctraderCallback = useCTraderCallback()

  // Tangkap query param '?code=...' bila baru dialihkan dari Spotware cTrader OAuth
  useEffect(() => {
    const code = searchParams.get('code')
    if (!code) return

    // Bersihkan parameter URL agar tidak diproses berulang kali
    searchParams.delete('code')
    setSearchParams(searchParams, { replace: true })

    // Ambil preferensi mode/risk yang disimpan sebelum diarahkan ke cTrader
    let settings = { mode: 'manual', risk_percent: 0.5 }
    try {
      const stored = sessionStorage.getItem('ctrader_pending_settings')
      if (stored) {
        settings = JSON.parse(stored)
        sessionStorage.removeItem('ctrader_pending_settings')
      }
    } catch {
      // gunakan default
    }

    ctraderCallback
      .mutateAsync({
        code,
        mode: settings.mode,
        risk_percent: settings.risk_percent,
      })
      .then((res) => {
        setCallbackNotice({
          type: 'success',
          message: `Berhasil menghubungkan ${res.total_connected || 'semua'} akun dari cTrader ID Anda.`,
        })
      })
      .catch((err) => {
        setCallbackNotice({
          type: 'error',
          message: err instanceof ApiError ? err.message : 'Gagal menghubungkan akun cTrader.',
        })
      })
  }, [searchParams, setSearchParams, ctraderCallback])

  const tier = user?.tier ?? 'free'
  const limit = LIMIT[tier] ?? 0
  const jumlah = data?.length ?? 0
  const bisaTambah = jumlah < limit

  return (
    <>
      <PageHeader
        title="Akun Broker"
        description={
          limit === 0
            ? 'Tier Free belum dapat menghubungkan akun. Tingkatkan ke Basic atau Plus untuk mulai.'
            : `Tier ${tier} dapat menghubungkan ${limit} akun. Saat ini ${jumlah} terhubung.`
        }
        actions={
          bisaTambah && !menambah ? (
            <Button size="sm" onClick={() => setMenambah(true)}>
              Hubungkan akun
            </Button>
          ) : null
        }
      />

      {isLoading && <Skeleton className="h-40 w-full" />}
      {isError && <Alert variant="destructive">{(error as Error).message}</Alert>}

      {ctraderCallback.isPending && (
        <Alert variant="default" className="border-border bg-secondary/50">
          <div className="flex items-center gap-2">
            <span className="h-2 w-2 animate-ping rounded-full bg-emerald-500" />
            <span className="text-xs font-semibold">Sedang menukar token dan mengimpor akun cTrader Anda...</span>
          </div>
        </Alert>
      )}

      {callbackNotice && (
        <Alert
          variant={callbackNotice.type === 'success' ? 'success' : 'destructive'}
          className="text-xs"
        >
          {callbackNotice.message}
        </Alert>
      )}

      {data?.length === 0 && !isLoading && (
        <EmptyState
          title="Belum ada akun terhubung"
          description={
            limit === 0
              ? 'Hubungkan akun broker agar sinyal dapat dieksekusi.'
              : 'Hubungkan akun cTrader (OAuth 1-klik) atau Match-Trader (Access Token) untuk memulai.'
          }
        />
      )}

      {data && data.length > 0 && (
        <Section title="Akun terhubung" description="Kredensial tersimpan terenkripsi.">
          <div className="flex flex-col gap-3">
            {data.map((account) => (
              <AccountCard key={account.id} account={account} />
            ))}
          </div>
        </Section>
      )}

      {/* Modal Tambah Akun Broker */}
      <Modal
        open={menambah}
        onClose={() => setMenambah(false)}
        size="lg"
        title="Hubungkan Akun Broker"
        description="Pilih platform trading broker Anda (cTrader OAuth atau Match-Trader)."
      >
        <div className="py-2">
          <AccountForm tier={tier} onDone={() => setMenambah(false)} />
        </div>
      </Modal>

      {!bisaTambah && limit > 0 && (
        <p className="flex items-center gap-2 text-[12px] text-muted-foreground">
          <StatusDot state="idle" label="batas tercapai" />
          Batas akun untuk tier {tier} sudah tercapai.
        </p>
      )}
    </>
  )
}
