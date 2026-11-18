import { afterEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { Pagination } from '@/components/ui'
import { BacktestTradeTable } from '@/features/backtest/components/BacktestTradeTable'
import SecuritySettingsPage from '@/pages/SecuritySettingsPage'
import { PushNotificationPanel } from '@/features/notifications/PushNotificationPanel'
import { pushApi } from '@/lib/push-notifications'

function setViewportWidth(width: number) {
  Object.defineProperty(window, 'innerWidth', {
    writable: true,
    configurable: true,
    value: width,
  })
  window.dispatchEvent(new Event('resize'))
}

describe('Mobile UI/UX (360px) & Web Push', () => {
  afterEach(() => {
    setViewportWidth(1024)
    vi.restoreAllMocks()
  })

  it('Pagination merender mode ringkas bebas overflow di layar 360px', () => {
    setViewportWidth(360)
    const onChange = vi.fn()

    render(
      <Pagination
        page={4}
        totalPages={20}
        shown={10}
        total={200}
        unit="baris"
        onPageChange={onChange}
      />,
    )

    expect(screen.getByText('Hal 4 / 20')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Halaman berikutnya' }))
    expect(onChange).toHaveBeenCalledWith(5)
  })

  it('BacktestTradeTable merender kartu mobile (bukan tabel 10 kolom) pada layar 360px', () => {
    setViewportWidth(360)

    render(
      <MemoryRouter>
        <BacktestTradeTable
          runID="run-1"
          symbol="XAUUSD"
          trades={[
            {
              seq: 1,
              arah: 'BUY',
              waktu_masuk: 1746093600,
              waktu_keluar: 1746100800,
              entry: 2650.25,
              stop_loss: 2640.0,
              take_profit: 2670.75,
              rr_rencana: 2.0,
              hasil: 'TP',
              r: 2.0,
              durasi_detik: 7200,
              ekuitas_akhir: 10200,
              ambigu: false,
              ringkasan: '',
              langkah_berikut: '',
              chart_tersedia: true,
            },
          ]}
        />
      </MemoryRouter>,
    )

    expect(document.querySelector('table')).toBeNull()
    expect(screen.getByText('#1')).toBeInTheDocument()
    expect(screen.getByText('+2.00R')).toBeInTheDocument()
  })

  it('SecuritySettingsPage merender menu hub drill-down di layar 360px', () => {
    setViewportWidth(360)
    const qc = new QueryClient({ defaultOptions: { queries: { retry: 0 } } })

    render(
      <QueryClientProvider client={qc}>
        <MemoryRouter initialEntries={['/pengaturan/keamanan']}>
          <Routes>
            <Route path="/pengaturan/keamanan" element={<SecuritySettingsPage />} />
          </Routes>
        </MemoryRouter>
      </QueryClientProvider>,
    )

    expect(screen.getByText('Keamanan Akun')).toBeInTheDocument()
    expect(screen.getByText('Ubah Kata Sandi')).toBeInTheDocument()

    fireEvent.click(screen.getByText('Ubah Kata Sandi'))
    expect(screen.getByText('Kembali ke Menu Keamanan')).toBeInTheDocument()
    expect(screen.getByLabelText('Sandi saat ini')).toBeInTheDocument()
  })

  it('PushNotificationPanel menampilkan perangkat terdaftar dan saklar entry/exit', async () => {
    vi.spyOn(pushApi, 'status').mockResolvedValue({
      supported: true,
      vapid_public_key: 'BEl62iUYgUivxIkv69yViEuiBIa-Ib9-SkvMeAtA3LFgDzkrxZJjSgSnfckjBJuBkr3qBUYIHBQFLXYp5Nksh8U',
      subscriptions: [
        {
          id: 'sub-1',
          device_label: 'Chrome · Android',
          notify_entry: true,
          notify_exit: true,
          last_sent_at: null,
          created_at: '2026-11-18T06:00:00Z',
          updated_at: '2026-11-18T06:00:00Z',
          endpoint_host: 'fcm.googleapis.com',
        },
      ],
    })

    const qc = new QueryClient({ defaultOptions: { queries: { retry: 0 } } })
    render(
      <QueryClientProvider client={qc}>
        <PushNotificationPanel />
      </QueryClientProvider>,
    )

    expect(await screen.findByText('Chrome · Android')).toBeInTheDocument()
    expect(screen.getByText('Sinyal Siap & Entry Dibuka')).toBeInTheDocument()
    expect(screen.getByText('Posisi Tutup (TP / SL / Manual)')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /Kirim Tes Notifikasi/i })).toBeInTheDocument()
  })
})
