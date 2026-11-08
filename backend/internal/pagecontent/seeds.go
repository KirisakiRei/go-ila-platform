package pagecontent

// Seeds mengembalikan isi awal setiap halaman publik.
func Seeds() []Seed {
	return []Seed{features(), pricing(), about(), help(), legal()}
}

func features() Seed {
	return Seed{
		Slug:            "fitur",
		Eyebrow:         "Fitur",
		Title:           "Yang dikerjakan sistem, dan yang tidak.",
		Lead:            "Imaginary Liquidity membaca kondisi likuiditas pasar, menyusun rencana eksekusi, lalu mengirimkannya ke akun broker Anda. Halaman ini menjelaskan setiap bagian pekerjaan itu beserta batasannya, supaya Anda tahu persis di mana peran sistem berhenti dan peran Anda mulai.",
		MetaTitle:       "Fitur — Imaginary Liquidity",
		MetaDescription: "Alur kerja, kemampuan, dan batasan Imaginary Liquidity: analisis likuiditas, rencana eksekusi, dan kendali akun broker.",
		Sections: []Section{
			{
				Key:   "alur",
				Kind:  KindSteps,
				Title: "Empat langkah dari data pasar sampai akun Anda",
				Lead:  "Setiap langkah punya keluaran yang bisa diperiksa. Tidak ada tahap yang berjalan tanpa jejak.",
				Items: []Item{
					{
						Label: "Pemantauan pasar",
						Value: "Sistem mengamati instrumen yang termasuk dalam paket Anda dan menandai area likuiditas yang terbentuk dari perilaku harga sebelumnya.",
					},
					{
						Label: "Penyusunan rencana",
						Value: "Dari area yang tertandai, sistem menyusun rencana: titik masuk, batas kerugian, dan sasaran. Rencana selalu memuat ketiganya, bukan hanya arah.",
					},
					{
						Label: "Penyampaian sinyal",
						Value: "Rencana dikirim ke panel dan kanal notifikasi Anda. Pada paket dengan jeda, sinyal tayang lebih dulu di mode simulasi sebelum dapat dieksekusi.",
					},
					{
						Label: "Eksekusi di akun Anda",
						Value: "Eksekusi berjalan di akun broker Anda sendiri. Anda tetap pemegang akun, dan sistem hanya mengirim perintah sesuai kewenangan yang Anda berikan.",
					},
				},
			},
			{
				Key:   "kemampuan",
				Kind:  KindFeatures,
				Title: "Kemampuan utama",
				Lead:  "Daftar ini menjelaskan apa yang sistem lakukan beserta batasnya. Batasan ditulis apa adanya, bukan sebagai catatan kaki.",
				Items: []Item{
					{
						Label: "Pemetaan likuiditas",
						Value: "Menandai area di mana harga sebelumnya berbalik atau tertahan, lalu memakai area itu sebagai acuan rencana.",
						Note:  "Area yang terbentuk dari data lampau tidak menjamin harga akan berbalik di sana pada kesempatan berikutnya.",
						Icon:  "layers",
					},
					{
						Label: "Rencana dengan batas kerugian",
						Value: "Setiap rencana membawa batas kerugian dan sasaran sejak awal, sehingga risiko satu posisi sudah ditentukan sebelum posisi dibuka.",
						Note:  "Batas kerugian dapat terlampaui saat pasar melompati harga (gap) atau likuiditas menipis.",
						Icon:  "shield",
					},
					{
						Label: "Notifikasi yang dapat ditindaklanjuti",
						Value: "Sinyal dikirim lewat panel dan kanal yang Anda hubungkan, lengkap dengan konteks dan perintah yang dapat dijalankan atau diabaikan.",
						Note:  "Keterlambatan pengiriman kanal pihak ketiga berada di luar kendali kami.",
						Icon:  "bell",
					},
					{
						Label: "Kendali atas eksekusi",
						Value: "Paket dengan otomasi penuh dapat mengeksekusi sendiri; paket lain menunggu konfirmasi Anda untuk setiap sinyal.",
						Note:  "Kewenangan eksekusi harus Anda berikan sendiri; sistem tidak pernah mendapatkannya secara diam-diam.",
						Icon:  "zap",
					},
					{
						Label: "Pembatasan risiko per akun",
						Value: "Jumlah akun, instrumen, dan pola eksekusi dibatasi sesuai paket agar risiko tidak menumpuk tanpa disadari.",
						Note:  "Pembatasan ini mengatur sistem, bukan membatasi kerugian yang dapat timbul dari keputusan Anda sendiri.",
						Icon:  "gauge",
					},
					{
						Label: "Catatan aktivitas",
						Value: "Setiap sinyal, konfirmasi, dan eksekusi tercatat sehingga dapat Anda tinjau kembali bersama hasilnya.",
						Note:  "Catatan aktivitas adalah alat pemeriksaan, bukan laporan kinerja atau bukti keuntungan.",
						Icon:  "history",
					},
				},
			},
			{
				Key:   "pengaman",
				Kind:  KindProse,
				Title: "Pengaman yang berjalan bersamaan",
				Lead:  "Pengaman bertugas menahan, bukan menambah. Ia bekerja tanpa perlu Anda aktifkan.",
				Body: `- **Persetujuan bertahap.** Dokumen risiko baru diminta ketika Anda menghubungkan akun broker nyata, saat risikonya benar-benar mulai berjalan.
- **Penahanan saat pemeliharaan.** Selama mode pemeliharaan aktif, eksekusi dan pembayaran ditahan sampai persetujuan Anda diperbarui.
- **Pemisahan kewenangan.** Perubahan pada akun, pembayaran, dan pengaturan internal dicatat di jejak audit yang tidak dapat dihapus dari panel.
- **Batas eksekusi per akun.** Bila satu akun dibatasi atau dibekukan, sinyal untuk akun itu tidak dieksekusi.`,
			},
			{
				Key:   "batasan",
				Kind:  KindCallout,
				Title: "Yang tidak dilakukan sistem",
				Body: `Imaginary Liquidity tidak menjamin hasil, tidak menampilkan proyeksi keuntungan, dan tidak mengelola dana Anda. Dana tetap berada di akun broker atas nama Anda. Sistem juga tidak menggantikan penilaian Anda: sinyal adalah usulan yang dapat diabaikan, dan keputusan akhir selalu berada di tangan Anda.

Perdagangan dengan leverage dapat menimbulkan kerugian melebihi ekspektasi Anda. Baca bagian risiko sebelum menghubungkan akun nyata.`,
			},
			{
				Key:   "faq",
				Kind:  KindFaq,
				Title: "Pertanyaan yang sering diajukan",
				Items: []Item{
					{
						Label: "Apakah sistem memegang dana saya?",
						Value: "Tidak. Dana tetap berada di akun broker Anda. Sistem hanya mengirim perintah eksekusi sesuai kewenangan yang Anda berikan.",
					},
					{
						Label: "Apakah saya bisa memakai akun yang sudah ada?",
						Value: "Bisa. Anda menghubungkan akun broker yang sudah Anda miliki, tanpa memindahkan dana ke tempat lain.",
					},
					{
						Label: "Apa yang terjadi bila sinyal terlewat?",
						Value: "Sinyal yang tidak dieksekusi tidak menghasilkan posisi. Tidak ada pengejaran harga otomatis, karena keputusan itu tetap milik Anda.",
					},
				},
			},
		},
	}
}

