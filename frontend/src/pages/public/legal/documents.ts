/**
 * Isi dokumen legal publik situs Imaginary Liquidity.
 *
 * Sumber kebenaran resmi tetap `legal_document` di server: dokumen berversi,
 * persetujuannya dicatat, dan admin dapat menerbitkan versi baru lewat CMS.
 * Berkas ini menyediakan teks keluaran awal yang sudah disunting dari draf
 * sistem menjadi bahasa yang layak dibaca publik — tanpa penanda tempat yang
 * belum terisi, tanpa istilah teknis internal, dan tanpa janji hasil.
 *
 * Bila server mengirim versi terbit, versi server yang dipakai. Bila dokumen
 * belum diterbitkan, halaman jatuh ke isi di bawah agar pengunjung tetap
 * mendapat penjelasan yang utuh.
 */

export type LegalSlug = 'privacy' | 'tos' | 'risk' | 'refund' | 'fair_use'

export type LegalDocumentContent = {
  slug: LegalSlug
  title: string
  /** Ringkasan satu kalimat untuk kartu hub dan kepala dokumen. */
  summary: string
  /** Kelompok dokumen pada hub Legal Center. */
  group: 'Kebijakan' | 'Perjanjian'
  /** Dokumen yang paling sering dibuka, ditandai di hub. */
  featured?: boolean
  body: string
}

export const LEGAL_GROUPS: { key: LegalDocumentContent['group']; label: string; note: string }[] = [
  {
    key: 'Perjanjian',
    label: 'Perjanjian',
    note: 'Dasar hubungan Anda dengan layanan. Disetujui saat mendaftar dan sebelum menghubungkan akun broker.',
  },
  {
    key: 'Kebijakan',
    label: 'Kebijakan',
    note: 'Ketentuan mengenai data, pembayaran, dan penggunaan layanan yang wajar.',
  },
]

