/// <reference types="vite/client" />

/*
 * Deklarasi tipe untuk aset yang diimpor langsung dari `src/`.
 *
 * Vite menangani impor gambar dengan mengubahnya menjadi URL saat build, tapi
 * TypeScript tidak tahu soal itu. Tanpa deklarasi di bawah, setiap `import`
 * berkas gambar dianggap modul yang tidak ada dan `tsc --noEmit` gagal.
 *
 * Aset di `public/` tidak butuh ini karena dirujuk sebagai path string biasa.
 */
declare module '*.png' {
  const src: string
  export default src
}

declare module '*.jpg' {
  const src: string
  export default src
}

declare module '*.jpeg' {
  const src: string
  export default src
}

declare module '*.svg' {
  const src: string
  export default src
}

declare module '*.webp' {
  const src: string
  export default src
}