func pricing() Seed {
	return Seed{
		Slug:            "harga",
		Eyebrow:         "Harga",
		Title:           "Harga yang jelas, tanpa biaya tersembunyi.",
		Lead:            "Setiap paket menampilkan manfaat dan batasannya. Biaya yang tampil adalah biaya yang Anda bayar; tidak ada potongan tambahan di luar yang tercantum.",
		MetaTitle:       "Harga — Imaginary Liquidity",
		MetaDescription: "Paket langganan Imaginary Liquidity beserta manfaat, batasan, dan kebijakan pengembalian dana.",
		Sections: []Section{
			{
				Key:   "perbandingan",
				Kind:  KindTable,
				Title: "Perbandingan paket",
				Lead:  "Tabel ini disusun dari konfigurasi paket yang sama dengan yang dipakai sistem saat mengeksekusi.",
			},
			{
				Key:   "faq",
				Kind:  KindFaq,
				Title: "Pertanyaan seputar pembayaran",
				Lead:  "Ketentuan pengembalian dana berlaku penuh pada semua paket dan dijelaskan lebih lanjut di halaman kebijakan.",
				Items: []Item{
					{
						Label: "Kapan tagihan berikutnya ditarik?",
						Value: "Langganan berlaku satu bulan dan diperpanjang pada tanggal yang sama. Anda menerima pemberitahuan sebelum perpanjangan berjalan.",
					},
					{
						Label: "Bisakah saya berhenti kapan saja?",
						Value: "Bisa. Paket aktif sampai akhir periode yang sudah dibayar, lalu akun kembali ke paket gratis tanpa penghentian layanan mendadak.",
					},
					{
						Label: "Apakah ada masa percobaan?",
						Value: "Ada. Paket gratis dapat dipakai tanpa biaya dan tanpa batas waktu, dengan batasan jumlah akun dan instrumen yang lebih sempit.",
					},
					{
						Label: "Apakah biaya broker terpisah?",
						Value: "Ya. Komisi, spread, dan biaya swap ditetapkan oleh broker Anda dan tidak termasuk dalam biaya langganan.",
					},
				},
			},
			{
				Key:   "catatan",
				Kind:  KindCallout,
				Title: "Catatan sebelum memilih",
				Body:  `Biaya langganan tidak sebanding dengan hasil trading, dan tidak ada paket yang menjamin keuntungan. Pilih paket berdasarkan seberapa banyak kendali yang ingin Anda pegang, bukan berdasarkan target keuntungan.`,
			},
		},
	}
}

