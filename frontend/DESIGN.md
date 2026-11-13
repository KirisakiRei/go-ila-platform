# DESIGN.md — Sistem Visual ILA Platform

Dokumen ini merekam sistem visual yang **sedang dipakai**, bukan yang dicita-citakan.
Tujuannya: perubahan berikutnya tetap berada di dalam sistem, dan alat audit tidak
menagih ulang keputusan yang sudah diambil sadar.

Sumber kebenaran teknis tetap `src/styles/globals.css`, `tailwind.config.js`, dan
`src/components/ui/`. Dokumen ini menjelaskan *mengapa*-nya.

---

## Keputusan yang disengaja (jangan "diperbaiki")

Bagian ini melindungi keputusan desain dari saran generik alat audit. Setiap butir
menyebutkan aturan detektor yang mungkin menandainya, dan alasan penolakannya.

### 1. Monokrom total

Seluruh palet berada pada rona 0 (abu netral). Tidak ada warna semantik.

Makna disampaikan lewat **bobot huruf, garis, ikon tekstual, dan bentuk** — bukan rona.
`--destructive` (25%), `--success` (20%), dan `--warning` (35%) semuanya abu, dibedakan
dari teks utama hanya oleh kekuatan nada.

**Konsekuensi yang diterima:** pengguna dengan buta warna total tetap bisa membedakan
state, karena informasi tidak pernah hanya-rona.

**Aturan detektor yang ditolak:** saran menambahkan rona pada state error/sukses.

**Penyimpangan yang tercatat:** beberapa komponen memakai keluarga `emerald-*` dan
`amber-*` Tailwind (mis. `OnboardingCard`, `OnboardingChecklist`, `TimeframeRadarMatrix`,
`SettingsMobileView:205`). Ini **belum selaras** dengan prinsip monokrom dan perlu
keputusan produk tersendiri — bukan bagian dari audit keterbacaan.

### 2. Inter sebagai brand font

`fontFamily.sans = Inter` dipakai di 100% antarmuka. Ini keputusan merek, bukan default
tanpa pikir.

**Aturan detektor yang ditolak:** `overused-font`. Di-allowlist lewat
`npx impeccable ignores add-value overused-font Inter` dengan alasan ini.

### 3. Dua bundel terpisah

`index.html` memuat `src/public-main.tsx` (PublicApp — situs publik).
`app.html` memuat `src/main.tsx` (App — panel pelanggan, `basename="/app"`).

Bundel dashboard tidak ikut terunduh pengunjung tanpa akun (rujukan: `plans/11 J5b`).
**Jangan digabung.** Pemisahan ini juga sebabnya halaman panel memakai komponen
`*MobileView` tersendiri, bukan hasil responsif dari halaman desktop.

### 4. Skala tipografi: minimum 12px untuk teks fungsional

Teks yang membawa informasi wajib **>= 12px**. Ini berlaku untuk label navigasi,
header tabel, label metrik, badge status, dan teks isi.

Di bawah 12px hanya boleh dipakai bila **ketiganya** terpenuhi:

1. Elemen dekoratif — informasi yang sama tersedia di tempat lain;
2. Ruang terbatas secara struktural (mis. label sumbu di dalam chart);
3. Tidak dapat dijangkau fokus keyboard.

### 5. Pengecualian terdokumentasi: label chart 9.5px

`src/features/signals/components/TimeframeRadarMatrix.tsx` (baris 82, 91, 100, 109)
memakai `text-[9.5px] font-mono` untuk label di dalam matriks radar.

Alasan dipertahankan: ruang sel matriks terbatas secara struktural, teks memakai
`font-mono` sehingga lebar karakter konsisten, dan nilai yang sama selalu tersedia di
`SignalDetail` pada ukuran normal. Memenuhi ketiga syarat butir 4.

**Aturan detektor yang ditolak:** `undersized-ui-text` pada empat baris tersebut.

### 6. Navigasi bawah mobile: 11px, bukan 12px

`MobileBottomNav` memakai slot `w-16` (64px) per item dengan label 11px.

Alasan menyimpang dari minimum 12px pada butir 4: pengukuran di viewport 390px
menunjukkan label "Pengaturan" memakai 54px dari slot 56px — sisa 2px. Pada 12px label
itu menjadi ~65px dan **tidak muat**. Pada 11px menjadi ~59px dan masih muat di 64px,
termasuk di viewport 320px (5 x 64 = 320px).

**Aturan detektor yang ditolak:** `undersized-ui-text` pada label `MobileBottomNav`.

---

## Palet

Semua nilai HSL, rona selalu 0.

| Token | Nilai | Pemakaian |
| --- | --- | --- |
| `--background` | `0 0% 100%` | dasar halaman |
| `--foreground` | `0 0% 9%` | teks utama |
| `--card` | `0 0% 100%` | permukaan kartu |
| `--secondary` | `0 0% 96%` | permukaan sekunder, hover |
| `--muted` | `0 0% 96%` | latar tenang |
| `--muted-foreground` | `0 0% 40%` | teks sekunder |
| `--accent` | `0 0% 93%` | sorotan halus |
| `--border` | `0 0% 88%` | garis pemisah |
| `--input` | `0 0% 84%` | garis input |
| `--ring` | `0 0% 9%` | cincin fokus |
| `--destructive` | `0 0% 25%` | teks galat |
| `--success` | `0 0% 20%` | teks berhasil |
| `--warning` | `0 0% 35%` | teks peringatan |

