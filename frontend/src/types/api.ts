/** Bentuk balasan seragam dari backend (pkg/httpx). */

export type Tier = 'free' | 'basic' | 'plus'
export type Role = 'user' | 'admin' | 'support'

export type ApiResponse<T> = {
  success: boolean
  message: string
  data?: T
  meta?: PaginationMeta
  error?: string
  details?: unknown
}

export type PaginationMeta = {
  page: number
  per_page: number
  total: number
  total_pages: number
  has_more?: boolean
  sort?: string
  order?: string
  /** Penanda total hanya perkiraan pada tabel besar; tampilkan dengan "~". */
  approx_total?: boolean
  /** Kursor halaman berikutnya pada daftar panjang. */
  next_cursor?: string
}

/**
 * TimePair menyajikan satu waktu dalam dua zona. Server yang menghitung,
 * bukan browser, supaya hasilnya sama di perangkat mana pun (plans/22).
 */
export type TimePair = {
  utc: string
  wib: string
  ny: string
}

export type User = {
  id: string
  name: string
  username: string
  phone: string
  email: string
  role: Role
  tier: Tier
  /** Memutuskan apakah modal "masih di sini?" ditampilkan. */
  idle_warning: boolean
  email_verified_at: string | null
  created_at: string
  updated_at: string
}

/** identifier menerima email atau username dalam satu kolom. */
export type LoginPayload = {
  identifier: string
  password: string
}

export type RegisterPayload = {
  name: string
  /** Wajib diisi backend hari ini; lihat plans/09 untuk aturan final. */
  username: string
  email: string
  phone: string
  password: string
  /** Wajib true: backend menolak pendaftaran tanpa persetujuan. */
  accept_terms: boolean
  /**
   * Kode dari tautan undangan. Wajib saat platform sedang invite only;
   * diabaikan pada pendaftaran mandiri biasa.
   */
  invite_code?: string
}

export type LoginResult = {
  /** "complete" berarti sesi sudah terbit; "two_factor_required" belum. */
  step?: 'complete' | 'two_factor_required'
  /** Diisi hanya saat langkah 2FA diperlukan. */
  challenge_id?: string
  expires_in_seconds?: number
  session_id?: string
  expires_at: string
  user: User
}

/** Hasil verifikasi 2FA: sama seperti login yang tuntas. */
export type VerifyTwoFactorResult = LoginResult & { step: 'complete' }

export type BrokerType = 'matchtrader' | 'ctrader'
export type AccountMode = 'manual' | 'auto'

/** Mode risiko per akun: persentase ekuitas atau ukuran lot tetap. */
export type RiskMode = 'percent' | 'lot'

/** Permintaan simulasi risiko ke server. */
export type RiskSimRequest = {
  account_id?: string
  symbol: string
  entry_price: number
  stop_loss: number
  equity?: number
  mode?: RiskMode
  risk_percent?: number
  lot_size?: number
}

/** Satu baris tabel padanan persen (1%–5%). */
export type RiskTabelPersen = {
  risk_percent: number
  lots: number
  risk_amount: number
  warning: boolean
}

/** Satu baris tabel padanan lot. */
export type RiskTabelLot = {
  lots: number
  risk_amount: number
  risk_percent: number
  warning: boolean
}

/** Hasil simulasi risiko dari server. */
export type RiskSimResult = {
  /** Asal nilai per lot: "broker" | "katalog" | "kategori". */
  sumber_nilai: 'broker' | 'katalog' | 'kategori'
  value_per_lot: number
  contract_size: number
  min_lot: number
  max_lot: number
  lot_step: number
  equity: number
  stop_distance: number
  mode: RiskMode
  hasil: {
    lots: number
    risk_amount: number
    risk_percent: number
    stop_distance: number
    warning: boolean
  }
  tabel: RiskTabelPersen[]
  tabel_lot: RiskTabelLot[]
  peringatan: string[]
}

export type BrokerAccount = {
  id: string
  broker_type: BrokerType
  label: string
  external_account_id: string
  mode: AccountMode
  risk_mode: RiskMode
  risk_percent: number
  lot_size: number
  is_active: boolean
  is_frozen: boolean
  is_dry_run: boolean
  has_credential: boolean
  last_verified_at: string | null
  last_error: string
  connected: boolean
  balance: number
  equity: number
  free_margin: number
  last_synced_at: string | null
}

export type CreateAccountPayload = {
  broker_type: BrokerType
  label: string
  mode: AccountMode
  risk_percent: number
  is_dry_run: boolean
  token: string
  base_url: string
}