const PRIVACY = `
Dokumen ini menjelaskan data apa yang kami kumpulkan saat Anda memakai Imaginary Liquidity, mengapa data itu diperlukan, berapa lama disimpan, dan cara Anda mengendalikannya. Nama penanggung jawab dan alamat surel dukungan di bawah ini adalah kanal resmi untuk setiap permintaan terkait data pribadi.

| Penyelenggara | Layanan | Kontak |
|---|---|---|
| Imaginary Liquidity, rekavia.com | Analisis likuiditas dan alat bantu eksekusi trading | dukungan@rekavia.com |

## 1. Ruang lingkup kebijakan

Kebijakan ini berlaku untuk situs, panel pelanggan, dan saluran notifikasi yang kami sediakan. Kebijakan ini tidak mengatur pemrosesan data oleh broker tempat Anda membuka akun. Broker memiliki kebijakan privasinya sendiri, dan kami menyarankan Anda membacanya.

## 2. Data yang kami kumpulkan

Kami hanya mengumpulkan data yang dibutuhkan untuk menjalankan layanan, menjaga keamanan akun, dan memenuhi kewajiban hukum kami.

| Kategori | Contoh data | Alasan kami memerlukannya |
|---|---|---|
| Identitas akun | Nama, alamat surel, kata sandi terenkripsi | Membuat akun, masuk, dan berkomunikasi dengan Anda |
| Keamanan akun | Kata sandi dalam bentuk hash, kode autentikasi dua faktor, cadangan kodenya | Melindungi akun bila kata sandi diketahui pihak lain |
| Kredensial broker | Token akses broker, disimpan terenkripsi | Menjalankan permintaan atas nama Anda ke broker |
| Aktivitas | Waktu masuk, perangkat, catatan perubahan pengaturan | Mendeteksi akses tidak sah dan menyusun jejak audit |
| Aktivitas trading | Sinyal yang diterima, posisi, dan hasilnya | Menyajikan riwayat yang dapat Anda periksa |
| Pembayaran | Nomor tagihan, nominal, status, referensi pembayaran | Menerbitkan tagihan dan memenuhi kewajiban perpajakan |
| Komunikasi dukungan | Isi percakapan dengan dukungan | Menjawab pertanyaan dan menyelesaikan keluhan |

> [!catatan] Data sensitif dalam bentuk ringkas
> Alamat IP dicatat sebagai hash, bukan alamat mentah. Nomor kartu atau rekening pembayaran tidak pernah kami simpan; data itu ditangani langsung oleh penyedia pembayaran.

## 3. Data yang tidak kami kumpulkan

Agar batasnya jelas, berikut data yang sengaja tidak kami minta dan tidak kami simpan.

| Data | Alasan |
|---|---|
| Nomor kartu atau kredensial pembayaran | Ditangani penyedia pembayaran; tidak pernah melewati server kami |
| Nomor identitas kependudukan dan data biometrik | Tidak diperlukan untuk menyediakan layanan |
| Kontak, pesan, atau berkas di perangkat Anda | Layanan tidak membutuhkannya |
| Saldo dan dana Anda | Dana tetap berada di akun broker Anda; kami tidak mengelolanya |

## 4. Dasar pemrosesan dan penggunaan data

Kami memproses data untuk menjalankan perjanjian layanan dengan Anda, memenuhi kewajiban hukum, dan melindungi layanan dari penyalahgunaan. Secara rinci, data dipakai untuk:

1. Mengautentikasi akses dan menjaga keamanan akun Anda.
2. Meneruskan permintaan eksekusi ke broker sesuai aturan yang Anda tetapkan.
3. Mengirim notifikasi sinyal dan pemberitahuan penting melalui panel, Telegram, atau surel.
4. Menyusun riwayat dan jejak keputusan yang dapat Anda periksa.
5. Menerbitkan tagihan, mencatat pembayaran, dan menangani permintaan pengembalian dana.
6. Mendeteksi, mencegah, dan menindak penyalahgunaan layanan.

Kami tidak menggunakan data Anda untuk iklan bertarget, tidak menjualnya, dan tidak menyusun profil untuk kepentingan pihak ketiga.

## 5. Pengamanan data

Kata sandi disimpan sebagai hash satu arah dan tidak dapat dibaca ulang, termasuk oleh kami. Token akses broker dan kode autentikasi dua faktor disimpan terenkripsi dengan kunci yang disimpan terpisah dari basis data.

> [!penting] Kredensial broker tidak pernah ditampilkan
> Setelah tersimpan, kredensial broker tidak dapat ditampilkan kembali kepada siapa pun, termasuk kepada staf kami dan administrator sistem. Kredensial hanya dipakai untuk mengirim permintaan ke broker atas nama Anda.

Akses staf dibatasi berdasarkan peran dan dicatat dalam jejak audit. Setiap tindakan administratif yang menyentuh akun pengguna meninggalkan catatan yang tidak dapat dihapus.

## 6. Pihak ketiga yang membantu layanan

Kami bekerja sama dengan penyedia terbatas yang diperlukan agar layanan berjalan. Mereka hanya menerima data yang relevan dengan tugasnya.

| Pihak | Data yang diterima | Tujuan |
|---|---|---|
| Broker Anda | Kredensial akses dan permintaan order | Menjalankan eksekusi atas nama Anda |
| Penyedia pembayaran | Nomor tagihan, nominal, alamat surel | Memproses pembayaran dan pengembalian dana |
| Telegram | Pengenal percakapan dan isi notifikasi | Mengirim pemberitahuan yang Anda aktifkan |
| Penyedia infrastruktur | Data yang tersimpan pada server | Menjalankan dan mencadangkan layanan |

Penyedia tersebut terikat kewajiban kerahasiaan dan hanya boleh memakai data untuk keperluan yang kami tentukan.

## 7. Masa simpan data

| Jenis data | Masa simpan | Alasan |
|---|---|---|
| Tagihan dan catatan pembayaran | 10 tahun | Kewajiban perpajakan dan akuntansi |
| Riwayat trading dan jejak keputusan | 1 tahun, lalu diringkas | Evaluasi layanan dan penyelesaian sengketa |
| Catatan teknis dan aktivitas sistem | 30 hari | Pemantauan keamanan dan perbaikan layanan |
| Kredensial broker | Selama akun broker terhubung | Menjalankan eksekusi; dihapus saat akun dilepas |
| Bukti persetujuan dokumen | Selama akun aktif dan setelahnya, dalam bentuk teranonimkan | Membuktikan dokumen yang Anda setujui |

Setelah masa simpan berakhir, data dihapus atau diringkas sehingga tidak lagi dapat dikaitkan dengan Anda.

## 8. Hak Anda atas data

Anda berhak mengetahui, memperbaiki, dan mengendalikan data pribadi Anda. Permintaan dapat diajukan melalui dukungan dan kami tanggapi dalam waktu yang wajar.

| Hak | Cara mengajukannya |
|---|---|
| Melihat data akun dan riwayat Anda | Langsung melalui panel pelanggan |
| Meminta salinan data pribadi | Ajukan melalui dukungan; kami kirimkan dalam format yang dapat dibaca |
| Memperbaiki data yang keliru | Perbarui sendiri di pengaturan akun atau ajukan melalui dukungan |
| Menghapus akun | Ajukan melalui dukungan; data pribadi dihapus, bukti persetujuan dipertahankan dalam bentuk teranonimkan |
| Menghentikan notifikasi | Atur sendiri di pengaturan akun atau melalui perintah bot Telegram |

Beberapa data, seperti tagihan, tetap kami simpan setelah akun dihapus karena diwajibkan oleh ketentuan perpajakan.

## 9. Persetujuan dan bukti persetujuan

Saat Anda menyetujui sebuah dokumen, kami mencatat jenis dokumen, nomor versi, waktu persetujuan, dan penanda teknis yang diperlukan sebagai bukti. Pencatatan ini dilakukan agar kami dapat menunjukkan dokumen versi mana yang berlaku pada saat Anda menyetujuinya. Anda dapat menolak dokumen versi baru; konsekuensinya dijelaskan pada Syarat & Ketentuan.

## 10. Perubahan kebijakan

Kebijakan ini dapat diperbarui. Setiap perubahan diterbitkan sebagai versi baru dengan tanggal berlaku yang jelas. Apabila perubahan menyangkut cara kami menangani data Anda, kami akan meminta persetujuan Anda sebelum perubahan itu berlaku bagi akun Anda.

## 11. Menghubungi kami

Pertanyaan, keberatan, atau permintaan terkait data pribadi dapat disampaikan melalui dukungan@rekavia.com. Kami akan mengonfirmasi penerimaan permintaan Anda dan menyampaikan perkembangannya.
`

