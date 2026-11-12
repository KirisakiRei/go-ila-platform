import { useState } from 'react'
import { Link, useLocation } from 'react-router-dom'
import { Paperclip } from 'lucide-react'
import { Alert, Button, Card, Input, Label, Select } from '@/components/ui'
import { ApiError } from '@/lib/api-error'
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
 * LaporkanBugPage adalah formulir laporan bug.
 *
 * Satu layar, bukan langkah bertahap: formulirnya pendek, dan langkah-langkah
 * hanya menambah gesekan pada hal yang sedang tidak menyenangkan bagi pengguna.
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

  function pilihBerkas(file: File | null) {
    setGalat(null)
    if (file && file.size > BUG_MAX_LAMPIRAN_BYTE) {
      setBerkas(null)
      setGalat(`Ukuran lampiran maksimal ${BUG_MAKS_LAMPIRAN_MB} MB.`)
      return
    }
    setBerkas(file)
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
          <p className="mt-0.5 text-[11.5px] leading-relaxed text-muted-foreground">
            Ada yang tidak berjalan semestinya? Ceritakan di sini.
          </p>
        </div>
        <Button asChild size="sm" variant="outline">
          <Link to="/laporkan-bug/riwayat">Riwayat laporan</Link>
        </Button>
      </div>

      <Card className="p-4">
        <form onSubmit={onSubmit} noValidate className="flex flex-col gap-3.5">
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

          <div className="flex flex-col gap-1">
            <Label htmlFor="lampiran-bug">Lampiran (opsional)</Label>
            <input
              id="lampiran-bug"
              type="file"
              accept="image/png,image/jpeg,image/webp,image/gif,application/pdf"
              onChange={(e) => pilihBerkas(e.target.files?.[0] ?? null)}
              className="text-[12.5px] text-muted-foreground file:mr-2 file:rounded-md file:border file:border-border file:bg-secondary file:px-2.5 file:py-1 file:text-xs file:text-foreground"
            />
            <p className="flex items-center gap-1 text-[11px] text-muted-foreground">
              <Paperclip className="h-3 w-3" />
              Gambar (png, jpg, webp, gif) atau PDF, maksimal {BUG_MAKS_LAMPIRAN_MB} MB.
            </p>
            {berkas && (
              <p className="text-[11px] text-foreground">
                Terpilih: {berkas.name} ({Math.ceil(berkas.size / 1024)} KB)
              </p>
            )}
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
