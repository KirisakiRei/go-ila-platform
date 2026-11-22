import { useState, type FormEvent } from 'react'
import { useMutation } from '@tanstack/react-query'
import { motion, useReducedMotion } from 'framer-motion'
import { ArrowRight, CheckCircle2, Loader2 } from 'lucide-react'
import { Link } from 'react-router-dom'
import { ApiError } from '@/lib/api-error'
import { inviteRequestApi } from '@/lib/invite-api'
import { SocialLinksRow } from './SocialLinksRow'
import type { SocialLinks } from '@/types/system'

const SUBMITTED_STORAGE_KEY = 'ila_invite_request_submitted_v1'
const SUBMITTED_TTL_MS = 24 * 60 * 60 * 1000
const EASE_OUT_STRONG = [0.23, 1, 0.32, 1] as const

type SubmittedRecord = {
  name: string
  email: string
  submittedAt: number
}

function readSubmittedRecord(): SubmittedRecord | null {
  try {
    const raw = window.localStorage.getItem(SUBMITTED_STORAGE_KEY)
    if (!raw) return null
    const parsed = JSON.parse(raw) as Partial<SubmittedRecord>
    if (
      typeof parsed.email !== 'string' ||
      typeof parsed.name !== 'string' ||
      typeof parsed.submittedAt !== 'number'
    ) {
      return null
    }
    if (Date.now() - parsed.submittedAt > SUBMITTED_TTL_MS) {
      window.localStorage.removeItem(SUBMITTED_STORAGE_KEY)
      return null
    }
    return { name: parsed.name, email: parsed.email, submittedAt: parsed.submittedAt }
  } catch {
    return null
  }
}

function writeSubmittedRecord(record: SubmittedRecord) {
  try {
    window.localStorage.setItem(SUBMITTED_STORAGE_KEY, JSON.stringify(record))
  } catch {
    // Abaikan kegagalan storage (misal mode privat browser); proteksi utama tetap di backend.
  }
}

type InviteRequestSectionProps = {
  socialLinks?: SocialLinks | null
}

/**
 * InviteRequestSection adalah bagian CTA akhir beranda pada zona gelap
 * (menyatu langsung di bawah Hero 5 "Eksekusi cepat").
 *
 * Copy sengaja ringkas: pengunjung hanya mengisi identitas dasar (tanpa kata
 * sandi maupun kredensial broker) untuk masuk antrean undangan admin.
 */