const TOS = `
Ketentuan ini mengatur hubungan antara Anda sebagai pengguna dan Imaginary Liquidity sebagai penyelenggara layanan. Bacalah dengan saksama sebelum membuat akun. Dengan mendaftar, menghubungkan akun broker, atau menggunakan layanan, Anda menyatakan telah membaca dan menyetujui ketentuan ini.

| Berlaku sejak | Penyelenggara | Kontak |
|---|---|---|
| 8 November 2026 | Imaginary Liquidity, rekavia.com | dukungan@rekavia.com |

## 1. Definisi

Beberapa istilah dalam dokumen ini memiliki makna tertentu agar tidak menimbulkan tafsir ganda.

1. **Layanan** adalah platform berlangganan yang menyediakan analisis teknik berdasarkan metode Imaginary Liquidity, pemberitahuan sinyal, dan pada paket tertentu menjalankan eksekusi sesuai aturan yang Anda tetapkan.
2. **Sinyal** adalah pemberitahuan bahwa suatu kondisi teknik terpenuhi pada sebuah instrumen, disertai rencana masuk, batas kerugian, dan target pengelolaan posisi.
3. **Akun Broker** adalah akun trading milik Anda pada broker pihak ketiga yang Anda hubungkan sendiri ke Layanan.
4. **Langganan** adalah hak menggunakan Layanan untuk jangka waktu tertentu sesuai paket yang Anda pilih.
5. **Penyelenggara** adalah Imaginary Liquidity sebagai pihak yang menyediakan Layanan.

## 2. Ruang lingkup layanan

Layanan menyediakan analisis teknik, pemberitahuan sinyal, pencatatan aktivitas, dan pada paket tertentu eksekusi otomatis berdasarkan aturan risiko yang Anda tentukan sendiri.

> [!penting] Batas layanan yang perlu dipahami
> Kami tidak mengelola dana Anda, tidak menjadi kustodian, dan tidak menjadi perantara transaksi di pasar. Dana Anda tetap berada di Akun Broker milik Anda dan tidak pernah berpindah ke kami. Kami juga tidak bertindak sebagai penasihat keuangan.

## 3. Bukan nasihat keuangan

Layanan ini bukan nasihat investasi, bukan rekomendasi keuangan, dan tidak memperhitungkan kondisi keuangan pribadi Anda. Sinyal yang Anda terima adalah hasil pembacaan kondisi pasar berdasarkan aturan yang berlaku umum bagi seluruh pelanggan, bukan penilaian atas kesesuaian instrumen dengan situasi Anda.

Keputusan untuk membuka atau menutup posisi sepenuhnya berada di tangan Anda. Jika Anda memerlukan nasihat yang sesuai dengan kondisi pribadi Anda, konsultasikan dengan penasihat keuangan yang berwenang.

## 4. Risiko

Perdagangan mata uang dan instrumen derivatif memiliki risiko kerugian yang tinggi, dan kerugian dapat melebihi modal yang Anda tempatkan. Risiko ini diatur lebih rinci dalam Disclaimer Risiko Trading yang merupakan bagian tidak terpisahkan dari Ketentuan ini.

## 5. Kewajiban pengguna

Selama menggunakan Layanan, Anda menyatakan dan menjamin bahwa:

1. Akun Broker yang Anda hubungkan adalah milik Anda sendiri, atau akun yang Anda berhak gunakan secara sah.
2. Kredensial yang Anda berikan benar, masih berlaku, dan Anda berwenang memberikannya.
3. Anda menjaga kerahasiaan kata sandi, kode autentikasi dua faktor, dan kode cadangan akun Layanan Anda.
4. Anda mematuhi hukum dan peraturan yang berlaku di wilayah Anda, termasuk ketentuan broker Anda.
5. Anda memahami sepenuhnya cara kerja paket yang Anda pilih, termasuk apakah paket tersebut menjalankan eksekusi otomatis.

> [!perhatian] Akun milik pihak lain
> Menghubungkan akun broker milik orang lain tanpa izin tertulis merupakan pelanggaran berat dan dapat mengakibatkan pemblokiran segera tanpa pengembalian dana.

## 6. Larangan

Hal-hal berikut dilarang dan dapat mengakibatkan tindakan sesuai Kebijakan Penggunaan Wajar.

1. Menyebarluaskan, menjual kembali, atau membagikan sinyal kepada pihak lain tanpa izin tertulis dari kami.
2. Memakai satu Langganan untuk beberapa orang atau mengalihkan akses akun kepada pihak lain.
3. Membuat beberapa akun untuk menyalahgunakan masa uji coba atau promo.
4. Melakukan manipulasi pembayaran atau mengklaim pembayaran yang tidak pernah terjadi.
5. Membebani sistem secara berlebihan atau melakukan upaya akses tanpa izin terhadap data dan sistem kami.
6. Menggunakan Layanan untuk kegiatan yang melanggar hukum.

## 7. Biaya, pembayaran, dan pengembalian dana

Biaya Langganan mengikuti paket yang Anda pilih dan ditagihkan untuk setiap periode berjalan. Harga yang berlaku adalah harga yang tercantum pada halaman Harga atau tagihan Anda pada saat penagihan diterbitkan. Ketentuan pengembalian dana diatur dalam Kebijakan Pengembalian Dana.

## 8. Penangguhan dan pemblokiran

Kami berhak menangguhkan atau memblokir akun apabila terdapat indikasi pelanggaran terhadap Ketentuan ini, penyalahgunaan sistem, upaya manipulasi, atau kewajiban pembayaran yang tidak dipenuhi.

> [!catatan] Hak Anda saat dikenai tindakan
> Setiap tindakan disertai alasan yang disampaikan melalui kanal komunikasi yang tersedia dan dicatat dalam jejak audit sistem. Anda dapat mengajukan keberatan melalui dukungan, dan keberatan tersebut akan ditinjau.

## 9. Batas tanggung jawab

Sejauh diperbolehkan oleh hukum yang berlaku, kami tidak bertanggung jawab atas kerugian yang timbul dari penggunaan Layanan, termasuk kerugian akibat keputusan trading Anda, perubahan kondisi pasar, gangguan koneksi ke broker, atau kegagalan layanan pihak ketiga.

Kami tidak menjamin bahwa Layanan akan selalu tersedia tanpa gangguan. Pemeliharaan terjadwal dan gangguan teknis dapat menahan permintaan eksekusi baru, sementara posisi yang sudah terbuka tetap dikelola oleh broker Anda.

## 10. Hak kekayaan intelektual

Seluruh isi Layanan, termasuk metode, dokumentasi, tampilan, dan perangkat lunaknya, dilindungi hak kekayaan intelektual. Anda memperoleh hak terbatas, tidak eksklusif, dan tidak dapat dialihkan untuk menggunakan Layanan selama Langganan aktif. Hak tersebut tidak mencakup hak untuk menyalin, memodifikasi, mendistribusikan, atau membuat karya turunan dari Layanan.

## 11. Perubahan ketentuan

Ketentuan ini dapat diperbarui. Setiap perubahan diterbitkan sebagai versi baru dengan nomor versi dan tanggal berlaku.

Apabila perubahan menyangkut hak atau kewajiban Anda, kami akan meminta persetujuan Anda atas versi baru sebelum perubahan itu berlaku. Selama Anda belum menyetujuinya, Anda tetap dapat masuk ke akun, tetapi eksekusi dan pembayaran ditahan sampai persetujuan diberikan. Anda dapat memilih untuk tidak menyetujui dan menghentikan penggunaan Layanan.

## 12. Penghentian penggunaan

Anda dapat berhenti menggunakan Layanan kapan saja. Langganan yang sedang berjalan tetap berlaku sampai akhir periode yang sudah dibayar, kecuali ditentukan lain dalam Kebijakan Pengembalian Dana.

## 13. Hukum yang berlaku dan penyelesaian sengketa

Ketentuan ini tunduk pada hukum Republik Indonesia. Setiap perselisihan akan diupayakan terlebih dahulu melalui musyawarah dalam jangka waktu yang wajar. Apabila penyelesaian secara musyawarah tidak tercapai, para pihak akan menempuh jalur penyelesaian sengketa sesuai hukum yang berlaku di Republik Indonesia.

## 14. Kontak

Pertanyaan mengenai Ketentuan ini dapat disampaikan melalui dukungan@rekavia.com.
`

