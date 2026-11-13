import { useEffect, useMemo, useRef, useState } from 'react'
import { Link, useLocation } from 'react-router-dom'
import { FileText, ImageUp, Paperclip, X } from 'lucide-react'
import { Alert, Button, Card, Input, Label, Select } from '@/components/ui'
import { ApiError } from '@/lib/api-error'
import { cn } from '@/lib/utils'
import {
  BUG_MAX_LAMPIRAN_BYTE,
  BUG_MAKS_LAMPIRAN_MB,
  useBugCategories,
  useKirimLaporanBug,
} from '@/lib/bug-api'

/** Label ramah untuk kategori yang datang dari server. */
const LABEL_KATEGORI: Record<string, string> = {
  tampilan: 'Tampilan / halaman rusak',
  sinyal: 'Sinyal tidak muncul atau salah',
  eksekusi: 'Eksekusi order',
  'akun-broker': 'Akun broker',
  login: 'Masuk / kata sandi',
  kinerja: 'Lambat / error',
  lainnya: 'Lainnya',
}

/**
 * Tipe yang diterima. Server tetap memeriksa magic bytes, jadi daftar ini
 * semata-mata untuk umpan balik cepat agar pengguna tidak menunggu unggahan
 * yang pasti ditolak.
 */
const TIPE_DIIZINKAN = ['image/png', 'image/jpeg', 'image/webp', 'image/gif', 'application/pdf']

function labelUkuran(byte: number): string {
  if (byte < 1024) return `${byte} B`
  return `${Math.ceil(byte / 1024)} KB`
}

/**
 * LaporkanBugPage adalah formulir laporan bug.
 *
 * Satu layar, bukan langkah bertahap: formulirnya pendek, dan langkah-langkah
 * hanya menambah gesekan pada hal yang sedang tidak menyenangkan bagi pengguna.
 * Lampiran menerima tiga jalur masuk (pilih, tarik-lepas, tempel) karena
 * tangkapan layar bug biasanya sudah ada di papan klip sebelum halaman dibuka.
 */
