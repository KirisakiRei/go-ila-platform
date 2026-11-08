import { useMutation } from '@tanstack/react-query'
import { useNavigate } from 'react-router-dom'
import { authApi } from '@/lib/auth-api'
import type { LoginPayload, RegisterPayload } from '@/types/api'
import { useAuthContext } from '../auth-context'

/** Mutation untuk masuk; sesi disimpan backend, UI hanya menerima data akun. */
export function useSignIn() {
  const { signIn } = useAuthContext()
  const navigate = useNavigate()

  return useMutation({
    mutationFn: (payload: LoginPayload) => authApi.login(payload),
    // Sesi hanya diterbitkan bila langkah login benar-benar tuntas. Tanpa
    // pemeriksaan ini, akun ber-2FA akan masuk tanpa memasukkan kode.
    onSuccess: (result) => {
      if (result.step === 'two_factor_required') return
      signIn(result.user)
      navigate('/', { replace: true })
    },
  })
}

/** Menuntaskan masuk dengan kode 2FA setelah challenge diterbitkan. */
export function useVerifyTwoFactor() {
  const { signIn } = useAuthContext()
  const navigate = useNavigate()

  return useMutation({
    mutationFn: ({ challengeId, code }: { challengeId: string; code: string }) =>
      authApi.verifyTwoFactor(challengeId, code),
    onSuccess: (result) => {
      signIn(result.user)
      navigate('/', { replace: true })
    },
  })
}

/** Mutation untuk mendaftar; akun baru belum memiliki sesi. */
export function useSignUp() {
  return useMutation({
    mutationFn: (payload: RegisterPayload) => authApi.register(payload),
  })
}

/** Aksi keluar untuk dipakai komponen mana pun di dalam provider. */
export function useSignOut() {
  return useAuthContext().signOut
}

export { useAuthContext }