export type UpdateAccountPayload = {
  label?: string
  mode?: AccountMode
  risk_mode?: RiskMode
  risk_percent?: number
  lot_size?: number
  is_active?: boolean
  is_dry_run?: boolean
  /** Diisi hanya saat token broker diganti; kosong berarti token lama dipakai. */
  token?: string
  base_url?: string
}

export type StrategyVersion = 'ila-2.0' | 'ila-3.0'

/** Satu baris matriks saklar metode: satu akun dan status saklar tiap metode. */
export type StrategySwitchAccount = {
  account_id: string
  label: string
  broker_type: BrokerType
  is_active: boolean
  is_dry_run: boolean
  /** Nama metode -> saklar menyala. */
  strategies: Record<string, boolean>
  /** Nama metode -> apakah tier pengguna berhak. Metode tanpa hak disembunyikan. */
  allowed: Record<string, boolean>
}

/** Matriks saklar metode untuk satu pair. */
export type StrategySwitchMatrix = {
  symbol: string
  accounts: StrategySwitchAccount[]
}

/** Satu pasangan timeframe yang dikenal dari data sinyal nyata. */
export type TimeframePair = {
  htf: string
  ltf: string
  label: string
}

/** Saklar satu akun untuk satu pasangan timeframe. */
export type TimeframeSwitchItem = {
  htf: string
  ltf: string
  /** Dipantau: sinyalnya diminta ke engine dan ditampilkan. */
  watch: boolean
  /** Diizinkan dieksekusi otomatis. Selalu false bila tidak dipantau. */
  auto: boolean
}

export type TimeframeSwitchAccount = {
  account_id: string
  label: string
  pairs: TimeframeSwitchItem[]
}

export type StrategyTimeframeGroup = {
  strategy: StrategyVersion
  accounts: TimeframeSwitchAccount[]
}

/**
 * Matriks saklar timeframe untuk satu pair.
 *
 * `tersedia` kosong berarti engine belum pernah mengirim pasangan timeframe —
 * bukan berarti tidak ada pilihan.
 */
export type TimeframeSwitchMatrix = {
  symbol: string
  tersedia: Record<string, TimeframePair[]>
  auto_allowed: boolean
  groups: StrategyTimeframeGroup[]
}

export type PairItem = {
  symbol: string
  name: string
  description?: string
  aliases?: string
  /** Simbol pembanding SMT, hanya informasi. */
  correlated_symbols?: string[]
  /** Metadata kontrak untuk kalkulator risiko; 0 berarti belum diisi. */
  value_per_lot?: number
  contract_size?: number
  min_lot?: number
  max_lot?: number
  lot_step?: number
  /** Sumber candle pilihan admin, mis. "FOREXCOM" atau "ICMARKETS". */
  feed_source?: string
}

export type PairGroupView = {
  key: string
  label: string
  pairs: PairItem[]
}

export type PairCategoryView = {
  key: string
  label: string
  groups: PairGroupView[]
}

export type PairCatalogResponse = {
  categories: PairCategoryView[]
  symbols: string[]
}

export type AccountPairSelectionResponse = {
  account_id: string
  risk_percent: number
  selected: string[]
  catalog: PairCatalogResponse
}

export type Subscription = {
  id: string
  symbol: string
  broker_account_id: string | null
  is_active: boolean
}

/**
 * Pair pantauan bersifat global.
 *
 * Metode analisis tidak lagi dipilih saat memantau pair: satu pair dipantau
 * oleh kedua metode, dan metode mana yang dieksekusi ditentukan per akun
 * trading lewat matriks saklar (`GET /pairs/:symbol/strategies`).
 * `broker_account_id` juga opsional dan tidak dipakai halaman Pair Pantauan.
 */
export type CreateSubscriptionPayload = {
  symbol: string
  broker_account_id?: string
}

export type StrategyStatus = 'active' | 'beta' | 'coming_soon'

export type Strategy = {
  version: StrategyVersion
  name: string
  status: StrategyStatus
  available: boolean
  allowed: boolean
  selected: boolean
  description: string
}

