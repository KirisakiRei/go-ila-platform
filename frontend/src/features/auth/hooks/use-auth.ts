import { useMutation } from '@tanstack/react-query'
import { useNavigate } from 'react-router-dom'
import { authApi } from '@/lib/auth-api'
import type { LoginPayload, RegisterPayload } from '@/types/api'
import { useAuthContext } from '../auth-context'

/** Mutation untuk masuk; mengisi sesi lalu mengalihkan ke dashboard. */
export function useSignIn() {
  const { signIn } = useAuthContext()
  const navigate = useNavigate()

  return useMutation({
    mutationFn: (payload: LoginPayload) => authApi.login(payload),
    onSuccess: (result) => {
      signIn(result.token, result.user)
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
