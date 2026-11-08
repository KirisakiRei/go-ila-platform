import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { PublicApp } from '@/app/PublicApp'
import '@/styles/globals.css'

const container = document.getElementById('root')

if (!container) {
  throw new Error('Elemen #root tidak ditemukan. Periksa index.html.')
}

createRoot(container).render(
  <StrictMode>
    <PublicApp />
  </StrictMode>,
)
