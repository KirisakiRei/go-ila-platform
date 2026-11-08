import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { BrowserRouter } from 'react-router-dom'
import { AuthProvider } from '@/features/auth/AuthProvider'
import { AppRoutes } from './AppRoutes'

/**
 * QueryClient dibuat sekali di level modul agar cache tidak ter-reset
 * setiap render. Retry dibatasi karena kegagalan autentikasi tidak akan
 * membaik dengan pengulangan.
 */
const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: 1,
      staleTime: 30_000,
      refetchOnWindowFocus: false,
    },
    mutations: {
      retry: 0,
    },
  },
})

export function App() {
  return (
    <QueryClientProvider client={queryClient}>
      {/* Panel pelanggan dilayani di /app agar entri publik tetap di akar. */}
      <BrowserRouter basename="/app">
        <AuthProvider>
          <AppRoutes />
        </AuthProvider>
      </BrowserRouter>
    </QueryClientProvider>
  )
}
