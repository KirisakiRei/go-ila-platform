import { LoginForm } from '@/features/auth/components/LoginForm'

export default function LoginPage() {
  return (
    <main className="flex min-h-full items-center justify-center px-4 py-12">
      <section className="w-full max-w-sm rounded-xl border border-border bg-card p-6 shadow-sm">
        <div className="mb-6 flex items-center gap-2">
          <span className="text-lg font-bold tracking-tight">ILA</span>
          <span className="text-[13px] text-muted-foreground">Bot</span>
        </div>
        <LoginForm />
      </section>
    </main>
  )
}
