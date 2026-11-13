import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import {
  ExternalLink,
  HelpCircle,
  KeyRound,
  ShieldCheck,
  Sparkles,
} from 'lucide-react'
import { Alert, Button, Input, Label, Select } from '@/components/ui'
import { accountApi, legalApi } from '@/lib/api'
import { ApiError } from '@/lib/api-error'
import { useCreateAccount } from '@/lib/queries'
import type { Tier } from '@/types/api'

const matchTraderSchema = z.object({
  label: z.string().min(1, 'Nama akun wajib diisi').max(100, 'Maksimal 100 karakter'),
  token: z.string().min(20, 'Access token minimal 20 karakter'),
  mode: z.enum(['manual', 'auto']),
  risk_percent: z.coerce.number().min(0.1, 'Minimal 0,10%').max(5, 'Maksimal 5,00%'),
  is_dry_run: z.boolean(),
})

type MatchTraderFormValues = z.input<typeof matchTraderSchema>

/**
 * AccountForm melayani 2 platform broker:
 * 1. cTrader (OAuth 1-klik via Spotware ID)
 * 2. Match-Trader (Access Token terenkripsi + Panduan)
 */
export function AccountForm({ tier, onDone }: { tier: Tier; onDone?: () => void }) {
  const [activeTab, setActiveTab] = useState<'ctrader' | 'matchtrader'>('ctrader')
  const [submitError, setSubmitError] = useState<string | null>(null)
  const [ctraderLoading, setCtraderLoading] = useState(false)
  const [showGuide, setShowGuide] = useState(true)

  // Status persetujuan dokumen legal (wajib Disclaimer Risiko sebelum akun broker terhubung)
  const queryClient = useQueryClient()
  const legalQuery = useQuery({
    queryKey: ['me', 'legal'],
    queryFn: legalApi.me,
    staleTime: 60_000,
  })

  const isRiskAccepted = Boolean(
    legalQuery.data?.documents?.find((doc) => doc.slug === 'risk')?.accepted,
  )
  const [acceptRisk, setAcceptRisk] = useState(false)

  // Pengaturan default untuk cTrader OAuth
  const [ctraderMode, setCtraderMode] = useState<'manual' | 'auto'>('manual')
  const [ctraderRisk, setCtraderRisk] = useState<number>(0.5)

  const createAccount = useCreateAccount()

  const {
    register,
    handleSubmit,
    reset,
    formState: { errors, isSubmitting },
  } = useForm<MatchTraderFormValues>({
    resolver: zodResolver(matchTraderSchema),
    defaultValues: {
      label: '',
      token: '',
      mode: 'manual',
      risk_percent: 0.5,
      is_dry_run: true,
    },
  })

  // Alur 1: cTrader OAuth
  const handleConnectCTrader = async () => {
    setSubmitError(null)

    if (!isRiskAccepted && !acceptRisk) {
      setSubmitError('Harap centang persetujuan Disclaimer Risiko Trading sebelum menghubungkan akun broker.')
      return
    }

    setCtraderLoading(true)
    try {
      if (!isRiskAccepted) {
        await legalApi.accept('risk', 0)
        queryClient.invalidateQueries({ queryKey: ['me', 'legal'] })
      }

      const res = await accountApi.ctraderAuthUrl()
      if (!res.auth_url) {
        throw new Error('Gagal mendapatkan tautan otorisasi cTrader dari server.')
      }

      // Simpan preferensi sementara agar dikirim saat callback kembali
      sessionStorage.setItem(
        'ctrader_pending_settings',
        JSON.stringify({
          mode: ctraderMode,
          risk_percent: ctraderRisk,
        }),
      )

      // Arahkan browser ke cTrader OAuth Spotware
      window.location.href = res.auth_url
    } catch (err) {
      setCtraderLoading(false)
      setSubmitError(err instanceof Error ? err.message : 'Gagal menghubungkan ke cTrader.')
    }
  }

  // Alur 2: Match-Trader Token
  const onSubmitMatchTrader = handleSubmit(async (values) => {
    setSubmitError(null)

    if (!isRiskAccepted && !acceptRisk) {
      setSubmitError('Harap centang persetujuan Disclaimer Risiko Trading sebelum menghubungkan akun broker.')
      return
    }

    try {
      if (!isRiskAccepted) {
        await legalApi.accept('risk', 0)
        queryClient.invalidateQueries({ queryKey: ['me', 'legal'] })
      }

      await createAccount.mutateAsync({
        broker_type: 'matchtrader',
        label: values.label,
        mode: values.mode,
        risk_percent: Number(values.risk_percent),
        is_dry_run: Boolean(values.is_dry_run),
        token: values.token,
        base_url: '',
      })
      reset()
      onDone?.()
    } catch (error) {
      // Bila server masih menolak karena risk belum tercatat, coba terima sekali lagi lalu buat akun
      if (error instanceof ApiError && error.message.toLowerCase().includes('disclaimer risiko')) {
        try {
          await legalApi.accept('risk', 0)
          queryClient.invalidateQueries({ queryKey: ['me', 'legal'] })
          await createAccount.mutateAsync({
            broker_type: 'matchtrader',
            label: values.label,
            mode: values.mode,
            risk_percent: Number(values.risk_percent),
            is_dry_run: Boolean(values.is_dry_run),
            token: values.token,
            base_url: '',
          })
          reset()
          onDone?.()
          return
        } catch (retryErr) {
          setSubmitError(retryErr instanceof ApiError ? retryErr.message : 'Gagal menyimpan akun Match-Trader.')
          return
        }
      }
      setSubmitError(error instanceof ApiError ? error.message : 'Gagal menyimpan akun Match-Trader.')
    }
  })

  return (
    <div className="space-y-4">
      {/* Tab Selector Platform Broker */}
      <div className="flex rounded-xl border border-border p-1 bg-secondary/30">
        <button
          type="button"
          onClick={() => {
            setActiveTab('ctrader')
            setSubmitError(null)
          }}
          className={`flex flex-1 items-center justify-center gap-2 rounded-lg py-2 text-xs font-semibold transition-all ${
            activeTab === 'ctrader'
              ? 'bg-foreground text-background shadow-sm'
              : 'text-muted-foreground hover:text-foreground'
          }`}
        >
          <Sparkles className="h-3.5 w-3.5" />
          <span>cTrader</span>
          <span className="rounded-full bg-emerald-500/20 px-1.5 py-0.2 text-[10px] font-bold text-emerald-400">
            OAuth
          </span>
        </button>

        <button
          type="button"
          onClick={() => {
            setActiveTab('matchtrader')
            setSubmitError(null)
          }}
          className={`flex flex-1 items-center justify-center gap-2 rounded-lg py-2 text-xs font-semibold transition-all ${
            activeTab === 'matchtrader'
              ? 'bg-foreground text-background shadow-sm'
              : 'text-muted-foreground hover:text-foreground'
          }`}
        >
          <KeyRound className="h-3.5 w-3.5" />
          <span>Match-Trader</span>
          <span className="rounded-full bg-zinc-500/20 px-1.5 py-0.2 text-[10px] font-medium text-muted-foreground">
            Token
          </span>
        </button>
      </div>

      {submitError && <Alert variant="destructive">{submitError}</Alert>}

      {/* --- KONTEN TAB 1: CTRADER (OAUTH) --- */}
      {activeTab === 'ctrader' && (
        <div className="space-y-4 text-xs">
          <div className="rounded-xl border border-border bg-secondary/20 p-4 space-y-2">
            <div className="flex items-center gap-2 font-semibold text-foreground">
              <ShieldCheck className="h-4 w-4 text-emerald-600 dark:text-emerald-400" />
              <span>Koneksi Resmi & Aman via Spotware cTrader ID</span>
            </div>
            <p className="text-muted-foreground leading-relaxed">
              Anda tidak perlu menyalin token atau memasukkan kata sandi. Cukup otorisasi akses menggunakan akun Spotware cTrader Anda dalam 1 kali klik. Seluruh akun trading di bawah cTrader ID Anda akan otomatis tersambung.
            </p>
          </div>

          <div className="grid gap-3 sm:grid-cols-2">
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="ctrader-mode">Mode Eksekusi Awal</Label>
              <Select
                id="ctrader-mode"
                value={ctraderMode}
                onChange={(e) => setCtraderMode(e.target.value as 'manual' | 'auto')}
                disabled={tier !== 'plus'}
              >
                <option value="manual">Manual (Perlu Konfirmasi)</option>
                <option value="auto">Otomatis (Langsung Eksekusi)</option>
              </Select>
              {tier !== 'plus' && (
                <p className="text-[11px] text-muted-foreground">Otomatis hanya untuk tier Plus.</p>
              )}
            </div>

            <div className="flex flex-col gap-1.5">
              <Label htmlFor="ctrader-risk">Batas Risiko per Transaksi (%)</Label>
              <Input
                id="ctrader-risk"
                type="number"
                step="0.05"
                min="0.1"
                max="1"
                value={ctraderRisk}
                onChange={(e) => setCtraderRisk(Number(e.target.value))}
              />
              <p className="text-[11px] text-muted-foreground">Rentang aman: 0,10% – 1,00%.</p>
            </div>
          </div>

          {/* Kotak Persetujuan Disclaimer Risiko Trading (Wajib sebelum koneksi akun broker) */}
          <div className="rounded-xl border border-border/80 bg-secondary/15 p-3 space-y-1.5">
            <label className="flex items-start gap-2.5 cursor-pointer">
              <input
                type="checkbox"
                checked={isRiskAccepted || acceptRisk}
                disabled={isRiskAccepted}
                onChange={(e) => {
                  setAcceptRisk(e.target.checked)
                  if (e.target.checked) setSubmitError(null)
                }}
                className="mt-0.5 h-4 w-4 shrink-0 rounded accent-foreground"
              />
              <span className="text-[11.5px] leading-relaxed text-foreground">
                Saya memahami risiko perdagangan derivatif dan menyetujui{' '}
                <a
                  href="/legal/risk"
                  target="_blank"
                  rel="noreferrer"
                  className="font-semibold underline underline-offset-2 hover:text-primary inline-flex items-center gap-0.5"
                  onClick={(e) => e.stopPropagation()}
                >
                  Disclaimer Risiko Trading
                  <ExternalLink className="h-3 w-3" />
                </a>
                .
                {isRiskAccepted && (
                  <span className="ml-1 text-[10.5px] font-medium text-emerald-600 dark:text-emerald-400">
                    (Sudah disetujui)
                  </span>
                )}
              </span>
            </label>
          </div>

          <div className="pt-2">
            <Button
              className="w-full gap-2 font-semibold shadow-sm"
              size="lg"
              disabled={ctraderLoading}
              isLoading={ctraderLoading}
              onClick={handleConnectCTrader}
            >
              <span>Hubungkan via cTrader ID (OAuth)</span>
              <ExternalLink className="h-4 w-4" />
            </Button>
            <p className="mt-2 text-center text-[11px] text-muted-foreground">
              Anda akan diarahkan ke laman login resmi Spotware cTrader ID.
            </p>
          </div>
        </div>
      )}

      {/* --- KONTEN TAB 2: MATCH-TRADER (TOKEN) --- */}
      {activeTab === 'matchtrader' && (
        <form onSubmit={onSubmitMatchTrader} className="flex flex-col gap-3.5 text-xs" noValidate>
          {/* Panduan Pengambilan Token */}
          <div className="rounded-xl border border-border bg-secondary/20 p-3.5 space-y-2">
            <button
              type="button"
              onClick={() => setShowGuide(!showGuide)}
              className="flex w-full items-center justify-between text-left font-semibold text-foreground hover:underline"
            >
              <span className="flex items-center gap-1.5">
                <HelpCircle className="h-3.5 w-3.5 text-foreground" />
                Cara Mendapatkan Access Token Match-Trader
              </span>
              <span className="text-[11px] text-muted-foreground font-normal">
                {showGuide ? 'Sembunyikan' : 'Lihat Panduan'}
              </span>
            </button>

            {showGuide && (
              <ol className="list-decimal space-y-1 pl-4 pt-1 text-[11.5px] leading-relaxed text-muted-foreground">
                <li>Buka aplikasi atau portal web platform <strong>Match-Trader</strong> broker Anda.</li>
                <li>Masuk ke menu <strong>Profil / Pengaturan Akun</strong> di pojok kanan atas.</li>
                <li>Cari tab <strong>API Access / Token</strong>, lalu buat atau salin token akses Anda.</li>
                <li>Tempelkan token ke kolom formulir di bawah ini.</li>
              </ol>
            )}
          </div>

          <div className="flex flex-col gap-1.5">
            <Label htmlFor="label">Nama Label Akun</Label>
            <Input id="label" placeholder="Contoh: Akun Utama Match-Trader" {...register('label')} />
            {errors.label && <p className="text-[12px] text-destructive">{errors.label.message}</p>}
          </div>

          <div className="flex flex-col gap-1.5">
            <Label htmlFor="token">Access Token Match-Trader</Label>
            <Input
              id="token"
              type="password"
              autoComplete="off"
              placeholder="Tempel token akses Match-Trader di sini"
              className="font-mono"
              {...register('token')}
            />
            {errors.token && <p className="text-[12px] text-destructive">{errors.token.message}</p>}
            <p className="text-[11px] text-muted-foreground">
              Token dienkripsi di server dan tidak dapat ditampilkan kembali.
            </p>
          </div>

          <div className="grid gap-3 sm:grid-cols-2">
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="mode">Mode Eksekusi</Label>
              <Select id="mode" {...register('mode')} disabled={tier !== 'plus'}>
                <option value="manual">Manual (Perlu Konfirmasi)</option>
                <option value="auto">Otomatis (Langsung Eksekusi)</option>
              </Select>
              {tier !== 'plus' && (
                <p className="text-[11px] text-muted-foreground">Otomatis hanya untuk tier Plus.</p>
              )}
            </div>

            <div className="flex flex-col gap-1.5">
              <Label htmlFor="risk_percent">Risiko per Trade (%)</Label>
              <Input
                id="risk_percent"
                type="number"
                step="0.05"
                min="0.1"
                max="1"
                {...register('risk_percent')}
              />
              {errors.risk_percent && (
                <p className="text-[12px] text-destructive">{errors.risk_percent.message}</p>
              )}
            </div>
          </div>

          <label className="flex items-center gap-2 text-xs">
            <input type="checkbox" className="h-4 w-4 accent-foreground" {...register('is_dry_run')} />
            <span>Jalankan sebagai simulasi (tidak mengirim order nyata ke broker)</span>
          </label>

          {/* Kotak Persetujuan Disclaimer Risiko Trading (Wajib sebelum koneksi akun broker) */}
          <div className="rounded-xl border border-border/80 bg-secondary/15 p-3 space-y-1.5">
            <label className="flex items-start gap-2.5 cursor-pointer">
              <input
                type="checkbox"
                checked={isRiskAccepted || acceptRisk}
                disabled={isRiskAccepted}
                onChange={(e) => {
                  setAcceptRisk(e.target.checked)
                  if (e.target.checked) setSubmitError(null)
                }}
                className="mt-0.5 h-4 w-4 shrink-0 rounded accent-foreground"
              />
              <span className="text-[11.5px] leading-relaxed text-foreground">
                Saya memahami risiko perdagangan derivatif dan menyetujui{' '}
                <a
                  href="/legal/risk"
                  target="_blank"
                  rel="noreferrer"
                  className="font-semibold underline underline-offset-2 hover:text-primary inline-flex items-center gap-0.5"
                  onClick={(e) => e.stopPropagation()}
                >
                  Disclaimer Risiko Trading
                  <ExternalLink className="h-3 w-3" />
                </a>
                .
                {isRiskAccepted && (
                  <span className="ml-1 text-[10.5px] font-medium text-emerald-600 dark:text-emerald-400">
                    (Sudah disetujui)
                  </span>
                )}
              </span>
            </label>
          </div>

          <Button type="submit" size="lg" className="w-full mt-1" isLoading={isSubmitting}>
            Hubungkan Akun Match-Trader
          </Button>
        </form>
      )}
    </div>
  )
}