export type SignalStatus =
  | 'READY_TO_ENTRY'
  | 'WAIT_HTF_SWEEP'
  | 'WAIT_HTF_CANDLE_CLOSE'
  | 'WAIT_LTF_CISD'
  /** Status kemajuan ILA 3.0 dari engine (tahap HTF → MTF → LTF). */
  | 'WAIT_ILA3_HTF_KEY_LEVEL'
  | 'WAIT_ILA3_MTF_REVERSAL'
  | 'WAIT_ILA3_LTF_CISD_1'
  | 'ILA3_EXPANSION_1_ACTIVE'
  | 'WAIT_ILA3_MTF_FVG_RETR'
  | 'WAIT_ILA3_LTF_CISD_2'
  | 'ILA3_EXPANSION_2_ACTIVE'
  | 'ILA3_TARGET_REACHED'
  | 'INVALIDATED'
  | 'ACKED'
  | 'REJECTED'
  /**
   * Peluang: setup tidak dieksekusi otomatis, tetapi tetap ditampilkan agar
   * user bisa memutuskan sendiri. Bukan penolakan.
   */
  | 'OPPORTUNITY'
  | 'EXPIRED'
  | 'CANCELLED'
  | 'SUPERSEDED'
  | 'CONSUMED'
  | 'SKIPPED'

export type TradeDirection = 'BUY' | 'SELL' | 'NEUTRAL' | string

export type SignalHtfSweep = {
  detected: boolean
  sweep_in_progress: boolean
  direction: string
  stop_loss: number
  target_dol: number
  details: string
  sweep_time?: string
  /** Area FVG yang terbentuk pada sweep; digambar sebagai kotak di chart. */
  fvg?: SignalFairValueGap
  /** Lilin yang menyapu likuiditas. */
  sweep_candle?: SignalCandle
  /** Lilin yang ekornya tersapu; high/low-nya menjadi level sweep. */
  swept_candle?: SignalCandle
}

/** Satu lilin pendukung rincian sweep/CISD. */
export type SignalCandle = {
  timestamp: string
  open: number
  high: number
  low: number
  close: number
  is_closed: boolean
}

/** Fair Value Gap: area antara harga atas dan bawah. */
export type SignalFairValueGap = {
  type?: string
  top: number
  bottom: number
  candle3_time?: string
  mitigated: boolean
}

export type SignalCISD = {
  detected: boolean
  direction: string
  cisd_level: number
  extreme_sl: number
  entry_price: number
  details: string
  break_time?: string
  ref_candle?: SignalCandle
  break_candle?: SignalCandle
}

export type Signal = {
  signal_id: string
  symbol: string
  strategy_version: StrategyVersion
  engine_version: string
  /**
   * Pasangan timeframe sumber sinyal, mis. H4/M15.
   *
   * String kosong berarti engine belum mengirimkannya — jangan ditampilkan.
   */
  htf: string
  ltf: string
  direction: TradeDirection
  entry_price: number
  stop_loss: number
  take_profit: number
  risk_reward: number
  status: SignalStatus
  summary: string
  next_action: string
  reject_reason: string
  htf_sweep?: SignalHtfSweep
  cisd?: SignalCISD
  /**
   * Sumber candle yang dipakai engine saat sinyal ini dihitung, mis.
   * "FOREXCOM" atau "ICMARKETS". Chart dashboard memakai nilai ini sebagai
   * acuan agar garis setup jatuh pada lilin yang sama.
   */
  feed_source?: string
  published_at: string
  /**
   * Kapan baris sinyal terakhir diperbarui (ringkasan/masa berlaku disegarkan
   * engine). Dipakai label "Update" agar tidak tampak macet pada waktu terbit.
   */
  updated_at?: string
  expires_at: string
  can_confirm: boolean
  delayed: boolean
}

export type PositionStatus = 'OPEN' | 'CLOSED_TP' | 'CLOSED_SL' | 'CLOSED_MANUAL' | 'UNTRACKED'

/** Satu lilin pada chart. `time` adalah Unix detik UTC. */
export type MarketCandle = {
  time: number
  open: number
  high: number
  low: number
  close: number
  volume: number
}

/** Balasan endpoint candle: feed efektif ikut dikirim agar bisa dilabeli. */
export type MarketCandlesResponse = {
  symbol: string
  feed: string
  timeframe: string
  candles: MarketCandle[]
}

export type Position = {
  id: string
  symbol: string
  strategy_version: StrategyVersion
  direction: TradeDirection
  ticket_number: string
  lot_size: number
  open_price: number
  stop_loss: number
  take_profit: number
  exit_price: number
  profit_loss: number
  outcome_r: number
  risk_percent: number
  status: PositionStatus
  is_dry_run: boolean
  opened_at: string
  opened_at_pair: TimePair
  closed_at: string | null
  closed_at_pair: TimePair | null
  /**
   * false berarti hasil belum terbaca dari broker. Jangan tampilkan 0 —
   * gunakan garis "(—)" agar tidak menyesatkan (plans/19 C9).
   */
  pnl_known: boolean
}

