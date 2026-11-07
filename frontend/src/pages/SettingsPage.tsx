import { useState } from 'react'
import { Alert, Badge, Button, PageHeader, Section, Skeleton } from '@/components/ui'
import { useStartTrial, useTelegramLink, useTelegramUnlink, useTrial } from '@/lib/queries'
import { ApiError } from '@/lib/api-error'
import { formatDate, formatDateTime } from '@/lib/format'

function TrialSection() {
  const { data, isLoading, isError, error } = useTrial()
  const mulai = useStartTrial()
  const [galat, setGalat] = useState<string | null>(null)

  const onMulai = async () => {
    setGalat(null)
    try {
      await mulai.mutateAsync()
    } catch (err) {
      setGalat(err instanceof ApiError ? err.message : 'Gagal memulai uji coba.')
    }
  }

  return (
    <Section
      title="Uji coba 7 hari"
      description="Setara tier Plus, namun seluruh eksekusi berjalan sebagai simulasi."
    >
      <div className="flex flex-col gap-3 border border-border p-4">
        {isLoading && <Skeleton className="h-16 w-full" />}
        {isError && <Alert variant="destructive">{(error as Error).message}</Alert>}

        {data && (
          <>
            {data.active ? (
              <div className="flex flex-col gap-2 text-[13px]">
                <div className="flex flex-wrap items-center gap-2">
                  <Badge variant="solid">Aktif</Badge>
                  <span>
                    sisa <strong>{data.days_remaining} hari</strong>
                  </span>
                  <Badge variant="outline">Simulasi wajib</Badge>
                </div>
                <p className="text-[12px] text-muted-foreground">
                  Mulai {formatDateTime(data.trial_started_at)} · berakhir{' '}
                  {formatDate(data.trial_expires_at)}
                </p>
              </div>
            ) : data.used ? (
              <p className="text-[13px] text-muted-foreground">
                Uji coba sudah pernah dipakai. Masa berlaku tidak dapat diulang.
              </p>
            ) : (
              <div className="flex flex-col gap-2">
                <p className="text-[13px] text-muted-foreground">
                  Anda belum memakai uji coba. Uji coba hanya bisa dipakai sekali per akun.
                </p>
                <Button
                  size="sm"
                  className="self-start"
                  isLoading={mulai.isPending}
                  onClick={() => void onMulai()}
                >
                  Mulai uji coba
                </Button>
              </div>
            )}
          </>
        )}

        {galat && <Alert variant="destructive">{galat}</Alert>}
      </div>
    </Section>
  )
}

function TelegramSection() {
  const link = useTelegramLink()
  const unlink = useTelegramUnlink()
  const [galat, setGalat] = useState<string | null>(null)

  const onLink = async () => {
    setGalat(null)
    try {
      const hasil = await link.mutateAsync()
      // Dibuka di tab baru agar sesi dashboard tidak hilang.
      window.open(hasil.url, '_blank', 'noopener,noreferrer')
    } catch (err) {
      setGalat(err instanceof ApiError ? err.message : 'Gagal membuat tautan Telegram.')
    }
  }

  return (
    <Section
      title="Notifikasi Telegram"
      description="Bot hanya mengirim pemberitahuan. Bot tidak pernah mengakses kredensial akun Anda."
    >
      <div className="flex flex-col gap-3 border border-border p-4">
        <div className="flex flex-wrap gap-2">
          <Button size="sm" isLoading={link.isPending} onClick={() => void onLink()}>
            Hubungkan Telegram
          </Button>
          <Button
            size="sm"
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

        {galat && <Alert variant="destructive">{galat}</Alert>}
        {unlink.isSuccess && <Alert variant="success">Chat Telegram sudah diputus.</Alert>}
      </div>
    </Section>
  )
}

export default function SettingsPage() {
  return (
    <>
      <PageHeader title="Pengaturan" description="Uji coba dan saluran notifikasi." />
      <TrialSection />
      <TelegramSection />
    </>
  )
}
