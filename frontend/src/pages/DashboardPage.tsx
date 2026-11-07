import { SystemStatusCard } from '@/features/dashboard/components/SystemStatusCard'
import { TierCard } from '@/features/dashboard/components/TierCard'
import { EmptyState } from '@/features/dashboard/components/EmptyState'
import type { User } from '@/types/api'

export default function DashboardPage({ user }: { user: User }) {
  const firstName = user.name.split(' ')[0] ?? user.name

  return (
    <div className="flex flex-col gap-4">
      <header className="flex flex-col gap-1">
        <h1 className="text-xl font-semibold tracking-tight">Selamat datang, {firstName}</h1>
        <p className="text-[13px] text-muted-foreground">
          Pemantauan pasar akan aktif setelah akun broker dihubungkan.
        </p>
      </header>

      <SystemStatusCard user={user} />
      <EmptyState />
      <TierCard tier={user.tier} />
    </div>
  )
}