### Ambang kontras

`--muted-foreground` di `40%` menghasilkan:

| Di atas | Rasio | Ambang AA (4.5:1) |
| --- | --- | --- |
| `--background` (putih) | 5.74:1 | lolos |
| `--secondary` / `--muted` (`96%`) | 5.10:1 | lolos |
| `--accent` (`93%`) | 4.80:1 | lolos |

Sebelumnya `45%` hanya mencapai 4.43:1 di atas `--secondary` dan gagal AA.

**Larangan:** jangan memakai opacity modifier pada `text-muted-foreground`
(mis. `text-muted-foreground/70`). Peredupan itu menjatuhkan rasio ke 2.7:1.

---

## Tipografi

### Skala resmi

Didefinisikan di `tailwind.config.js` -> `theme.extend.fontSize`.

| Utilitas | Ukuran | Line-height | Pemakaian |
| --- | --- | --- | --- |
| `text-micro` | 12px | 1.45 | label, badge, header tabel, label metrik |
| `text-caption` | 12.5px | 1.5 | meta, hint, deskripsi |
| `text-body-sm` | 13px | 1.55 | teks UI utama |
| `text-body` | 14px | 1.6 | paragraf |

Nilai di bawah `text-micro` hanya sah untuk butir 4 dan 5 di atas.

### Peran

| Peran | Ukuran | Bobot | Warna |
| --- | --- | --- | --- |
| Judul halaman (`PageHeader`) | 18px | 600 | `--foreground` |
| Judul section | 12px + `uppercase` + `tracking-[0.14em]` | 600 | `--muted-foreground` |
| Judul kartu | 14px | 600 | `--foreground` |
| Teks isi | 13–14px | 400 | `--foreground` |
| Teks sekunder | 12.5px | 400 | `--muted-foreground` |
| Label metrik | 12px + `uppercase` | 600 | `--muted-foreground` |
| Angka tabel | 13px + `.tabular` | 400/600 | `--foreground` |

### Aturan

- Angka yang dibandingkan berkolom wajib memakai kelas `.tabular`.
- Tidak ada teks lebih kecil dari 12px kecuali butir 4 dan 5.
- `text-wrap: pretty` pada teks isi dan `text-wrap: balance` pada judul
  (lihat `globals.css`) untuk menghindari baris terakhir satu kata.

---

## Bentuk

| Token | Nilai |
| --- | --- |
| `--radius` | `0.5rem` (8px) |
| `rounded-lg` | `var(--radius)` |
| `rounded-md` | `calc(var(--radius) - 2px)` |
| `rounded-sm` | `calc(var(--radius) - 4px)` |

Kartu memakai `rounded-2xl` untuk blok besar; kontrol (tombol, input, badge) memakai
`rounded-md`/`rounded-lg`.

---

## Komponen bersama

`src/components/ui/` — sumber tunggal untuk kontrol.

`alert`, `badge`, `button`, `card`, `empty-state`, `input`, `label`, `modal`,
`page-header`, `pagination`, `select`, `skeleton`, `stat`, `status-dot`, `status-pill`,
`table`.

Aturan: halaman **tidak** membuat ulang kontrol ini secara lokal. Bila tampilan perlu
berbeda, tambahkan varian di komponen bersama, jangan menyalin kelasnya.

---

## Struktur halaman

- `PageHeader` selalu memuat `<h1>` judul halaman. Satu `h1` per halaman.
- Hierarki heading tidak boleh melompati level. `h1` -> `h2` -> `h3`.
- `Section` memisahkan kelompok konten dengan judul tenang dan garis bawah.
- Kartu tidak diletakkan di dalam kartu. Untuk mengelompokkan di dalam kartu, pakai
  garis pemisah (`divide-y`, `border-t`) dan jarak, bukan permukaan kedua.
- Jarak di atas judul harus lebih besar daripada jarak di bawahnya, supaya judul
  terbaca terikat pada isinya, bukan pada blok sebelumnya.

---

## Fokus dan gerak

- Fokus keyboard selalu terlihat: `:focus-visible` memakai `ring-2` + `ring-offset-2`
  dengan `--ring`. Jangan dihilangkan walau tanpa warna.
- Animasi memakai `ease-out`; tidak ada easing memantul.
- Status "hidup" hanya berdenyut bila datanya benar-benar berubah langsung.
  Indikator statis dengan label jelas lebih jujur dan lebih tenang.

---

## Aksesibilitas

- Rasio kontras minimum 4.5:1 untuk teks isi, 3:1 untuk teks besar (>= 18.66px bold
  atau >= 24px) dan untuk batas kontrol.
- Setiap kontrol punya label yang terlihat atau `aria-label` yang bermakna.
- State tidak boleh disampaikan hanya lewat rona (lihat butir 1).
- Target sentuh minimum 44x44px pada tata letak mobile.
- `prefers-reduced-motion` dihormati untuk animasi yang tidak esensial.