/**
 * JournalStats merangkum jurnal eksekusi.
 *
 * Angka R dan winrate hanya memakai posisi yang sudah ditutup, sehingga tidak
 * berubah setiap halaman dimuat oleh posisi baru yang masih berjalan.
 */
export type JournalStats = {
  total_positions: number
  open_positions: number
  closed_tp: number
  closed_sl: number
  closed_manual: number
  untracked: number
  wins: number
  losses: number
  /** Persen 0-100; nol bila belum ada penutupan yang hasilnya diketahui. */
  win_rate: number
  total_r: number
  average_r: number
  total_profit: number
  dry_run_positions: number
}

/**
 * JournalSkipped adalah sinyal yang tidak berakhir sebagai posisi.
 *
 * Ditampilkan agar jurnal tidak hanya berisi trade yang jadi: peluang yang
 * terlewat juga bagian dari rekam jejak.
 */
export type JournalSkipped = {
  signal_id: string
  symbol: string
  strategy_version: StrategyVersion
  direction: TradeDirection
  htf: string
  ltf: string
  entry_price: number
  stop_loss: number
  take_profit: number
  risk_reward: number
  /** Status user_signal, bukan status sinyal. */
  status: string
  reason?: string
  occurred_at: string
  occurred_at_pair: TimePair
}

/** Balasan satu permintaan jurnal. */
export type JournalResponse = {
  stats: JournalStats
  positions: Position[]
  skipped: JournalSkipped[]
  /** Halaman daftar peluang terlewat; daftar bisa lebih panjang dari satu layar. */
  skipped_meta: JournalMeta
  filters: {
    from?: string
    to?: string
    strategy_version?: string
    symbol?: string
  }
}

/** JournalMeta menjelaskan halaman daftar peluang terlewat. */
export type JournalMeta = {
  page: number
  per_page: number
  total: number
  total_pages: number
  has_more: boolean
}

/** Satu perangkat yang sedang masuk ke akun ini. */
export type Session = {
  id: string
  device_label: string
  user_agent: string
  last_seen_at: string
  last_seen_pair: TimePair
  created_at: string
  created_pair: TimePair
  expires_at: string
  expires_pair: TimePair
  /** true untuk perangkat yang sedang dipakai. */
  current: boolean
}

export type MarketState = 'OPEN' | 'CLOSED' | 'ROLLOVER' | 'UNKNOWN'

export type SystemStatus = {
  market: { state: MarketState; label: string; updated_at: string }
  engines: {
    strategy: StrategyVersion
    connected: boolean
    engine_version: string
    last_seen_at: string
  }[]
  brokers: { total: number; connected: number; any_error: boolean }
  daily: {
    day: string
    win_count: number
    loss_count: number
    total_trades: number
    max_trades: number
    open_count: number
    locked: boolean
    lock_reason: string
  }
  server_time: string
  daily_timezone: string
}

export type Trial = {
  used: boolean
  active: boolean
  duration_days: number
  days_remaining: number
  dry_run_only: boolean
  trial_started_at: string | null
  trial_expires_at: string | null
}

export type TelegramLink = {
  url: string
  expires_at: string
}

/** Token konfirmasi untuk tombol eksekusi di dashboard. */
export type ApprovalIssue = {
  token: string
  expires_at: string
  can_confirm: boolean
  blocked_reason?: string
}

export type Approval = {
  signal_id: string
  symbol: string
  direction: TradeDirection
  strategy_version: StrategyVersion
  entry_price: number
  stop_loss: number
  take_profit: number
  risk_reward: number
  expires_at: string
  broker_account_id: string
  can_confirm: boolean
  blocked_reason: string
}

export type ApprovalResult = {
  signal_id: string
  status: SignalStatus
  message: string
}

/** Posisi terakhir alur mulai; dipakai agar bisa dilanjutkan. */
export type OnboardingState = {
  step: number
  done: boolean
  total_steps: number
  steps: string[]
  step_done?: Record<string, boolean>
  completed_count?: number
  has_broker: boolean
  email_verified: boolean
  telegram_linked: boolean
  strategy_selected?: boolean
  risk_configured?: boolean
  next_action: string
}

/** Keadaan langganan untuk banner dashboard. */
export type SubscriptionBanner = {
  state: 'normal' | 'ending' | 'grace' | 'expired'
  title: string
  message: string
  expires_at?: string
  days_left: number
  level: 'info' | 'warning' | 'critical'
}
