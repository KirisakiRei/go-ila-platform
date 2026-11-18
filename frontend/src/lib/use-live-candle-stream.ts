import { useEffect, useState } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { marketApi } from './api'
import { queryKeys } from './queries'
import type { MarketCandle, MarketCandlesResponse } from '@/types/api'

type LiveStreamParams = {
  symbol: string | null
  timeframe: string | undefined
  limit?: number
  enabled?: boolean
}

function mergeLiveCandle(
  existing: MarketCandle[],
  incoming: MarketCandle,
  maxLimit: number,
): MarketCandle[] {
  if (!Number.isFinite(incoming.time) || incoming.time <= 0) {
    return existing
  }
  if (existing.length === 0) {
    return [incoming]
  }
  const lastIdx = existing.length - 1
  const last = existing[lastIdx]
  if (last && last.time === incoming.time) {
    if (
      last.open === incoming.open &&
      last.high === incoming.high &&
      last.low === incoming.low &&
      last.close === incoming.close &&
      last.volume === incoming.volume
    ) {
      return existing
    }
    const next = existing.slice()
    next[lastIdx] = incoming
    return next
  }
  if (last && incoming.time > last.time) {
    const next = [...existing, incoming]
    return next.length > maxLimit ? next.slice(next.length - maxLimit) : next
  }
  if (lastIdx >= 1) {
    const prev = existing[lastIdx - 1]
    if (prev && prev.time === incoming.time) {
      const next = existing.slice()
      next[lastIdx - 1] = incoming
      return next
    }
  }
  return existing
}

/**
 * useLiveCandleStream menghubungkan chart aktif ke endpoint SSE berbagi-pakai.
 *
 * Otomatis berhenti saat chart dinonaktifkan (mis. mode potongan histori,
 * accordion mobile tertutup, atau tab peramban tidak terlihat) sehingga tidak
 * membebani server maupun baterai perangkat.
 */
export function useLiveCandleStream({
  symbol,
  timeframe,
  limit = 200,
  enabled = true,
}: LiveStreamParams): { isLive: boolean } {
  const qc = useQueryClient()
  const [visible, setVisible] = useState<boolean>(() => {
    if (typeof document === 'undefined') return true
    return document.visibilityState !== 'hidden'
  })
  const [connectedKey, setConnectedKey] = useState<string | null>(null)

  useEffect(() => {
    if (typeof document === 'undefined') return
    const onVisChange = () => setVisible(document.visibilityState !== 'hidden')
    document.addEventListener('visibilitychange', onVisChange)
    return () => document.removeEventListener('visibilitychange', onVisChange)
  }, [])

  const normSym = (symbol ?? '').trim().toUpperCase()
  const normTf = (timeframe ?? '').trim().toUpperCase()
  const streamKey = normSym && normTf ? `${normSym}|${normTf}|${limit}` : ''
  const shouldConnect =
    enabled &&
    visible &&
    Boolean(streamKey) &&
    typeof window !== 'undefined' &&
    typeof window.EventSource !== 'undefined'

  useEffect(() => {
    if (!shouldConnect || !normSym || !normTf) {
      return
    }

    const url = marketApi.candlesStreamUrl(normSym, normTf)
    let closed = false
    let errorCount = 0
    const es = new window.EventSource(url, { withCredentials: true })

    const onReady = () => {
      if (closed) return
      errorCount = 0
      setConnectedKey(streamKey)
    }

    const onCandle = (ev: MessageEvent) => {
      if (closed || typeof ev.data !== 'string') return
      try {
        const parsed = JSON.parse(ev.data) as MarketCandle
        if (
          typeof parsed.time !== 'number' ||
          typeof parsed.open !== 'number' ||
          typeof parsed.high !== 'number' ||
          typeof parsed.low !== 'number' ||
          typeof parsed.close !== 'number'
        ) {
          return
        }
        errorCount = 0
        setConnectedKey(streamKey)

        const qKey = queryKeys.marketCandles(normSym, timeframe ?? normTf, limit)
        qc.setQueryData<MarketCandlesResponse>(qKey, (prev) => {
          if (!prev) return prev
          const updated = mergeLiveCandle(prev.candles, parsed, Math.max(limit, 200))
          if (updated === prev.candles) return prev
          return {
            ...prev,
            candles: updated,
          }
        })
      } catch {
        // Abaikan frame tidak valid tanpa memutus stream.
      }
    }

    const onError = () => {
      if (closed) return
      setConnectedKey((curr) => (curr === streamKey ? null : curr))
      errorCount += 1
      if (errorCount >= 2) {
        // Bila kapasitas stream penuh atau offline, tutup dengan tenang dan
        // biarkan penyegaran berkala REST mengambil alih.
        closed = true
        es.close()
      }
    }

    es.addEventListener('ready', onReady as EventListener)
    es.addEventListener('candle', onCandle as EventListener)
    es.onerror = onError

    return () => {
      closed = true
      setConnectedKey((curr) => (curr === streamKey ? null : curr))
      es.removeEventListener('ready', onReady as EventListener)
      es.removeEventListener('candle', onCandle as EventListener)
      es.close()
    }
  }, [shouldConnect, normSym, normTf, timeframe, limit, streamKey, qc])

  return {
    isLive: shouldConnect && connectedKey === streamKey,
  }
}
