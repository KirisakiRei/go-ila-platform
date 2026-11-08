import { PublicContentPage } from './ContentPage'
import { pageTitle } from './lib/page-title'

/**
 * FeaturesPage menampilkan isi halaman Fitur yang dikelola admin.
 *
 * Kerangka tampilan ada di `PageRenderer`, isinya di basis data. Dengan begitu
 * mengubah penjelasan fitur atau batasannya tidak memerlukan deploy ulang.
 */
export default function FeaturesPage() {
  return <PublicContentPage slug="fitur" fallbackTitle={pageTitle('fitur')} />
}
