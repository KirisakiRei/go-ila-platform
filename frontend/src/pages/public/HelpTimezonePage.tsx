import { useQuery } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { publicContentApi } from '@/lib/system-api'

/**
 * SESSION_ROWS adalah jadwal sesi dalam jam New York.
 *
 * Jam WIB sengaja tidak ditulis mati: selisihnya bergeser saat New York
 * menerapkan daylight saving, sehingga jam WIB dihitung dari jawaban server
 * pada saat halaman dibuka (plans/25 §2.3).
 */
const SESSION_ROWS = [
  { label: 'Buka mingguan', day: 'Minggu', time: '17:00' },
  { label: 'Sesi London', day: 'Senin–Jumat', time: '03:00' },
  { label: 'Sesi New York', day: 'Senin–Jumat', time: '08:00' },
  { label: 'Tutup sesi New York', day: 'Senin–Jumat', time: '17:00' },
  { label: 'Tutup mingguan', day: 'Jumat', time: '17:00' },
]

/** shiftWIB menggeser jam "HH:MM" New York ke WIB dengan selisih dari server. */
function shiftWIB(time: string, offsetMinutes: number): string {
  const [hours = NaN, minutes = NaN] = time.split(':').map(Number)
  if (!Number.isFinite(hours) || !Number.isFinite(minutes)) return '—'

  const total = (hours * 60 + minutes + offsetMinutes + 24 * 60) % (24 * 60)
  const hh = String(Math.floor(total / 60)).padStart(2, '0')
  const mm = String(total % 60).padStart(2, '0')
  return `${hh}:${mm}`
}

/** offsetWIBMenit menghitung selisih WIB terhadap New York dari pasangan waktu server. */
function offsetWIBMenit(pair?: { wib: string; ny: string }): number {
  if (!pair) return 11 * 60
  const parse = (value: string) => {
    const match = value.match(/(\d{1,2}):(\d{2})/)
    if (!match) return null
    return Number(match[1]) * 60 + Number(match[2])
  }
  const wib = parse(pair.wib)
  const ny = parse(pair.ny)
  if (wib === null || ny === null) return 11 * 60

  const diff = wib - ny
  return diff < 0 ? diff + 24 * 60 : diff
}

/**
 * HelpTimezonePage menjawab pertanyaan yang paling sering muncul: mengapa
 * panel menampilkan dua jam sekaligus.
 */
