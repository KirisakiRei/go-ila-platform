import { useHealth } from '../hooks/use-health'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
  StatusDot,
  type StatusState,
} from '@/components/ui'
import type { User } from '@/types/api'

type Row = {
  label: string
  state: StatusState
  detail: string
}

/**
 * SystemStatusCard menampilkan kesiapan komponen yang sudah ada.
 * Engine, akun broker, dan Telegram masih menunggu pipeline terkait dibangun,
 * jadi ditampilkan apa adanya alih-alih disembunyikan.
 */
export function SystemStatusCard({ user }: { user: User }) {
  const { data: health, isLoading } = useHealth()

  const backendState: StatusState = isLoading ? 'pending' : (health ?? 'offline')

  const rows: Row[] = [
    {
      label: 'Backend API',
      state: backendState,
      detail: isLoading ? 'memeriksa…' : backendState === 'online' ? 'aktif' : 'tidak terjangkau',
    },
    { label: 'Engine Trading', state: 'idle', detail: 'belum terhubung' },
    { label: 'Akun Broker', state: 'idle', detail: 'belum ada' },
    { label: 'Telegram', state: 'idle', detail: 'belum terhubung' },
  ]

  return (
    <Card>
      <CardHeader>
        <CardTitle>Status Sistem</CardTitle>
        <CardDescription>Tier aktif: {user.tier}</CardDescription>
      </CardHeader>
      <CardContent>
        <ul className="flex flex-col divide-y divide-border">
          {rows.map((row) => (
            <li key={row.label} className="flex items-center gap-3 py-2.5">
              <StatusDot state={row.state} label={`${row.label}: ${row.detail}`} />
              <span className="text-[13px]">{row.label}</span>
              <span className="ml-auto text-[13px] text-muted-foreground">{row.detail}</span>
            </li>
          ))}
        </ul>
      </CardContent>
    </Card>
  )
}