const RISK = `
Dokumen ini menjelaskan risiko yang melekat pada penggunaan Imaginary Liquidity. Kami menempatkan penjelasan ini di depan, bukan di catatan kaki, karena keputusan untuk menggunakan layanan ini sebaiknya diambil dengan pemahaman risiko yang utuh.

> [!penting] Peringatan utama
> Perdagangan mata uang dan instrumen derivatif memiliki risiko kerugian yang tinggi dan dapat melebihi modal yang Anda tempatkan. Layanan ini adalah alat bantu analisis dan eksekusi, bukan jaminan keuntungan dan bukan nasihat keuangan. Hasil masa lalu tidak menjamin hasil di masa depan.

## 1. Risiko pasar

Harga bergerak cepat dan dipengaruhi banyak faktor, termasuk berita ekonomi, keputusan bank sentral, dan perubahan likuiditas. Kondisi pasar yang tidak normal dapat membuat pergerakan harga melampaui perkiraan mana pun, termasuk perkiraan yang dihasilkan oleh analisis teknik.

## 2. Tidak ada jaminan keuntungan

Layanan ini memakai analisis teknik berdasarkan metode Imaginary Liquidity. Metode tersebut tidak menjamin keuntungan dan tidak menjamin akurasi. Tidak semua hari menghasilkan setup yang memenuhi syarat, dan sebagian setup dapat berakhir dengan kerugian meski seluruh aturan dijalankan dengan benar.

## 3. Sistem dapat membuat kesalahan

Kami berupaya menjaga keandalan layanan, namun sistem perangkat lunak tidak bebas dari kesalahan. Sumber kesalahan yang mungkin terjadi antara lain:

1. Perhitungan yang tidak menghasilkan nilai sebagaimana mestinya.
2. Data pasar yang terlambat, tidak lengkap, atau tidak akurat.
3. Keterlambatan koneksi dan gangguan jaringan.
4. Gangguan pada layanan broker, penyedia data, atau penyedia infrastruktur.
5. Pemeliharaan terjadwal yang menahan permintaan eksekusi baru.

## 4. Eksekusi tidak selalu pada harga sinyal

Harga yang Anda peroleh dapat berbeda dari harga pada sinyal karena slippage, pelebaran spread, berkurangnya likuiditas, dan kecepatan eksekusi broker. Layanan tidak menjamin eksekusi pada harga tertentu, dan selisih tersebut di luar kendali kami.

## 5. Tanggung jawab atas posisi tetap pada Anda

Apabila terjadi kesalahan eksekusi, posisi yang telah terbuka tetap menjadi tanggung jawab Anda. Kami menyarankan Anda memeriksa posisi secara berkala pada aplikasi broker Anda, terutama setelah perubahan pengaturan atau ketika pasar bergerak cepat.

## 6. Batasan pemantauan

Layanan memantau instrumen dan sesi tertentu sesuai paket Anda. Instrumen di luar cakupan paket, kondisi pasar di luar jam pemantauan, dan peristiwa mendadak di luar jadwal tidak ikut dianalisis. Ketiadaan sinyal bukan berarti pasar tidak bergerak.

## 7. Risiko likuiditas dan pendanaan

Pastikan dana yang Anda tempatkan adalah dana yang siap Anda hadapi risikonya. Jangan menggunakan dana pinjaman, dana kebutuhan pokok, atau dana yang Anda perlukan dalam waktu dekat untuk aktivitas trading.

## 8. Risiko teknologi dan akses akun

Keamanan akun bergantung pada kerahasiaan kata sandi, kode autentikasi dua faktor, dan kredensial broker Anda. Kelalaian menjaga kerahasiaan tersebut dapat menyebabkan akses tidak sah. Segera hubungi dukungan apabila Anda menduga ada akses yang tidak Anda kenali.

## 9. Batas tanggung jawab

Sejauh diperbolehkan oleh hukum yang berlaku, kami tidak bertanggung jawab atas kerugian yang timbul dari penggunaan Layanan, termasuk kerugian akibat keputusan trading Anda sendiri.

## 10. Pernyataan pemahaman

Dengan menyetujui dokumen ini, Anda menyatakan memahami bahwa perdagangan memiliki risiko kehilangan dana, bahwa layanan ini tidak menjamin keuntungan, bahwa sistem dapat membuat kesalahan, dan bahwa seluruh keputusan trading merupakan tanggung jawab Anda sendiri.

Pertanyaan mengenai dokumen ini dapat disampaikan melalui dukungan@rekavia.com.
`

