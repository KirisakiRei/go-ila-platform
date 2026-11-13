import type { PublicPlan } from '@/types/admin'

export type PlanBenefit = {
  label: string
  included: boolean
}

const TIER_AUDIENCE: Record<string, string> = {
  free: 'Untuk mengevaluasi cara kerja sinyal dalam mode demo.',
  basic: 'Untuk trader aktif yang eksekusi sendiri dengan konfirmasi sinyal.',
  plus: 'Untuk otomasi penuh di beberapa akun, dengan batas risiko per akun.',
}

/**
 * Mengubah flag teknis paket dari database menjadi daftar manfaat yang mudah
 * dipahami calon pengguna tanpa menampilkan nama variabel sistem mentah.
 */
export function getPlanAudience(plan: PublicPlan): string {
  return TIER_AUDIENCE[plan.code] ?? (plan.description || 'Paket langganan bulanan.')
}

export function formatPlanBenefits(plan: PublicPlan): PlanBenefit[] {
  const f = plan.features ?? {}
  const benefits: PlanBenefit[] = []

  const delay = f.signal_delay_minutes
  if (delay === '0') {
    benefits.push({ label: 'Sinyal tanpa jeda', included: true })
  } else if (delay) {
    benefits.push({ label: `Sinyal demo (jeda ${delay} menit)`, included: true })
  }

  const maxSymbols = f.max_symbols
  if (maxSymbols) {
    benefits.push({
      label:
        maxSymbols === '1'
          ? 'Pantau 1 pair utama'
          : `Pantau hingga ${maxSymbols} pair likuiditas utama`,
      included: true,
    })
  }

  const accountLimit = f.account_limit
  if (accountLimit) {
    benefits.push({
      label:
        accountLimit === '1'
          ? 'Terhubung ke 1 akun broker'
          : `Terhubung hingga ${accountLimit} akun broker`,
      included: true,
    })
  }

  const autoExec = f.auto_execute
  if (autoExec === 'true') {
    benefits.push({ label: 'Eksekusi otomatis & konfirmasi manual', included: true })
  } else if (autoExec === 'false') {
    benefits.push({
      label:
        plan.code === 'free'
          ? 'Mode demo tanpa eksekusi live'
          : 'Konfirmasi eksekusi manual via Telegram',
      included: true,
    })
  }

  if (plan.code === 'plus' || f.reason_ai === 'true') {
    benefits.push({ label: 'Ringkasan konteks setiap setup', included: true })
  } else if (plan.ila3_coming_soon && plan.code !== 'free') {
    benefits.push({ label: 'Strategi ILA 2.0 aktif', included: true })
  }

  if (plan.code === 'plus') {
    benefits.push({
      label: plan.ila3_coming_soon
        ? 'Akses prioritas ILA 3.0 (bertahap)'
        : 'Strategi ILA 2.0 & 3.0',
      included: true,
    })
  }

  return benefits.slice(0, 5)
}