export default function HelpTimezonePage() {
  const { data } = useQuery({ queryKey: ['public', 'status'], queryFn: publicContentApi.status })

  const offset = offsetWIBMenit(data?.time_pair)
  const market = data?.market

  return (
    <div className="bg-[#f5f5f7] text-[#09090b]">
      <div className="mx-auto w-full max-w-4xl px-4 pt-10 pb-20 sm:px-6 sm:pt-14 sm:pb-24">
        <nav aria-label="Jejak navigasi" className="text-[12px] text-[#6b6b74]">
          <ol className="flex flex-wrap items-center gap-1.5">
            <li>
              <Link to="/" className="transition-colors hover:text-[#09090b]">
                Beranda
              </Link>
            </li>
            <li aria-hidden="true">/</li>
            <li>
              <Link to="/bantuan" className="transition-colors hover:text-[#09090b]">
                Pusat Bantuan
              </Link>
            </li>
            <li aria-hidden="true">/</li>
            <li className="text-[#3f3f46]">Sesi pasar &amp; zona waktu</li>
          </ol>
        </nav>

        <header className="mt-8">
          <p className="text-[11px] font-semibold uppercase tracking-[0.18em] text-[#6b6b74]">
            Pusat Bantuan
          </p>
          <h1 className="mt-2 text-[28px] font-semibold leading-tight tracking-tight sm:text-[36px]">
            Kenapa ada dua jam?
          </h1>
          <p className="mt-4 max-w-2xl text-[15px] leading-relaxed text-[#52525b]">
            Kami memakai dua jam karena keduanya menjawab pertanyaan yang berbeda. Satu untuk pasar,
            satu untuk Anda.
          </p>
        </header>

        {/* Kalimat inti: harus muncul apa adanya (plans/25 §2.2). */}
        <blockquote className="mt-8 rounded-2xl border border-[#09090b]/85 bg-[#09090b] p-6 text-[15px] leading-relaxed text-white sm:p-7">
          <p className="font-medium">
            “Jam trading mengikuti New York. Hitungan hari Anda mengikuti WIB. Keduanya menunjuk ke
            waktu yang sama — hanya nama harinya berbeda.”
          </p>
        </blockquote>

        <div className="mt-6 grid gap-3 sm:grid-cols-2">
          <div className="rounded-xl border border-zinc-200 bg-white p-5">
            <p className="text-[11px] font-semibold uppercase tracking-[0.14em] text-[#6b6b74]">
              New York
            </p>
            <p className="mt-1.5 text-[15px] font-semibold">Jam pasar</p>
            <p className="mt-2 text-[13px] leading-relaxed text-[#52525b]">
              Penilaian setup dan sesi pasar mengikuti jam New York, karena di sanalah likuiditas
              utama bergerak.
            </p>
            {data?.time_pair?.ny && (
              <p className="mt-3 text-[12px] text-[#6b6b74] tabular">
                Saat ini {data.time_pair.ny} NY
              </p>
            )}
          </div>
          <div className="rounded-xl border border-zinc-200 bg-white p-5">
            <p className="text-[11px] font-semibold uppercase tracking-[0.14em] text-[#6b6b74]">
              Waktu Indonesia Barat
            </p>
            <p className="mt-1.5 text-[15px] font-semibold">Siklus Harian (WIB)</p>
            <p className="mt-2 text-[13px] leading-relaxed text-[#52525b]">
              Reset kuota harian, jumlah transaksi, dan batas risiko dihitung berdasarkan tanggal
              WIB.
            </p>
            {data?.time_pair?.wib && (
              <p className="mt-3 text-[12px] text-[#6b6b74] tabular">
                Saat ini {data.time_pair.wib} WIB
              </p>
            )}
          </div>
        </div>

        {/* Tabel jadwal sesi, dihitung dari selisih waktu server. */}
        <section className="mt-10">
          <h2 className="text-[15px] font-semibold tracking-tight">Jadwal sesi</h2>
          <p className="mt-1.5 text-[13px] leading-relaxed text-[#52525b]">
            Jam WIB di bawah dihitung dari selisih waktu yang dilaporkan server saat halaman ini
            dibuka.
          </p>
          <div className="mt-4 overflow-hidden rounded-xl border border-zinc-200 bg-white">
            <table className="w-full border-collapse text-left text-[13px]">
              <thead>
                <tr className="bg-[#f4f4f5]">
                  <th
                    scope="col"
                    className="border-b border-zinc-200 px-4 py-3 text-[11px] font-semibold uppercase tracking-[0.1em] text-[#3f3f46]"
                  >
                    Kejadian
                  </th>
                  <th
                    scope="col"
                    className="border-b border-zinc-200 px-4 py-3 text-[11px] font-semibold uppercase tracking-[0.1em] text-[#3f3f46]"
                  >
                    Hari
                  </th>
                  <th
                    scope="col"
                    className="border-b border-zinc-200 px-4 py-3 text-[11px] font-semibold uppercase tracking-[0.1em] text-[#3f3f46]"
                  >
                    New York
                  </th>
                  <th
                    scope="col"
                    className="border-b border-zinc-200 px-4 py-3 text-[11px] font-semibold uppercase tracking-[0.1em] text-[#3f3f46]"
                  >
                    WIB
                  </th>
                </tr>
              </thead>
              <tbody>
                {SESSION_ROWS.map((row) => (
                  <tr key={row.label} className="even:bg-[#fafafa]">
                    <th scope="row" className="border-b border-zinc-100 px-4 py-3 font-medium">
                      {row.label}
                    </th>
                    <td className="border-b border-zinc-100 px-4 py-3 text-[#52525b]">{row.day}</td>
                    <td className="border-b border-zinc-100 px-4 py-3 text-[#52525b] tabular">
                      {row.time}
                    </td>
                    <td className="border-b border-zinc-100 px-4 py-3 text-[#52525b] tabular">
                      {shiftWIB(row.time, offset)}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          <p className="mt-3 text-[12px] leading-relaxed text-[#6b6b74]">
            Saat New York beralih ke daylight saving, jam WIB ikut bergeser satu jam. Tabel ini
            menampilkan nilai terbaru setiap kali halaman dibuka.
          </p>
        </section>

        {/* Kasus yang paling membingungkan. */}
        <section className="mt-10 rounded-xl border border-zinc-200 bg-white p-5 sm:p-6">
          <h2 className="text-[15px] font-semibold tracking-tight">
            Saya trading pukul 23.00 dan 01.00 WIB. Mengapa dihitung dua hari?
          </h2>
          <p className="mt-2 text-[13.5px] leading-relaxed text-[#52525b]">
            23.00 WIB dan 01.00 WIB memang dua tanggal berbeda, dan siklus harian mengikuti tanggal
            WIB. Kuota harian Anda direset pada 00.00 WIB.
          </p>
        </section>

        {market && (
          <section className="mt-4 rounded-xl border border-zinc-200 bg-white p-5 sm:p-6">
            <h2 className="text-[15px] font-semibold tracking-tight">Status pasar saat ini</h2>
            <p className="mt-2 text-[13.5px] leading-relaxed text-[#52525b]">
              Pasar sedang <span className="font-medium text-[#09090b]">{market.label}</span>.
              {market.next_label && (
                <>
                  {' '}
                  Berikutnya {market.next_label.toLowerCase()} pada {market.next_change_pair.wib}{' '}
                  WIB ({market.next_change_pair.ny} NY).
                </>
              )}
            </p>
          </section>
        )}

        <div className="mt-10 flex flex-wrap gap-3 border-t border-zinc-200 pt-6">
          <Link
            to="/bantuan"
            className="inline-flex items-center justify-center rounded-lg border border-zinc-300 bg-white px-4 py-2.5 text-[13px] font-medium text-[#09090b] transition-colors hover:bg-zinc-100"
          >
            Semua topik bantuan
          </Link>
          <a
            href="mailto:dukungan@rekavia.com"
            className="inline-flex items-center justify-center rounded-lg bg-[#09090b] px-4 py-2.5 text-[13px] font-semibold text-white transition-colors hover:bg-zinc-800"
          >
            Hubungi dukungan
          </a>
        </div>
      </div>
    </div>
  )
}
