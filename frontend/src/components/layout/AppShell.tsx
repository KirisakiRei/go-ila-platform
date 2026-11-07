import type { ReactNode } from 'react'
import { Header } from './Header'
import type { User } from '@/types/api'

type AppShellProps = {
  user: User
  children: ReactNode
}

/** Kerangka halaman: header tetap di atas, konten terbatas lebarnya. */
export function AppShell({ user, children }: AppShellProps) {
  return (
    <div className="flex min-h-full flex-col">
      <Header user={user} />
      <main className="mx-auto w-full max-w-5xl flex-1 px-4 py-6">{children}</main>
    </div>
  )
}