export default function LaporkanBugPage() {
  const location = useLocation()
  const kategori = useBugCategories()
  const kirim = useKirimLaporanBug()

  const [category, setCategory] = useState('')
  const [title, setTitle] = useState('')
  const [body, setBody] = useState('')
  const [berkas, setBerkas] = useState<File | null>(null)
  const [galat, setGalat] = useState<string | null>(null)
  const [terkirim, setTerkirim] = useState(false)
  const [menarik, setMenarik] = useState(false)

  const inputBerkasRef = useRef<HTMLInputElement>(null)
  // Penghitung kedalaman drag: dragenter/leave ikut terpicu oleh elemen anak,
  // tanpa penghitung ini sorotan berkedip saat kursor melewati teks di dalam zona.
  const kedalamanDrag = useRef(0)

  const pratinjau = usePratinjauLampiran(berkas)

  function pilihBerkas(file: File | null) {
    setGalat(null)
    if (!file) {
      setBerkas(null)
      return
    }
    // Server memutuskan tipe dari isi berkas, bukan dari header klien; cek di
    // sini hanya untuk menghemat waktu pengguna.
    if (!TIPE_DIIZINKAN.includes(file.type)) {
      setBerkas(null)
      setGalat('Lampiran harus berupa gambar (png, jpg, webp, gif) atau PDF.')
      return
    }
    if (file.size > BUG_MAX_LAMPIRAN_BYTE) {
      setBerkas(null)
      setGalat(`Ukuran lampiran maksimal ${BUG_MAKS_LAMPIRAN_MB} MB.`)
      return
    }
    setBerkas(file)
  }

  function ambilDariDaftar(daftar: FileList | null | undefined) {
    const semua = daftar ? Array.from(daftar) : []
    if (semua.length === 0) return
    pilihBerkas(semua[0] ?? null)
    if (semua.length > 1) {
      setGalat('Hanya satu lampiran yang bisa disertakan. Berkas pertama dipakai.')
    }
  }

  function onDrop(e: React.DragEvent<HTMLDivElement>) {
    e.preventDefault()
    kedalamanDrag.current = 0
    setMenarik(false)
    ambilDariDaftar(e.dataTransfer?.files)
  }

  function onDragOver(e: React.DragEvent<HTMLDivElement>) {
    // Tanpa ini peramban membuka berkasnya dan pengguna meninggalkan halaman.
    e.preventDefault()
    if (e.dataTransfer) e.dataTransfer.dropEffect = 'copy'
  }

  function onDragEnter(e: React.DragEvent<HTMLDivElement>) {
    e.preventDefault()
    kedalamanDrag.current += 1
    setMenarik(true)
  }

  function onDragLeave(e: React.DragEvent<HTMLDivElement>) {
    e.preventDefault()
    kedalamanDrag.current = Math.max(0, kedalamanDrag.current - 1)
    if (kedalamanDrag.current === 0) setMenarik(false)
  }

  function onPaste(e: React.ClipboardEvent<HTMLFormElement>) {
    const berkasDariPapan = berkasDariClipboard(e.clipboardData)
    // Hanya cegah perilaku bawaan bila memang ada berkas. Menahan semua paste
    // akan mematikan tempel teks di kolom judul dan kronologi bug.
    if (berkasDariPapan.length === 0) return
    e.preventDefault()
    pilihBerkas(berkasDariPapan[0] ?? null)
    if (berkasDariPapan.length > 1) {
      setGalat('Hanya satu lampiran yang bisa disertakan. Berkas pertama dipakai.')
    }
  }

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault()
    setGalat(null)

    if (!category) return setGalat('Pilih kategori laporan lebih dulu.')
    if (title.trim().length < 3) return setGalat('Judul minimal 3 karakter.')
    if (body.trim().length < 10) return setGalat('Ceritakan bugnya minimal 10 karakter.')

    try {
      await kirim.mutateAsync({
        category,
        title: title.trim(),
        body: body.trim(),
        pageUrl: window.location.origin + location.pathname,
        attachment: berkas,
      })
      setTerkirim(true)
    } catch (err) {
      setGalat(err instanceof ApiError ? err.message : 'Laporan gagal dikirim. Coba lagi.')
    }
  }

  if (terkirim) {
    return (
      <Card className="flex flex-col gap-3 p-5">
        <h1 className="text-base font-bold tracking-tight text-foreground">Laporan terkirim</h1>
        <p className="text-sm text-muted-foreground">
          Terima kasih. Tim kami akan meninjau laporan ini. Statusnya bisa dilihat di riwayat
          laporan.
        </p>
        <div className="flex flex-wrap gap-2">
          <Button asChild size="sm">
            <Link to="/laporkan-bug/riwayat">Lihat riwayat</Link>
          </Button>
          <Button
            asChild
            size="sm"
            variant="outline"
            onClick={() => {
              setTerkirim(false)
              setCategory('')
              setTitle('')
              setBody('')
              setBerkas(null)
            }}
          >
            <Link to="/laporkan-bug">Lapor lagi</Link>
          </Button>
        </div>
      </Card>
    )
  }

  return (
    <div className="flex flex-col gap-3.5 pb-2">
      <div className="flex flex-wrap items-start justify-between gap-2">
        <div>
          <h1 className="text-lg font-bold tracking-tight text-foreground">Laporkan Bug</h1>
          <p className="mt-0.5 text-caption leading-relaxed text-muted-foreground">
            Ada yang tidak berjalan semestinya? Ceritakan di sini.
          </p>
        </div>
        <Button asChild size="sm" variant="outline">
          <Link to="/laporkan-bug/riwayat">Riwayat laporan</Link>
        </Button>
      </div>

      <Card className="p-4">
        <form onSubmit={onSubmit} onPaste={onPaste} noValidate className="flex flex-col gap-3.5">
          {galat && <Alert>{galat}</Alert>}

          <div className="flex flex-col gap-1">
            <Label htmlFor="kategori-bug">Kategori</Label>
            <Select
              id="kategori-bug"
              value={category}
              onChange={(e) => setCategory(e.target.value)}
              required
            >
              <option value="">Pilih kategori</option>
              {(kategori.data ?? []).map((k) => (
                <option key={k} value={k}>
                  {LABEL_KATEGORI[k] ?? k}
                </option>
              ))}
            </Select>
          </div>

          <div className="flex flex-col gap-1">
            <Label htmlFor="judul-bug">Judul singkat</Label>
            <Input
              id="judul-bug"
              value={title}
              maxLength={120}
              placeholder="misal: tombol simpan tidak muncul"
              onChange={(e) => setTitle(e.target.value)}
              required
            />
          </div>

          <div className="flex flex-col gap-1">
            <Label htmlFor="isi-bug">Apa yang terjadi?</Label>
            <textarea
              id="isi-bug"
              value={body}
              maxLength={5000}
              rows={6}
              placeholder="Langkah yang Anda lakukan, yang seharusnya terjadi, dan yang benar-benar terjadi."
              onChange={(e) => setBody(e.target.value)}
              className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
              required
            />
          </div>

          <div className="flex flex-col gap-1.5">
            <Label htmlFor="lampiran-bug">
              Lampiran
              <span className="font-normal text-muted-foreground">(opsional)</span>
            </Label>

            <div
              onDrop={onDrop}
              onDragOver={onDragOver}
              onDragEnter={onDragEnter}
              onDragLeave={onDragLeave}
              onClick={() => inputBerkasRef.current?.click()}
              onKeyDown={(e) => {
                if (e.key === 'Enter' || e.key === ' ') {
                  e.preventDefault()
                  inputBerkasRef.current?.click()
                }
              }}
              role="button"
              tabIndex={0}
              aria-label="Pilih, tarik-lepas, atau tempel berkas"
              aria-describedby="petunjuk-lampiran"
              className={cn(
                'flex cursor-pointer flex-col items-center gap-1.5 border border-dashed px-3 py-4 text-center transition-colors',
                'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background',
                menarik
                  ? 'border-foreground bg-secondary'
                  : 'border-border bg-transparent hover:border-foreground/40 hover:bg-secondary/60',
              )}
            >
              <ImageUp className="h-5 w-5 text-muted-foreground" aria-hidden="true" />
              <p className="text-[12.5px] font-medium text-foreground">
                {menarik ? 'Lepaskan berkas di sini' : 'Tarik gambar ke sini, atau klik untuk pilih'}
              </p>
              <p className="flex items-center gap-1 text-micro text-muted-foreground">
                <Paperclip className="h-3 w-3" aria-hidden="true" />
                Bisa juga Ctrl+V untuk menempel tangkapan layar
              </p>
            </div>

            {/* Input asli dipertahankan agar tetap bisa diakses lewat keyboard
                dan perangkat yang tidak mendukung drag atau tempel berkas. */}
            <input
              ref={inputBerkasRef}
              id="lampiran-bug"
              type="file"
              accept={TIPE_DIIZINKAN.join(',')}
              onChange={(e) => {
                ambilDariDaftar(e.target.files)
                // Reset agar berkas yang sama bisa dipilih ulang setelah dihapus.
                e.target.value = ''
              }}
              className="sr-only"
            />

            <p id="petunjuk-lampiran" className="text-micro text-muted-foreground">
              Gambar (png, jpg, webp, gif) atau PDF, maksimal {BUG_MAKS_LAMPIRAN_MB} MB.
            </p>

            <div aria-live="polite">
              {berkas && (
                <div className="flex items-center gap-2.5 rounded-lg border border-border px-2.5 py-2">
                  {pratinjau ? (
                    <img
                      src={pratinjau}
                      alt={`Pratinjau ${berkas.name}`}
                      className="h-12 w-12 shrink-0 rounded-md border border-border object-cover"
                    />
                  ) : (
                    <FileText
                      className="h-12 w-12 shrink-0 rounded-md border border-border p-3 text-muted-foreground"
                      aria-hidden="true"
                    />
                  )}
                  <div className="min-w-0 flex-1">
                    <p className="truncate text-[12.5px] font-medium text-foreground">
                      {berkas.name}
                    </p>
                    <p className="text-micro text-muted-foreground">
                      {labelUkuran(berkas.size)}
                    </p>
                  </div>
                  <Button
                    type="button"
                    variant="ghost"
                    size="icon"
                    aria-label={`Hapus lampiran ${berkas.name}`}
                    onClick={(e) => {
                      e.stopPropagation()
                      pilihBerkas(null)
                    }}
                  >
                    <X className="h-4 w-4" aria-hidden="true" />
                  </Button>
                </div>
              )}
            </div>
          </div>

          <div className="flex justify-end">
            <Button type="submit" size="sm" isLoading={kirim.isPending}>
              Kirim laporan
            </Button>
          </div>
        </form>
      </Card>
    </div>
  )
}