export function InviteRequestSection({ socialLinks }: InviteRequestSectionProps) {
  const prefersReducedMotion = useReducedMotion() ?? false
  const [name, setName] = useState('')
  const [email, setEmail] = useState('')
  const [contact, setContact] = useState('')
  const [note, setNote] = useState('')
  const [website, setWebsite] = useState('')
  const [submittedRecord, setSubmittedRecord] = useState<SubmittedRecord | null>(() =>
    readSubmittedRecord(),
  )
  const [errorMessage, setErrorMessage] = useState<string | null>(null)

  function lockAsSubmitted(cleanName: string, cleanEmail: string) {
    const record: SubmittedRecord = {
      name: cleanName || 'Anda',
      email: cleanEmail,
      submittedAt: Date.now(),
    }
    writeSubmittedRecord(record)
    setSubmittedRecord(record)
    setErrorMessage(null)
  }

  const submitMutation = useMutation({
    mutationFn: () =>
      inviteRequestApi.submit({
        name: name.trim(),
        email: email.trim(),
        contact: contact.trim() || undefined,
        note: note.trim() || undefined,
        website,
      }),
    onSuccess: () => {
      lockAsSubmitted(name.trim(), email.trim().toLowerCase())
    },
    onError: (err) => {
      // Bila email sudah tercatat pending di server (409), langsung kunci UI sebagai sudah diajukan.
      if (err instanceof ApiError && err.status === 409) {
        lockAsSubmitted(name.trim(), email.trim().toLowerCase())
        return
      }
      setErrorMessage(
        err instanceof Error
          ? err.message
          : 'Permohonan belum berhasil dikirim. Silakan coba lagi sebentar lagi.',
      )
    },
  })

  function handleSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault()
    if (submittedRecord || submitMutation.isPending) return
    setErrorMessage(null)
    if (name.trim().length < 2) {
      setErrorMessage('Nama minimal 2 karakter.')
      return
    }
    if (!email.trim().includes('@')) {
      setErrorMessage('Masukkan alamat email yang valid.')
      return
    }
    submitMutation.mutate()
  }

  return (
    <section
      id="ajukan-akses"
      className="relative z-10 scroll-mt-20 px-4 pb-8 pt-16 sm:px-6 sm:pb-12 sm:pt-20"
    >
      <div className="mx-auto w-full max-w-6xl border-t border-white/10 pt-14 sm:pt-16">
        <div className="grid gap-10 lg:grid-cols-12 lg:items-start">
          {/* Kolom kiri: ringkas & bersih */}
          <div className="flex flex-col justify-between space-y-6 lg:col-span-5">
            <div className="space-y-3">
              <p className="font-mono text-[11px] font-medium uppercase tracking-[0.2em] text-zinc-400">
                Akses Undangan
              </p>
              <h2 className="text-3xl font-semibold leading-[1.12] tracking-tight sm:text-4xl">
                <span className="text-white">Bergabung lewat</span>{' '}
                <span className="text-zinc-400">undangan terbatas.</span>
              </h2>
              <p className="max-w-md text-[14px] leading-relaxed text-zinc-300">
                Pendaftaran mandiri ditutup. Ajukan akses di sini — tanpa kata sandi atau data
                broker.
              </p>
            </div>

            <div className="space-y-4">
              <SocialLinksRow links={socialLinks} variant="dark" showLabel />

              <a
                href="/app/login"
                className="inline-flex items-center gap-1.5 text-[13px] font-medium text-white underline underline-offset-4 transition-colors hover:text-zinc-300"
              >
                Sudah punya akun? Masuk
              </a>
            </div>

            <p className="max-w-md pt-2 text-[12px] leading-relaxed text-zinc-500">
              Perdagangan mata uang dan derivatif memiliki risiko kerugian tinggi. Alat bantu
              analisis, bukan nasihat keuangan.{' '}
              <Link
                to="/legal/risk"
                className="text-zinc-300 underline underline-offset-4 transition-colors hover:text-white"
              >
                Disclaimer Risiko
              </Link>
            </p>
          </div>

          {/* Kolom kanan: formulir bersih tanpa kredensial */}
          <div className="lg:col-span-7">
            <div className="rounded-2xl border border-white/10 bg-white/[0.04] p-5 sm:p-6">
              {submittedRecord ? (
                <motion.div
                  initial={prefersReducedMotion ? false : { opacity: 0, scale: 0.96, y: 6 }}
                  animate={{ opacity: 1, scale: 1, y: 0 }}
                  transition={{ duration: 0.24, ease: EASE_OUT_STRONG }}
                  className="flex flex-col items-start gap-4 py-4"
                >
                  <div className="flex h-10 w-10 items-center justify-center rounded-full border border-white/15 bg-white/10 text-white">
                    <CheckCircle2 className="h-5 w-5" aria-hidden="true" />
                  </div>
                  <div className="space-y-1.5">
                    <h3 className="text-lg font-semibold tracking-tight text-white">
                      Permohonan akses diterima
                    </h3>
                    <p className="text-[13px] leading-relaxed text-zinc-300">
                      Terima kasih, <strong className="text-white">{submittedRecord.name}</strong>.
                      Permohonan untuk{' '}
                      <span className="break-all font-mono text-zinc-200">
                        {submittedRecord.email}
                      </span>{' '}
                      sudah masuk antrean tinjauan. Tautan undangan akan dikirim ke email Anda bila
                      disetujui.
                    </p>
                  </div>
                  <button
                    type="button"
                    disabled
                    className="inline-flex w-full cursor-default items-center justify-center gap-2 rounded-lg border border-white/15 bg-white/10 px-4 py-2 text-[13px] font-medium text-zinc-200 sm:w-auto"
                  >
                    <CheckCircle2 className="h-3.5 w-3.5 text-white" aria-hidden="true" />
                    Sudah Diajukan
                  </button>
                </motion.div>
              ) : (
                <form onSubmit={handleSubmit} className="space-y-4" noValidate>
                  <input
                    type="text"
                    name="website"
                    value={website}
                    onChange={(e) => setWebsite(e.target.value)}
                    tabIndex={-1}
                    autoComplete="off"
                    aria-hidden="true"
                    className="hidden"
                  />

                  <div className="grid gap-4 sm:grid-cols-2">
                    <div className="space-y-1.5">
                      <label
                        htmlFor="invite-req-name"
                        className="block text-[12px] font-medium text-zinc-200"
                      >
                        Nama / Panggilan <span className="text-zinc-500">*</span>
                      </label>
                      <input
                        id="invite-req-name"
                        type="text"
                        required
                        maxLength={100}
                        value={name}
                        onChange={(e) => setName(e.target.value)}
                        placeholder="Nama Anda"
                        className="w-full rounded-lg border border-white/15 bg-black/40 px-3.5 py-2.5 text-[13px] text-white placeholder:text-zinc-500 transition-colors focus:border-white/40 focus:outline-none"
                      />
                    </div>

                    <div className="space-y-1.5">
                      <label
                        htmlFor="invite-req-email"
                        className="block text-[12px] font-medium text-zinc-200"
                      >
                        Email <span className="text-zinc-500">*</span>
                      </label>
                      <input
                        id="invite-req-email"
                        type="email"
                        required
                        maxLength={150}
                        value={email}
                        onChange={(e) => setEmail(e.target.value)}
                        placeholder="nama@email.com"
                        className="w-full rounded-lg border border-white/15 bg-black/40 px-3.5 py-2.5 text-[13px] text-white placeholder:text-zinc-500 transition-colors focus:border-white/40 focus:outline-none"
                      />
                    </div>
                  </div>

                  <div className="space-y-1.5">
                    <label
                      htmlFor="invite-req-contact"
                      className="block text-[12px] font-medium text-zinc-200"
                    >
                      Telegram / Discord <span className="text-zinc-500">(opsional)</span>
                    </label>
                    <input
                      id="invite-req-contact"
                      type="text"
                      maxLength={100}
                      value={contact}
                      onChange={(e) => setContact(e.target.value)}
                      placeholder="@username"
                      className="w-full rounded-lg border border-white/15 bg-black/40 px-3.5 py-2.5 text-[13px] text-white placeholder:text-zinc-500 transition-colors focus:border-white/40 focus:outline-none"
                    />
                  </div>

                  <div className="space-y-1.5">
                    <label
                      htmlFor="invite-req-note"
                      className="block text-[12px] font-medium text-zinc-200"
                    >
                      Catatan singkat <span className="text-zinc-500">(opsional)</span>
                    </label>
                    <textarea
                      id="invite-req-note"
                      rows={2}
                      maxLength={500}
                      value={note}
                      onChange={(e) => setNote(e.target.value)}
                      placeholder="Pair fokus atau gaya trading Anda"
                      className="w-full resize-none rounded-lg border border-white/15 bg-black/40 px-3.5 py-2.5 text-[13px] text-white placeholder:text-zinc-500 transition-colors focus:border-white/40 focus:outline-none"
                    />
                  </div>

                  {errorMessage && (
                    <p
                      role="alert"
                      className="rounded-lg border border-red-400/30 bg-red-500/10 px-3.5 py-2 text-[12px] text-red-200"
                    >
                      {errorMessage}
                    </p>
                  )}

                  <div className="flex justify-end pt-1">
                    <button
                      type="submit"
                      disabled={submitMutation.isPending}
                      className="inline-flex w-full items-center justify-center gap-2 rounded-lg bg-white px-5 py-2.5 text-[13px] font-semibold text-[#09090b] press hover:bg-zinc-200 disabled:cursor-not-allowed disabled:opacity-60 sm:w-auto"
                    >
                      {submitMutation.isPending ? (
                        <>
                          <Loader2 className="h-3.5 w-3.5 animate-spin" aria-hidden="true" />
                          Mengirim…
                        </>
                      ) : (
                        <>
                          Ajukan Bergabung
                          <ArrowRight className="h-3.5 w-3.5" aria-hidden="true" />
                        </>
                      )}
                    </button>
                  </div>
                </form>
              )}
            </div>
          </div>
        </div>
      </div>
    </section>
  )
}
