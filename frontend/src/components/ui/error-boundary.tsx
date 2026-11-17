import { Component, type ErrorInfo, type ReactNode } from 'react'
import { Button } from './button'

type ErrorBoundaryProps = {
  children: ReactNode
  fallbackTitle?: string
}

type ErrorBoundaryState = {
  hasError: boolean
}

/**
 * ErrorBoundary menahan kesalahan render pada satu cabang komponen supaya
 * aplikasi tidak berubah menjadi layar putih kosong di hadapan pengguna.
 */
export class ErrorBoundary extends Component<ErrorBoundaryProps, ErrorBoundaryState> {
  state: ErrorBoundaryState = { hasError: false }

  static getDerivedStateFromError(): ErrorBoundaryState {
    return { hasError: true }
  }

  componentDidCatch(error: Error, info: ErrorInfo): void {
    console.error('Kesalahan tampilan tertangkap ErrorBoundary:', error, info.componentStack)
  }

  render() {
    if (this.state.hasError) {
      return (
        <div className="flex min-h-[50vh] flex-col items-center justify-center gap-3 p-6 text-center">
          <h2 className="text-base font-semibold text-foreground">
            {this.props.fallbackTitle ?? 'Tampilan mengalami gangguan'}
          </h2>
          <p className="max-w-md text-xs text-muted-foreground">
            Bagian halaman ini gagal dimuat. Muat ulang halaman untuk melanjutkan tanpa kehilangan sesi Anda.
          </p>
          <Button
            size="sm"
            variant="outline"
            onClick={() => {
              this.setState({ hasError: false })
              window.location.reload()
            }}
          >
            Muat Ulang Halaman
          </Button>
        </div>
      )
    }

    return this.props.children
  }
}
