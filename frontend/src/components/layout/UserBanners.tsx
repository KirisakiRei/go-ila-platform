import { useQuery } from '@tanstack/react-query'
import { accountFlowApi } from '@/lib/auth-api'
import { publicContentApi } from '@/lib/system-api'
import { Alert } from '@/components/ui'

/**
 * UserBanners menampilkan pemberitahuan di atas konten dashboard.
 *
 * Tiga hal yang wajib terlihat: status langganan, email yang belum
 * terverifikasi, dan pengumuman aktif. Engine yang tidak melapor ikut
 * ditampilkan karena sinyal baru tidak akan datang tanpanya (plans/11 I6).
 */
export function UserBanners() {
  const banner = useQuery({ queryKey: ['me', 'banner'], queryFn: accountFlowApi.banner })
  const announcements = useQuery({ queryKey: ['public', 'announcements'], queryFn: publicContentApi.announcements })

  const items: { key: string; variant: 'warning' | 'destructive' | 'info'; text: string }[] = []

  if (banner.data && banner.data.state !== 'normal') {
    items.push({
      key: 'subscription',
      variant: banner.data.state === 'expired' ? 'destructive' : 'warning',
      text: `${banner.data.title}. ${banner.data.message}`,
    })
  }

  for (const announcement of announcements.data ?? []) {
    items.push({
      key: `announcement-${announcement.id}`,
      variant: announcement.level === 'critical' ? 'destructive' : 'info',
      text: `${announcement.title} — ${announcement.body}`,
    })
  }

  if (items.length === 0) return null

  return (
    <div className="space-y-2">
      {items.map((item) => (
        <Alert key={item.key} variant={item.variant}>
          {item.text}
        </Alert>
      ))}
    </div>
  )
}