func about() Seed {
	return Seed{
		Slug:            "tentang",
		Eyebrow:         "Tentang",
		Title:           "Membaca pasar dengan lebih tenang.",
		Lead:            "Imaginary Liquidity dibangun untuk satu alasan sederhana: keputusan trading yang tenang lebih mungkin dipertahankan daripada keputusan yang terburu-buru. Kami bukan pialang, bukan penasihat keuangan, dan tidak mengelola dana siapa pun.",
		MetaTitle:       "Tentang — Imaginary Liquidity",
		MetaDescription: "Prinsip kerja Imaginary Liquidity, apa yang kami bangun, dan hal-hal yang sengaja tidak kami janjikan.",
		Sections: []Section{
			{
				Key:   "ringkas",
				Kind:  KindStats,
				Title: "Ringkas",
				Items: []Item{
					{Label: "Peran kami", Value: "Alat bantu analisis dan eksekusi, bukan pengelola dana."},
					{Label: "Akun Anda", Value: "Tetap di broker Anda sendiri, atas nama Anda."},
					{Label: "Panduan waktu", Value: "Jam pasar mengikuti New York; hitungan hari Anda mengikuti WIB."},
				},
			},
			{
				Key:   "bangun",
				Kind:  KindCards,
				Title: "Apa yang kami bangun",
				Items: []Item{
					{
						Label: "Alat yang menjelaskan dirinya",
						Value: "Setiap sinyal membawa alasan dan batas risikonya. Alat yang tidak bisa menjelaskan mengapa ia mengambil tindakan sulit dipertanggungjawabkan.",
						Icon:  "layers",
					},
					{
						Label: "Kendali yang tetap di tangan Anda",
						Value: "Kewenangan eksekusi diberikan oleh Anda, dan dapat dicabut kapan saja tanpa menutup akun.",
						Icon:  "shield",
					},
					{
						Label: "Catatan yang dapat diperiksa",
						Value: "Aktivitas tercatat dan dapat ditinjau, sehingga penilaian atas kinerja bertumpu pada data, bukan ingatan.",
						Icon:  "history",
					},
				},
			},
			{
				Key:   "prinsip",
				Kind:  KindProse,
				Title: "Prinsip yang kami pegang",
				Body: `**Tidak menjanjikan hasil.** Keuntungan tidak bisa dijanjikan oleh siapa pun. Yang bisa kami kerjakan adalah membuat prosesnya lebih tertata dan lebih mudah diperiksa.

**Menjelaskan batasan.** Setiap kemampuan di situs ini ditulis bersama batasannya. Kami memilih pengguna yang memahami risikonya daripada pengguna yang tertarik pada janji.

**Data Anda bukan komoditas.** Data akun dipakai untuk menjalankan layanan dan memenuhi kewajiban hukum, bukan untuk dijual atau dipakai menargetkan iklan.

**Perubahan yang tercatat.** Perubahan pada isi kebijakan, harga, dan konfigurasi akun dicatat agar dapat ditelusuri kembali.`,
			},
			{
				Key:   "tidak",
				Kind:  KindCallout,
				Title: "Yang tidak kami janjikan",
				Body: `Kami tidak menjanjikan keuntungan, tidak menampilkan proyeksi hasil, dan tidak memberi rekomendasi beli atau jual atas instrumen tertentu. Kami juga tidak menjanjikan bahwa setiap sinyal akan menguntungkan — sebagian sinyal memang berakhir dengan kerugian, dan itu bagian dari cara kerja trading yang jujur diakui.

Bila ada pihak yang menjanjikan keuntungan atas nama kami, itu bukan pernyataan resmi kami.`,
			},
		},
	}
}