const REFUND = `
Kebijakan ini menjelaskan kapan pengembalian dana dapat diberikan, bagaimana prosesnya, dan hal-hal yang perlu Anda ketahui sebelum mengajukan permintaan. Kami menuliskan ketentuan ini secara terbuka agar Anda dapat memperkirakan hasilnya sejak awal.

| Berlaku sejak | Masa uji coba | Kontak |
|---|---|---|
| 8 November 2026 | 7 hari, tanpa biaya | dukungan@rekavia.com |

## 1. Prinsip dasar

Kami menyarankan Anda memanfaatkan masa uji coba sebelum berlangganan. Masa uji coba disediakan tanpa biaya dan tanpa kartu kredit, sehingga Anda dapat menilai kesesuaian layanan tanpa risiko pembayaran.

Pengembalian dana diberikan berdasarkan tingkat penggunaan layanan dalam periode tagihan yang sedang berjalan, bukan berdasarkan hasil trading Anda.

## 2. Ketentuan pengembalian dana

| Situasi | Pengembalian |
|---|---|
| Permintaan diajukan dalam 7 hari pertama dan layanan belum digunakan | 100% dari nilai tagihan |
| Permintaan diajukan dalam 7 hari pertama setelah layanan digunakan | 50% dari nilai tagihan |
| Permintaan diajukan setelah 7 hari penggunaan | Tidak ada pengembalian |
| Layanan tidak dapat diakses dari sisi kami lebih dari 48 jam berturut-turut | Ditinjau secara khusus |
| Akun diblokir karena pelanggaran Ketentuan | Tidak ada pengembalian |
| Masa uji coba | Tidak ada tagihan, sehingga tidak ada pengembalian |

> [!catatan] Cara kami menghitung penggunaan
> Layanan dianggap telah digunakan sejak akun broker terhubung dan pemantauan dimulai pada periode tagihan tersebut, atau sejak fitur berlangganan pertama kali diakses.

## 3. Prosedur pengajuan

1. Ajukan permintaan melalui dukungan@rekavia.com dari alamat surel yang terdaftar pada akun Anda.
2. Sertakan nomor tagihan atau tanggal pembayaran agar permintaan dapat ditelusuri dengan cepat.
3. Kami mengonfirmasi penerimaan permintaan dan menyampaikan keputusan paling lambat 7 hari kerja.
4. Apabila disetujui, dana dikembalikan ke kanal pembayaran asal sesuai waktu proses penyedia pembayaran.

## 4. Gangguan layanan dari sisi kami

Apabila layanan tidak dapat digunakan lebih dari 48 jam berturut-turut karena gangguan yang berasal dari sisi kami, permintaan Anda akan ditinjau secara khusus. Peninjauan ini mencakup pemberian perpanjangan masa langganan sebagai alternatif pengembalian dana, sesuai kesepakatan.

## 5. Hal yang perlu diperhatikan

1. Pengembalian dana tidak menghapus kewajiban Anda atas pelanggaran yang sudah terjadi.
2. Pemblokiran karena pelanggaran Ketentuan atau Kebijakan Penggunaan Wajar tidak memberikan hak pengembalian dana.
3. Nilai pengembalian dihitung dari nominal tagihan yang benar-benar dibayarkan, setelah memperhitungkan potongan kupon atau kredit yang digunakan.
4. Pengembalian dana tidak berlaku untuk perpanjangan otomatis yang sudah berjalan pada periode berikutnya; hentikan perpanjangan sebelum periode berakhir.

## 6. Menghentikan langganan

Anda dapat menghentikan perpanjangan langganan kapan saja melalui panel pelanggan. Langganan yang sudah dibayar tetap berlaku sampai akhir periode berjalan, dan Anda tetap dapat mengakses layanan sampai tanggal tersebut. Penghentian perpanjangan bukan pengembalian dana.

## 7. Kontak

Pertanyaan mengenai Kebijakan ini dapat disampaikan melalui dukungan@rekavia.com.
`

