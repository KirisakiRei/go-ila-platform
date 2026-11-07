import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui'
import type { Tier } from '@/types/api'

const TIER_DESCRIPTION: Record<Tier, string> = {
  free: 'Sinyal dengan jeda 15 menit dan simulasi tanpa akun nyata.',
  basic: 'Sinyal real-time pada satu akun dengan eksekusi manual.',
  plus: 'Eksekusi otomatis, dua akun, dan pengaturan risiko sendiri.',
}

export function TierCard({ tier }: { tier: Tier }) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Tier Anda</CardTitle>
        <CardDescription>{TIER_DESCRIPTION[tier]}</CardDescription>
      </CardHeader>
      <CardContent>
        <p className="text-[13px] text-muted-foreground">
          Perubahan tier hanya mengatur akses, tidak mengubah isi sinyal. Posisi yang sudah
          berjalan tetap berjalan sampai batas yang dipasang di broker.
        </p>
      </CardContent>
    </Card>
  )
}
