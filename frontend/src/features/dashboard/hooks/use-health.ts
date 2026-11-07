import { useQuery } from '@tanstack/react-query'

export type HealthStatus = 'online' | 'offline'

/**
 * useHealth memeriksa kesiapan backend-api.
 * Kegagalan jaringan dianggap "offline", bukan error yang menghentikan halaman.
 */
export function useHealth() {
  return useQuery<HealthStatus>({
    queryKey: ['health'],
    queryFn: async () => {
      try {
        const response = await fetch('/health', { credentials: 'same-origin' })
        return response.ok ? 'online' : 'offline'
      } catch {
        return 'offline'
      }
    },
    staleTime: 30_000,
    refetchInterval: 60_000,
    retry: 0,
  })
}