const FAIR_USE = `
Kebijakan ini menjelaskan batas penggunaan yang wajar atas layanan berlangganan, tindakan yang kami ambil bila batas itu dilampaui, dan cara Anda mengajukan keberatan. Tujuannya menjaga layanan tetap adil bagi seluruh pelanggan.

| Berlaku sejak | Ruang lingkup | Kontak |
|---|---|---|
| 8 November 2026 | Seluruh paket, termasuk masa uji coba | dukungan@rekavia.com |

## 1. Prinsip penggunaan wajar

Langganan bersifat pribadi dan diberikan kepada satu pengguna untuk satu akun. Sinyal yang Anda terima adalah hasil kerja yang disediakan bagi Anda sebagai pelanggan, bukan materi yang bebas dibagikan ulang.

## 2. Tindakan yang dilarang

| Tindakan | Contoh | Akibat |
|---|---|---|
| Membagikan sinyal | Mengirim ke grup, menyalin ke kanal, menjual kembali | Pemblokiran permanen |
| Berbagi akun | Satu langganan dipakai beberapa orang | Peringatan, lalu pemblokiran bila berulang |
| Akun ganda untuk uji coba | Mendaftar berulang dengan alamat surel berbeda | Uji coba dihentikan |
| Memakai akun broker milik orang lain | Menghubungkan akun tanpa izin pemiliknya | Pemblokiran segera |
| Manipulasi pembayaran | Mengklaim pembayaran yang tidak terjadi | Pemblokiran dan peninjauan |
| Membebani sistem berlebihan | Permintaan otomatis dalam jumlah besar | Pembatasan akses, lalu pemblokiran |
| Upaya akses tanpa izin | Mencoba membaca data pengguna lain | Pemblokiran dan pelaporan |

## 3. Tingkat pelanggaran dan tindakan

| Tingkat | Contoh | Tindakan pertama |
|---|---|---|
| Ringan | Kelalaian memperpanjang pembayaran | Pengingat dan tenggang waktu |
| Sedang | Akun dipakai bergantian oleh dua orang | Peringatan tertulis |
| Berat | Membagikan sinyal kepada pihak lain | Pemblokiran akun |
| Sangat berat | Memakai akun broker milik orang lain atau melakukan manipulasi pembayaran | Pemblokiran dan pertimbangan langkah hukum |

> [!perhatian] Akibat pemblokiran
> Akun yang diblokir karena pelanggaran tidak berhak atas pengembalian dana, sesuai Kebijakan Pengembalian Dana. Posisi yang sudah terbuka di akun broker Anda tetap dikelola oleh broker Anda sendiri.

## 4. Prosedur penegakan

Kami berupaya menegakkan kebijakan ini secara konsisten dan dapat dipertanggungjawabkan.

1. Temuan dicatat beserta bukti pendukungnya.
2. Tingkat pelanggaran ditentukan berdasarkan dampak dan kesengajaan.
3. Tindakan diberikan sesuai tingkat, disertai pemberitahuan alasan melalui kanal komunikasi yang tersedia.
4. Setiap tindakan dicatat dalam jejak audit sistem.

## 5. Mengajukan keberatan

Apabila Anda menilai tindakan yang dikenakan tidak tepat, ajukan keberatan melalui dukungan@rekavia.com dengan menyertakan alasan dan bukti yang Anda miliki. Kami meninjau keberatan tersebut dan menyampaikan hasilnya melalui kanal yang sama.

## 6. Kontak

Pertanyaan mengenai Kebijakan ini dapat disampaikan melalui dukungan@rekavia.com.
`

