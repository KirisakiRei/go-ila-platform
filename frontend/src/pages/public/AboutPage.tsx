import { PublicContentPage } from './ContentPage'
import { pageTitle } from './lib/page-title'

/**
 * AboutPage menampilkan isi halaman Tentang yang dikelola admin.
 *
 * Bagian "yang tidak kami janjikan" ditandai sebagai callout sehingga tidak
 * pernah hilang dari halaman: itu bagian yang paling sering sengaja dicari.
 */
export default function AboutPage() {
  return <PublicContentPage slug="tentang" fallbackTitle={pageTitle('tentang')} />
}
