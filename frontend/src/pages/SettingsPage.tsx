import { useState } from 'react'
import { useStartTrial, useTelegramLink, useTelegramUnlink, useTrial } from '@/lib/queries'
import { ApiError } from '@/lib/api-error'
import { formatDateTime, formatDate } from '@/lib/format'
import {
  Alert,
  Badge,
  Button,
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
  Skeleton,
} from '@/components/ui'

function TrialSection() {
  const { data, isLoading, isError, error } = useTrial()
  const start = useStartTrial()
  const [actionError, setActionError] = useState<string | null>(null)

  const handleStart = async () => {
    setActionError(null)
    try {
      await start.mutateAsync()
    } catch (err) {
      setActionError(err instanceof ApiError ? err.message : 'Gagal memulai uji coba.')
    }
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>Uji coba 7 hari</CardTitle>
        <CardDescription>
          Setara tier Plus, namun seluruh eksekusi berjalan sebagai simulasi.
        </CardDescription>
      </CardHeader>
      <CardContent className="flex flex-col gap-3">
        {isLoading && <Skeleton className="h-16 w-full" />}
        {isError && <Alert>{(error as Error).message}</Alert>}

        {data && (
          <>
            {data.active ? (
              <div className="flex flex-col gap-1 text-[13px]">
                <p>
                  Aktif · sisa <strong>{data.days_remaining} hari</strong>
                </p>
                <p className="text-muted-foreground">
                  Berakhir {formatDate(data.trial_expires_at)} · mulai{' '}
                  {formatDateTime(data.trial_started_at)}
                </p>
                <Badge variant="warning" className="self-start">
                  Mode simulasi wajib
                </Badge>
              </div>
            ) : data.used ? (
              <p className="text-[13px] text-muted-foreground">
                Uji coba sudah pernah dipakai. Masa berlaku tidak dapat diulang.
              </p>
            ) : (
              <p className="text-[13px] text-muted-foreground">
                Anda belum memakai uji coba. Uji coba hanya bisa dipakai sekali per akun.
              </p>
            )}

            {!data.used && (
              <Button
                className="self-start"
                isLoading={start.isPending}
                onClick={() => void handleStart()}
              >
                Mulai uji coba
              </Button>
            )}
          </>
        )}

        {actionError && <Alert>{actionError}</Alert>}
      </CardContent>
    </Card>
  )
}

function TelegramSection() {
  const link = useTelegramLink()
  const unlink = useTelegramUnlink()
  const [actionError, setActionError] = useState<string | null>(null)

  const handleLink = async () => {
    setActionError(null)
    try {
      const result = await link.mutateAsync()
      // Dibuka di tab baru agar sesi dashboard tidak hilang.
      window.open(result.url, '_blank', 'noopener,noreferrer')
    } catch (err) {
      setActionError(err instanceof ApiError ? err.message : 'Gagal membuat tautan Telegram.')
    }
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>Notifikasi Telegram</CardTitle>
        <CardDescription>
          Terima pemberitahuan sinyal lewat bot. Bot tidak pernah mengakses kredensial akun.
        </CardDescription>
      </CardHeader>
      <CardContent className="flex flex-col gap-3">
        <div className="flex flex-wrap gap-2">
          <Button isLoading={link.isPending} onClick={() => void handleLink()}>
            Hubungkan Telegram
          </Button>
          <Button
            variant="outline"
            isLoading={unlink.isPending}
            onClick={() => void unlink.mutateAsync()}
          >
            Putuskan
          </Button>
        </div>
        <p className="text-[11px] text-muted-foreground">
          Tautan berlaku 5 menit dan hanya bisa dipakai sekali.
        </p>
        {actionError && <Alert>{actionError}</Alert>}
        {unlink.isSuccess && <Alert variant="info">Chat Telegram sudah diputus.</Alert>}
      </CardContent>
    </Card>
  )
}

export default function SettingsPage() {
  return (
    <div className="flex flex-col gap-4">
      <header className="flex flex-col gap-1">
        <h1 className="text-xl font-semibold tracking-tight">Pengaturan</h1>
        <p className="text-[13px] text-muted-foreground">Uji coba dan saluran notifikasi.</p>
      </header>

      <TrialSection />
      <TelegramSection />
    </div>
  )
}
