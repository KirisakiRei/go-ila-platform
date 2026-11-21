import { useState } from 'react'
import { Link } from 'react-router-dom'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { ArrowLeft, CheckCircle2 } from 'lucide-react'
import { Alert, Badge, Button, Card, CardContent, CardHeader, CardTitle, PageHeader, Skeleton } from '@/components/ui'
import { PushNotificationPanel } from '@/features/notifications/PushNotificationPanel'
import { accountFlowApi } from '@/lib/auth-api'
import { ApiError } from '@/lib/api-error'
import { useTelegramLink, useTelegramUnlink } from '@/lib/queries'

function TelegramCard() {
  const queryClient = useQueryClient()
  const link = useTelegramLink()
  const unlink = useTelegramUnlink()
  const [galat, setGalat] = useState<string | null>(null)

  const { data: onboarding, isLoading } = useQuery({
    queryKey: ['me', 'onboarding'],
    queryFn: accountFlowApi.onboarding,
  })

  const isConnected = Boolean(onboarding?.telegram_linked)

  const onLink = async () => {
    setGalat(null)
    try {
      const hasil = await link.mutateAsync()
      window.open(hasil.url, '_blank', 'noopener,noreferrer')
    } catch (err) {
      setGalat(err instanceof ApiError ? err.message : 'Gagal membuat tautan Telegram.')
    }
  }

  const onUnlink = async () => {
    setGalat(null)
    try {
      await unlink.mutateAsync()
      await queryClient.invalidateQueries({ queryKey: ['me', 'onboarding'] })
      await queryClient.invalidateQueries({ queryKey: ['trial'] })
    } catch (err) {
      setGalat(err instanceof ApiError ? err.message : 'Gagal memutuskan Telegram.')
    }
  }

  return (
    <Card>
      <CardHeader className="pb-3">
        <div className="flex items-center justify-between gap-2">
          <CardTitle className="text-sm">Telegram</CardTitle>
          {!isLoading &&
            (isConnected ? (
              <span className="inline-flex items-center gap-1 rounded-md border border-emerald-500/30 bg-emerald-500/10 px-2 py-0.5 text-[11px] font-semibold text-emerald-700 dark:text-emerald-400">
                <CheckCircle2 className="h-3 w-3" />
                Terhubung
              </span>
            ) : (
              <Badge variant="outline" className="text-muted-foreground">
                Belum Terhubung
              </Badge>
            ))}
        </div>
      </CardHeader>
      <CardContent className="space-y-3">
        {isLoading ? (
          <Skeleton className="h-14 w-full" />
        ) : (
          <>
            <p className="text-xs text-muted-foreground">
              Bot <span className="font-mono text-foreground">@imaginaryliquiditybot</span> — kirim
              alert sinyal siap entry dan status order.
            </p>

            <div className="flex flex-wrap gap-2">
              {isConnected ? (
                <>
                  <Button
                    size="sm"
                    variant="outline"
                    isLoading={unlink.isPending}
                    onClick={() => void onUnlink()}
                  >
                    Putuskan
                  </Button>
                  <Button
                    size="sm"
                    variant="ghost"
                    isLoading={link.isPending}
                    onClick={() => void onLink()}
                  >
                    Hubungkan Ulang
                  </Button>
                </>
              ) : (
                <Button size="sm" isLoading={link.isPending} onClick={() => void onLink()}>
                  Hubungkan Telegram
                </Button>
              )}
            </div>

            {galat && <Alert variant="destructive">{galat}</Alert>}
            {unlink.isSuccess && <Alert variant="success">Telegram diputus.</Alert>}
          </>
        )}
      </CardContent>
    </Card>
  )
}

/**
 * NotifikasiPage menyatukan pengaturan Telegram dan Web Push dalam satu halaman ringkas.
 */
export default function NotifikasiPage() {
  return (
    <>
      <div className="mb-2">
        <Link
          to="/pengaturan"
          className="inline-flex items-center gap-1.5 text-xs font-medium text-muted-foreground transition-colors duration-150 hover:text-foreground"
        >
          <ArrowLeft className="h-3.5 w-3.5" />
          Kembali ke Pengaturan
        </Link>
      </div>
      <PageHeader
        title="Notifikasi"
        description="Atur saluran alert Telegram dan Web Push."
      />
      <div className="flex flex-col gap-4">
        <TelegramCard />
        <Card>
          <CardHeader className="pb-3">
            <CardTitle className="text-sm">Web Push</CardTitle>
          </CardHeader>
          <CardContent>
            <PushNotificationPanel />
          </CardContent>
        </Card>
      </div>
    </>
  )
}