func help() Seed {
	return Seed{
		Slug:            "bantuan",
		Eyebrow:         "Pusat Bantuan",
		Title:           "Panduan yang bisa diikuti langkah demi langkah.",
		Lead:            "Mulai dari menyiapkan akun, menghubungkan broker, sampai membaca tagihan. Bila ada yang belum terjawab, tim dukungan membalas pada hari kerja.",
		MetaTitle:       "Pusat Bantuan — Imaginary Liquidity",
		MetaDescription: "Panduan memulai, mengatur strategi, menghubungkan akun broker, tagihan, dan keamanan akun Imaginary Liquidity.",
		Sections: []Section{
			{
				Key:   "topik",
				Kind:  KindCards,
				Title: "Topik panduan",
				Lead:  "Pilih topik yang paling dekat dengan pertanyaan Anda.",
				Items: []Item{
					{
						Label: "Memulai",
						Value: "Membuat akun, menyelesaikan verifikasi surel, dan mengenali tampilan utama panel.",
						Icon:  "book",
						Href:  "/bantuan/memulai",
					},
					{
						Label: "Strategi",
						Value: "Memilih instrumen yang dipantau, membaca sinyal, dan meninjau hasil eksekusi.",
						Icon:  "layers",
						Href:  "/bantuan/strategi",
					},
					{
						Label: "Akun broker",
						Value: "Menghubungkan akun yang sudah ada, mengatur kewenangan eksekusi, dan membukanya kembali.",
						Icon:  "plug",
						Href:  "/bantuan/akun-broker",
					},
					{
						Label: "Tagihan",
						Value: "Membaca invoice, memahami perpanjangan, dan mengajukan pengembalian dana.",
						Icon:  "card",
						Href:  "/bantuan/tagihan",
					},
					{
						Label: "Keamanan",
						Value: "Mengaktifkan autentikasi dua faktor, mengelola sesi perangkat, dan kode pemulihan.",
						Icon:  "shield",
						Href:  "/bantuan/keamanan",
					},
					{
						Label: "Zona waktu",
						Value: "Memahami mengapa jam pasar mengikuti New York sementara hitungan hari Anda mengikuti WIB.",
						Icon:  "clock",
						Href:  "/bantuan/zona-waktu",
					},
				},
			},
			{
				Key:   "kontak",
				Kind:  KindProse,
				Title: "Bila panduan belum menjawab",
				Body: `Kirim pertanyaan Anda ke **dukungan@rekavia.com**. Sertakan alamat surel akun dan, bila menyangkut transaksi, nomor invoice agar kami dapat menelusuri tanpa meminta data tambahan.

Kami membalas pada hari kerja. Untuk kejadian yang berdampak pada eksekusi, tuliskan "URGENT" pada judul surel.`,
			},
		},
	}
}

func legal() Seed {
	return Seed{
		Slug:            "legal",
		Eyebrow:         "Legal",
		Title:           "Dokumen yang mengatur hubungan kita.",
		Lead:            "Seluruh dokumen resmi Imaginary Liquidity dikumpulkan di satu tempat. Setiap dokumen bernomor agar dapat dirujuk, dicetak, dan ditautkan langsung ke bagian tertentu.",
		MetaTitle:       "Dokumen Legal — Imaginary Liquidity",
		MetaDescription: "Kebijakan privasi, syarat layanan, disclaimer risiko, kebijakan pengembalian dana, dan penggunaan wajar Imaginary Liquidity.",
		Sections: []Section{
			{
				Key:   "kapan",
				Kind:  KindTable,
				Title: "Kapan persetujuan diminta",
				Lead:  "Dokumen tidak diminta sekaligus. Persetujuan mengikuti kewajiban yang relevan saat itu.",
			},
			{
				Key:   "catatan",
				Kind:  KindCallout,
				Title: "Versi yang berlaku",
				Body:  `Dokumen yang tampil di sini adalah versi yang sedang berlaku beserta tanggal berlakunya. Versi lama tetap tersimpan, sehingga persetujuan yang pernah Anda berikan dapat ditelusuri kembali.`,
			},
		},
	}
}
