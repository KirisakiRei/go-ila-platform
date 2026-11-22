import { useState, type FormEvent } from 'react'
import { useMutation } from '@tanstack/react-query'
import { ArrowRight, CheckCircle2, Loader2 } from 'lucide-react'
import { Link } from 'react-router-dom'
import { inviteRequestApi } from '@/lib/invite-api'
import { SocialLinksRow } from './SocialLinksRow'
import type { SocialLinks } from '@/types/system'

type InviteRequestSectionProps = {
  socialLinks?: SocialLinks | null
}

/**
 * InviteRequestSection adalah kartu CTA akhir beranda untuk mode invite-only.
 *
 * Pengunjung cukup mengirim identitas dasar (tanpa kata sandi maupun kredensial
 * broker). Permohonan masuk ke panel admin agar tautan undangan dapat dibuat
 * dan dikirimkan setelah peninjauan.
 */
export function InviteRequestSection({ socialLinks }: InviteRequestSectionProps) {
  const [name, setName] = useState('')
  const [email, setEmail] = useState('')
  const [contact, setContact] = useState('')
  const [note, setNote] = useState('')
  // Field jebakan bot (honeypot): disembunyikan dari manusia dan pembaca layar.
  const [website, setWebsite] = useState('')
  const [submitted, setSubmitted] = useState(false)
  const [errorMessage, setErrorMessage] = useState<string | null>(null)

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
      setErrorMessage(null)
      setSubmitted(true)
    },
    onError: (err) => {
      setErrorMessage(
        err instanceof Error
          ? err.message
          : 'Permohonan belum berhasil dikirim. Silakan coba lagi sebentar lagi.',
      )
    },
  })

  function handleSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault()
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
      className="relative z-10 scroll-mt-20 bg-transparent px-4 pb-2 pt-10 sm:px-6 sm:pb-4 sm:pt-12"
    >
      <div className="mx-auto w-full max-w-6xl">
        <div className="rounded-3xl bg-[#050507] px-6 py-10 text-white shadow-2xl shadow-black/15 sm:px-10 sm:py-14 lg:px-12">
          <div className="grid gap-10 lg:grid-cols-12 lg:items-start">
            {/* Kolom kiri: penjelasan invite-only, sosmed, dan disclaimer */}
            <div className="space-y-6 lg:col-span-5">
              <div className="space-y-3">
                <p className="text-[11px] font-semibold uppercase tracking-[0.16em] text-zinc-400">
                  Akses Undangan
                </p>
                <h2 className="text-3xl font-semibold leading-[1.12] tracking-tight sm:text-4xl">
                  <span className="text-white">Bergabung lewat</span>{' '}
                  <span className="text-zinc-400">undangan terbatas.</span>
                </h2>
                <p className="text-[14px] leading-relaxed text-zinc-300">
                  Saat ini pendaftaran mandiri masih ditutup. Isi formulir singkat di samping —
                  tanpa kata sandi maupun kredensial broker — dan tim kami akan meninjau sebelum
                  mengirimkan tautan undangan pribadi Anda.
                </p>
              </div>

              <SocialLinksRow links={socialLinks} variant="dark" showLabel />

              <p className="border-l-2 border-zinc-700 pl-4 text-[12px] leading-relaxed text-zinc-300">
                <strong className="font-semibold text-white">Disclaimer Risiko Trading:</strong>{' '}
                Perdagangan mata uang dan derivatif memiliki risiko kerugian tinggi. Platform ini
                adalah alat bantu analisis dan eksekusi. Tidak menjamin hasil, bukan nasihat
                keuangan.
              </p>

              <div className="flex flex-wrap items-center gap-5 pt-1 text-[13px]">
                <a
                  href="/app/login"
                  className="inline-flex items-center gap-1.5 font-medium text-white underline underline-offset-4 transition-colors hover:text-zinc-300"
                >
                  Sudah punya akun? Masuk
                </a>
                <Link
                  to="/legal/risk"
                  className="font-medium text-zinc-300 underline underline-offset-4 transition-colors hover:text-white"
                >
                  Baca Disclaimer Risiko
                </Link>
              </div>
            </div>

            {/* Kolom kanan: formulir permohonan akses tanpa kredensial */}
            <div className="lg:col-span-7">
              <div className="rounded-2xl border border-white/10 bg-white/[0.04] p-5 sm:p-6">
                {submitted ? (
                  <div className="flex flex-col items-start gap-4 py-4">
                    <div className="flex h-10 w-10 items-center justify-center rounded-full border border-white/15 bg-white/10 text-white">
                      <CheckCircle2 className="h-5 w-5" aria-hidden="true" />
                    </div>
                    <div className="space-y-1.5">
                      <h3 className="text-lg font-semibold tracking-tight text-white">
                        Permohonan akses diterima
                      </h3>
                      <p className="text-[13px] leading-relaxed text-zinc-300">
                        Terima kasih, <strong className="text-white">{name}</strong>. Permohonan
                        untuk <span className="font-mono text-zinc-200">{email}</span> sudah masuk
                        ke antrean tinjauan kami. Jika disetujui, tautan undangan akan dikirimkan
                        ke alamat email tersebut.
                      </p>
                    </div>
                    <button
                      type="button"
                      onClick={() => {
                        setSubmitted(false)
                        setName('')
                        setEmail('')
                        setContact('')
                        setNote('')
                      }}
                      className="text-[12px] font-medium text-zinc-300 underline underline-offset-4 hover:text-white"
                    >
                      Kirim permohonan lain
                    </button>
                  </div>
                ) : (
                  <form onSubmit={handleSubmit} className="space-y-4" noValidate>
                    {/* Honeypot */}
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
                          Nama / Panggilan <span className="text-zinc-400">*</span>
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
                          Email <span className="text-zinc-400">*</span>
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
                        placeholder="@username Telegram atau Discord"
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
                        placeholder="Mis. pair utama yang dipantau (XAUUSD, EURUSD, NAS100) atau gaya trading Anda"
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

                    <div className="flex flex-col gap-3 pt-1 sm:flex-row sm:items-center sm:justify-between">
                      <p className="text-[11px] leading-relaxed text-zinc-400">
                        Kami tidak pernah meminta kata sandi atau kredensial akun broker di sini.
                      </p>
                      <button
                        type="submit"
                        disabled={submitMutation.isPending}
                        className="inline-flex shrink-0 items-center justify-center gap-2 rounded-lg bg-white px-5 py-2.5 text-[13px] font-semibold text-[#09090b] press hover:bg-zinc-200 disabled:cursor-not-allowed disabled:opacity-60"
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
      </div>
    </section>
  )
}
