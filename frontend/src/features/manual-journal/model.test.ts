import { describe, expect, it } from 'vitest'
import {
  computeTradePreview,
  emptyTrade,
  filterTrades,
  friendlyJournalError,
  fromWib,
  localWib,
  normalizeDecimalInput,
  safeLink,
  tradeDay,
  validateDecimal,
  validateTrade,
  validateTradeFields,
  wibDay,
} from './model'
import type { JournalFilters, JournalTrade } from './types'

const filters: JournalFilters = {
  account_id: 'a1',
  from: '',
  to: '',
  status: '',
  asset: '',
  setup: '',
  tag: '',
  emotion: '',
  mistake: '',
}
const trade: JournalTrade = { ...emptyTrade('a1'), id: 't1', created_at: '2026-11-25T18:00:00Z' }

describe('manual journal domain', () => {
  it('uses WIB boundaries independent of browser timezone and round trips local input', () => {
    expect(wibDay('2026-11-25T17:00:00Z')).toBe('2026-11-26')
    expect(localWib('2026-11-25T17:00:00Z')).toBe('2026-11-26T00:00')
    expect(fromWib('2026-11-26T00:00')).toBe('2026-11-25T17:00:00.000Z')
    expect(tradeDay(trade)).toBe('2026-11-26')
  })
  it('filters draft creation dates and closed exit dates with the same inclusive WIB window', () => {
    const closed = {
      ...trade,
      id: 't2',
      status: 'closed' as const,
      opened_at: '2026-11-24T00:00:00Z',
      closed_at: '2026-11-25T17:00:00Z',
      asset: 'XAUUSD',
      setup: 'Breakout',
      tags: ['London'],
    }
    expect(
      filterTrades([trade, closed], { ...filters, from: '2026-11-26', to: '2026-11-26' }),
    ).toHaveLength(2)
    expect(filterTrades([trade, closed], { ...filters, asset: 'xau', tag: 'london' })).toEqual([
      closed,
    ])
    expect(filterTrades([trade], { ...filters, to: '2026-11-25' })).toEqual([])
  })
  it('preserves decimal strings and rejects exponent, excess precision, negative unsigned and overflow', () => {
    expect(validateDecimal('0.0001')).toBe(true)
    expect(validateDecimal('-123.4500', 4, true)).toBe(true)
    for (const value of ['1e3', 'NaN', '-1', '1.00001', '1000000000.0001', ''])
      expect(validateDecimal(value)).toBe(false)
    expect(validateDecimal('0.12345678', 8)).toBe(true)
  })
  it('allows incomplete drafts, requires complete open and accepts zero closed pnl', () => {
    expect(validateTrade(emptyTrade('a1'))).toBe('')
    expect(validateTrade({ ...emptyTrade('a1'), size: '0' })).toMatch(/lebih dari 0/)
    expect(validateTrade({ ...emptyTrade('a1'), stop_loss: '0' })).toMatch(/lebih dari 0/)
    expect(validateTrade({ ...emptyTrade('a1'), risk_amount: '0', capital_snapshot: '0' })).toBe('')
    expect(validateTrade({ ...emptyTrade('a1'), status: 'open' })).not.toBe('')
    const open = {
      ...emptyTrade('a1'),
      status: 'open' as const,
      asset: 'XAUUSD',
      opened_at: '2026-11-25T10:00:00Z',
      entry_price: '2600.00000001',
      size: '0.01',
      size_unit: 'lot',
    }
    expect(validateTrade(open)).toBe('')
    expect(
      validateTrade({ ...open, status: 'closed', closed_at: '2026-11-25T11:00:00Z', net_pnl: '0' }),
    ).toBe('')
    expect(validateTrade({ ...open, net_pnl: '1' })).toMatch(/hanya/)
    expect(
      validateTrade({ ...open, status: 'closed', closed_at: '2026-11-24T11:00:00Z', net_pnl: '0' }),
    ).toMatch(/sebelum/)
  })
  it('only accepts credential-free HTTPS TradingView links', () => {
    expect(safeLink('https://www.tradingview.com/x/test')).toBe(true)
    for (const url of [
      'javascript:alert(1)',
      'https://tradingview.com.evil.test/x',
      'https://user:pass@tradingview.com/x',
      'http://tradingview.com/x',
    ])
      expect(safeLink(url)).toBe(false)
  })

  it('computes live preview metrics honestly for long/short and omits incomplete ones', () => {
    const longMetrics = computeTradePreview({
      side: 'long',
      entry_price: '100',
      stop_loss: '90',
      take_profit: '130',
      net_pnl: '200',
      risk_amount: '100',
      capital_snapshot: '5000',
      opened_at: '2026-11-25T10:00:00Z',
      closed_at: '2026-11-25T12:15:00Z',
    })
    expect(longMetrics).toEqual({
      plannedRR: '1:3.00',
      realizedR: '+2.00R',
      riskPercent: '2.00%',
      gainPercent: '+4.00%',
      holdDuration: '2j 15m',
    })

    const invalidShort = computeTradePreview({
      side: 'short',
      entry_price: '100',
      stop_loss: '90',
      take_profit: '80',
      net_pnl: '-50',
      risk_amount: null,
      capital_snapshot: null,
      opened_at: null,
      closed_at: null,
    })
    expect(invalidShort.plannedRR).toBeNull()
    expect(invalidShort.realizedR).toBeNull()
    expect(invalidShort.holdDuration).toBeNull()
  })

  it('normalizes decimal input, returns human field errors, and translates server errors', () => {
    expect(normalizeDecimalInput(' 1000,50 ')).toBe('1000.50')
    expect(normalizeDecimalInput('+45,25', true)).toBe('45.25')
    expect(normalizeDecimalInput('   ')).toBeNull()

    const badCap = validateTradeFields({ ...emptyTrade('a1'), capital_snapshot: 'abc' })
    expect(badCap.firstTab).toBe('review')
    expect(badCap.errors.capital_snapshot).toMatch(/angka valid/i)
    expect(badCap.summary).toContain('Modal saat entry')
    expect(badCap.summary).not.toContain('capital_snapshot')

    expect(friendlyJournalError(new Error('account is archived'))).toMatch(/diarsipkan/i)
    expect(friendlyJournalError(new Error('closed_at precedes opened_at'))).toMatch(/sebelum waktu masuk/i)
  })

  it('guards TP/SL direction against Entry and risk_amount against capital_snapshot', () => {
    const badBuy = validateTradeFields({
      ...emptyTrade('a1'),
      side: 'long',
      entry_price: '3500',
      stop_loss: '3510',
      take_profit: '3480',
    })
    expect(badBuy.errors.stop_loss).toMatch(/Buy.*lebih rendah/i)
    expect(badBuy.errors.take_profit).toMatch(/Buy.*lebih tinggi/i)

    const badSell = validateTradeFields({
      ...emptyTrade('a1'),
      side: 'short',
      entry_price: '3500',
      stop_loss: '3490',
      take_profit: '3520',
    })
    expect(badSell.errors.stop_loss).toMatch(/Sell.*lebih tinggi/i)
    expect(badSell.errors.take_profit).toMatch(/Sell.*lebih rendah/i)

    const validSell = validateTradeFields({
      ...emptyTrade('a1'),
      side: 'short',
      entry_price: '3500',
      stop_loss: '3510',
      take_profit: '3480',
    })
    expect(validSell.summary).toBe('')

    const badRisk = validateTradeFields({
      ...emptyTrade('a1'),
      risk_amount: '1500',
      capital_snapshot: '1000',
    })
    expect(badRisk.errors.risk_amount).toMatch(/melebihi modal/i)
  })
})
