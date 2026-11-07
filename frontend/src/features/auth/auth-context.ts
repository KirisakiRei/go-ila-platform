import { createContext, useContext } from 'react'
import type { User } from '@/types/api'

export type AuthContextValue = {
  user: User | null
  isAuthenticated: boolean
  isLoading: boolean
  signIn: (token: string, user: User) => void
  signOut: () => void
}

/** Konteks sesi dipisah dari provider agar hot-reload tetap stabil. */
export const AuthContext = createContext<AuthContextValue | null>(null)

export function useAuthContext(): AuthContextValue {
  const context = useContext(AuthContext)
  if (!context) {
    throw new Error('useAuthContext harus dipakai di dalam AuthProvider')
  }
  return context
}
