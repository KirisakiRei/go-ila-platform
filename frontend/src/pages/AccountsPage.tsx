import { useState } from 'react'
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
import { useAccounts } from '@/lib/queries'
import { useAuthContext } from '@/features/auth/hooks/use-auth'

/** Batas akun per tier, mengikuti aturan backend. */
const LIMIT: Record<string, number> = { free: 0, basic: 1, plus: 2 }

export default function AccountsPage() {
  const { user } = useAuthContext()
  const { data, isLoading, isError, error } = useAccounts()
  const [menambah, setMenambah] = useState(false)

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

      {data?.length === 0 && !isLoading && (
        <EmptyState
          title="Belum ada akun terhubung"
          description={
            limit === 0
              ? 'Hubungkan akun broker agar sinyal dapat dieksekusi.'
              : 'Tambahkan akun Match-Trader dengan access token dari aplikasi broker Anda.'
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
        title="Hubungkan Akun Match-Trader"
        description="Hanya platform broker Match-Trader yang didukung saat ini. Token hanya dikirim sekali dan langsung dienkripsi."
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