/**
 * Membuat URL pratinjau untuk gambar dan selalu membebaskannya saat berkas
 * berganti atau halaman ditinggalkan. Tanpa revoke, blob menumpuk di memori
 * selama pengguna masih di halaman ini.
 */
function usePratinjauLampiran(berkas: File | null): string | null {
  // URL dibuat saat render, bukan lewat setState di effect: nilainya memang
  // diturunkan dari berkas, jadi tidak ada state tambahan yang perlu disinkronkan.
  const url = useMemo(() => {
    if (!berkas || !berkas.type.startsWith('image/')) return null
    return URL.createObjectURL(berkas)
  }, [berkas])

  useEffect(() => {
    if (!url) return
    return () => URL.revokeObjectURL(url)
  }, [url])

  return url
}

/**
 * Mengambil berkas dari papan klip. Chromium mengisi `files` untuk gambar hasil
 * tangkapan layar, sedangkan sebagian peramban hanya mengisi `items`; keduanya
 * dicoba, dan berkas duplikat (ada di dua tempat) dibuang agar tidak terhitung dua.
 */
function berkasDariClipboard(data: DataTransfer | null): File[] {
  if (!data) return []

  const hasil: File[] = Array.from(data.files ?? [])

  for (const item of Array.from(data.items ?? [])) {
    if (item.kind !== 'file') continue
    const file = item.getAsFile()
    if (!file) continue
    const sudahAda = hasil.some(
      (f) => f.name === file.name && f.size === file.size && f.type === file.type,
    )
    if (!sudahAda) hasil.push(file)
  }

  return hasil
}