const DOCUMENTS: Record<LegalSlug, Omit<LegalDocumentContent, 'slug' | 'body'>> = {
  privacy: {
    title: 'Kebijakan Privasi',
    summary:
      'Data apa yang kami kumpulkan, mengapa, berapa lama disimpan, dan cara Anda mengendalikannya.',
    group: 'Kebijakan',
    featured: true,
  },
  tos: {
    title: 'Syarat & Ketentuan',
    summary:
      'Dasar hubungan Anda dengan layanan: ruang lingkup, kewajiban, dan batas tanggung jawab.',
    group: 'Perjanjian',
    featured: true,
  },
  risk: {
    title: 'Disclaimer Risiko Trading',
    summary: 'Risiko yang melekat pada perdagangan dan penggunaan alat bantu eksekusi.',
    group: 'Perjanjian',
    featured: true,
  },
  refund: {
    title: 'Kebijakan Pengembalian Dana',
    summary: 'Kapan pengembalian dana dapat diberikan dan bagaimana prosesnya.',
    group: 'Kebijakan',
  },
  fair_use: {
    title: 'Kebijakan Penggunaan Wajar',
    summary: 'Batas penggunaan langganan dan tindakan atas pelanggaran.',
    group: 'Kebijakan',
  },
}

export const LEGAL_DOCUMENTS: Record<LegalSlug, LegalDocumentContent> = {
  privacy: { slug: 'privacy', body: PRIVACY.trim(), ...DOCUMENTS.privacy },
  tos: { slug: 'tos', body: TOS.trim(), ...DOCUMENTS.tos },
  risk: { slug: 'risk', body: RISK.trim(), ...DOCUMENTS.risk },
  refund: { slug: 'refund', body: REFUND.trim(), ...DOCUMENTS.refund },
  fair_use: { slug: 'fair_use', body: FAIR_USE.trim(), ...DOCUMENTS.fair_use },
}

/** Urutan tampil pada pengelompokan hub dan tab sub-navigasi dokumen. */
export const LEGAL_ORDER: LegalSlug[] = ['privacy', 'tos', 'risk', 'refund', 'fair_use']

/** isLegalSlug mempersempit segmen URL menjadi slug yang dikenal. */
export function isLegalSlug(value: string | undefined): value is LegalSlug {
  return value !== undefined && value in LEGAL_DOCUMENTS
}
