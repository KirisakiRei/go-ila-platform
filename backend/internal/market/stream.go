package market

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var (
	// ErrStreamCapacity dikembalikan saat batas stream serentak tercapai agar
	// server kecil tetap terlindungi dari lonjakan memori.
	ErrStreamCapacity = errors.New("kapasitas live stream penuh")
	// ErrStreamClosed dikembalikan saat hub sudah dimatikan.
	ErrStreamClosed = errors.New("live stream ditutup")
)

// StreamProvider mengalirkan pembaruan lilin berjalan dari sumber pasar.
type StreamProvider interface {
	StreamCandles(ctx context.Context, symbol, feed, timeframe string, onBars func([]Candle)) error
}

// StreamCandles membuka koneksi WebSocket ke TradingView dan meneruskan setiap
// pembaruan bar ("timescale_update" maupun "du") hingga ctx dibatalkan.
func (p *TradingViewProvider) StreamCandles(ctx context.Context, symbol, feed, timeframe string, onBars func([]Candle)) error {
	resolution := MapResolution(timeframe)
	if resolution == "" {
		return fmt.Errorf("timeframe tidak dikenal: %q", timeframe)
	}
	tvSymbol := MapSymbol(symbol, feed)

	dialer := websocket.DefaultDialer
	conn, _, err := dialer.DialContext(ctx, p.wsURL, p.headers)
	if err != nil {
		return fmt.Errorf("gagal menyambung stream TradingView: %w", err)
	}

	var closeOnce sync.Once
	closeConn := func() {
		closeOnce.Do(func() {
			_ = conn.Close()
		})
	}
	defer closeConn()

	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-ctx.Done():
			closeConn()
		case <-done:
		}
	}()

	chartSession := randomSessionID("cs_")
	_ = conn.WriteMessage(websocket.TextMessage, []byte(formatMsg("set_auth_token", []any{"unauthorized_user_token"})))
	_ = conn.WriteMessage(websocket.TextMessage, []byte(formatMsg("chart_create_session", []any{chartSession, ""})))

	symSpec, _ := json.Marshal(map[string]string{"symbol": tvSymbol, "adjustment": "splits"})
	_ = conn.WriteMessage(websocket.TextMessage, []byte(formatMsg("resolve_symbol", []any{chartSession, "sds_sym_1", "=" + string(symSpec)})))
	// Hanya 5 lilin ekor yang diminta karena riwayat 200 lilin sudah dimuat via REST cache.
	_ = conn.WriteMessage(websocket.TextMessage, []byte(formatMsg("create_series", []any{chartSession, "sds_1", "s1", "sds_sym_1", resolution, 5, ""})))

	const batasBacaStream = 45 * time.Second

	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		_ = conn.SetReadDeadline(time.Now().Add(batasBacaStream))

		_, msgBytes, readErr := conn.ReadMessage()
		if readErr != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return fmt.Errorf("membaca stream TradingView (%s): %w", tvSymbol, readErr)
		}

		for _, part := range tvMsgRegex.Split(string(msgBytes), -1) {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			if strings.HasPrefix(part, "~h~") {
				_ = conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("~m~%d~m~%s", len(part), part)))
				continue
			}

			var raw map[string]any
			if err := json.Unmarshal([]byte(part), &raw); err != nil {
				continue
			}
			method, _ := raw["m"].(string)
			switch method {
			case "symbol_error":
				return fmt.Errorf("simbol %s tidak tersedia pada feed (%s)", tvSymbol, tvErrorText(raw))
			case "series_error", "critical_error", "protocol_error":
				return fmt.Errorf("feed menolak stream %s (%s)", tvSymbol, tvErrorText(raw))
			}

			if method != "timescale_update" && method != "du" {
				continue
			}

			candles, ok := parseBars(raw)
			if !ok || len(candles) == 0 {
				continue
			}
			if len(candles) > 2 {
				candles = candles[len(candles)-2:]
			}
			onBars(candles)
		}
	}
}

// StreamHubOptions mengatur batas pemakaian memori dan jeda hidup stream.
type StreamHubOptions struct {
	MaxStreams      int
	MaxSubscribers  int
	IdleGrace       time.Duration
	MinEmitInterval time.Duration
	Cache           *Service
}

// StreamHub memultipleks satu koneksi upstream per (simbol, feed, timeframe) ke
// banyak penonton SSE, serta memutus koneksi otomatis saat tidak ada penonton.
type StreamHub struct {
	provider StreamProvider
	opts     StreamHubOptions

	rootCtx    context.Context
	rootCancel context.CancelFunc

	mu         sync.Mutex
	closed     bool
	totalSubs  int
	nextSubID  uint64
	channels   map[string]*streamChannel
}

type streamChannel struct {
	key       string
	symbol    string
	feed      string
	timeframe string

	cancel    context.CancelFunc
	subs      map[uint64]chan Candle
	idleTimer *time.Timer
	lastBars  []Candle
}

// NewStreamHub menyiapkan pengelola live stream candle hemat memori.
func NewStreamHub(provider StreamProvider, opts StreamHubOptions) *StreamHub {
	if opts.MaxStreams <= 0 {
		opts.MaxStreams = 16
	}
	if opts.MaxSubscribers <= 0 {
		opts.MaxSubscribers = 128
	}
	if opts.IdleGrace <= 0 {
		opts.IdleGrace = 15 * time.Second
	}
	if opts.MinEmitInterval <= 0 {
		opts.MinEmitInterval = 300 * time.Millisecond
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &StreamHub{
		provider:   provider,
		opts:       opts,
		rootCtx:    ctx,
		rootCancel: cancel,
		channels:   make(map[string]*streamChannel),
	}
}

// Subscribe mendaftarkan penonton baru pada pasangan simbol & timeframe.
func (h *StreamHub) Subscribe(symbol, feed, timeframe string) (<-chan Candle, string, func(), error) {
	sym := strings.ToUpper(strings.TrimSpace(symbol))
	if sym == "" {
		return nil, "", nil, fmt.Errorf("simbol wajib diisi")
	}
	res := MapResolution(timeframe)
	if res == "" {
		return nil, "", nil, fmt.Errorf("timeframe tidak didukung: %q", timeframe)
	}
	usedFeed := strings.ToUpper(strings.TrimSpace(feed))
	if usedFeed == "" {
		usedFeed = DefaultFeed(sym)
	}
	key := MapSymbol(sym, usedFeed) + "|" + res

	h.mu.Lock()
	if h.closed {
		h.mu.Unlock()
		return nil, usedFeed, nil, ErrStreamClosed
	}
	if h.totalSubs >= h.opts.MaxSubscribers {
		h.mu.Unlock()
		return nil, usedFeed, nil, ErrStreamCapacity
	}

	ch, exists := h.channels[key]
	if !exists && len(h.channels) >= h.opts.MaxStreams {
		h.mu.Unlock()
		return nil, usedFeed, nil, ErrStreamCapacity
	}

	if !exists {
		chanCtx, chanCancel := context.WithCancel(h.rootCtx)
		ch = &streamChannel{
			key:       key,
			symbol:    sym,
			feed:      usedFeed,
			timeframe: strings.ToUpper(strings.TrimSpace(timeframe)),
			cancel:    chanCancel,
			subs:      make(map[uint64]chan Candle),
		}
		h.channels[key] = ch
		go h.runUpstream(chanCtx, ch)
	} else if ch.idleTimer != nil {
		ch.idleTimer.Stop()
		ch.idleTimer = nil
	}

	h.nextSubID++
	subID := h.nextSubID
	subChan := make(chan Candle, 8)
	for _, b := range ch.lastBars {
		pushCandleNonBlocking(subChan, b)
	}
	ch.subs[subID] = subChan
	h.totalSubs++
	h.mu.Unlock()

	var unsubOnce sync.Once
	unsubscribe := func() {
		unsubOnce.Do(func() {
			h.unsubscribe(key, subID)
		})
	}
	return subChan, usedFeed, unsubscribe, nil
}

func (h *StreamHub) unsubscribe(key string, subID uint64) {
	h.mu.Lock()
	defer h.mu.Unlock()

	ch, ok := h.channels[key]
	if !ok {
		return
	}
	subChan, subOK := ch.subs[subID]
	if !subOK {
		return
	}
	delete(ch.subs, subID)
	close(subChan)
	if h.totalSubs > 0 {
		h.totalSubs--
	}

	if len(ch.subs) == 0 && !h.closed {
		if ch.idleTimer != nil {
			ch.idleTimer.Stop()
		}
		grace := h.opts.IdleGrace
		ch.idleTimer = time.AfterFunc(grace, func() {
			h.reapIdleChannel(key)
		})
	}
}

func (h *StreamHub) reapIdleChannel(key string) {
	h.mu.Lock()
	ch, ok := h.channels[key]
	if !ok || len(ch.subs) > 0 {
		h.mu.Unlock()
		return
	}
	delete(h.channels, key)
	cancel := ch.cancel
	h.mu.Unlock()

	cancel()
}

// ActiveStreamCount mengembalikan jumlah koneksi upstream yang sedang hidup.
func (h *StreamHub) ActiveStreamCount() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.channels)
}

// Close menghentikan seluruh stream upstream dan menutup kanal pelanggan.
func (h *StreamHub) Close() {
	h.mu.Lock()
	if h.closed {
		h.mu.Unlock()
		return
	}
	h.closed = true
	for _, ch := range h.channels {
		if ch.idleTimer != nil {
			ch.idleTimer.Stop()
			ch.idleTimer = nil
		}
		ch.cancel()
		for id, subChan := range ch.subs {
			delete(ch.subs, id)
			close(subChan)
		}
	}
	h.channels = make(map[string]*streamChannel)
	h.totalSubs = 0
	h.mu.Unlock()

	h.rootCancel()
}

func (h *StreamHub) runUpstream(ctx context.Context, ch *streamChannel) {
	incoming := make(chan []Candle, 16)
	done := make(chan struct{})

	go func() {
		defer close(done)
		backoffs := []time.Duration{time.Second, 2 * time.Second, 5 * time.Second}
		attempt := 0
		for {
			if ctx.Err() != nil {
				return
			}
			gotData := false
			err := h.provider.StreamCandles(ctx, ch.symbol, ch.feed, ch.timeframe, func(bars []Candle) {
				gotData = true
				cp := append([]Candle(nil), bars...)
				select {
				case incoming <- cp:
				default:
					// Ganti antrean lama agar konsumen selalu menerima harga terbaru.
					select {
					case <-incoming:
					default:
					}
					select {
					case incoming <- cp:
					default:
					}
				}
			})
			if ctx.Err() != nil {
				return
			}
			if gotData {
				attempt = 0
			} else {
				attempt++
			}
			if err != nil && attempt > len(backoffs) {
				h.forceCloseChannel(ch.key)
				return
			}
			wait := backoffs[len(backoffs)-1]
			if attempt > 0 && attempt-1 < len(backoffs) {
				wait = backoffs[attempt-1]
			}
			timer := time.NewTimer(wait)
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case <-timer.C:
			}
		}
	}()

	var pending []Candle
	var flushTimer *time.Timer
	var flushC <-chan time.Time
	lastEmit := time.Time{}

	flush := func() {
		if len(pending) == 0 {
			return
		}
		toSend := pending
		pending = nil
		lastEmit = time.Now()
		h.broadcast(ch.key, toSend)
	}

	for {
		select {
		case <-ctx.Done():
			if flushTimer != nil {
				flushTimer.Stop()
			}
			return
		case <-done:
			flush()
			return
		case bars := <-incoming:
			pending = mergeStreamBars(pending, bars)
			elapsed := time.Since(lastEmit)
			if elapsed >= h.opts.MinEmitInterval {
				if flushTimer != nil {
					flushTimer.Stop()
					flushC = nil
				}
				flush()
			} else if flushC == nil {
				wait := h.opts.MinEmitInterval - elapsed
				if flushTimer == nil {
					flushTimer = time.NewTimer(wait)
				} else {
					flushTimer.Reset(wait)
				}
				flushC = flushTimer.C
			}
		case <-flushC:
			flushC = nil
			flush()
		}
	}
}

func mergeStreamBars(existing, incoming []Candle) []Candle {
	if len(existing) == 0 {
		cp := append([]Candle(nil), incoming...)
		if len(cp) > 2 {
			cp = cp[len(cp)-2:]
		}
		return cp
	}
	byTime := make(map[int64]Candle, len(existing)+len(incoming))
	for _, b := range existing {
		byTime[b.Time] = b
	}
	for _, b := range incoming {
		byTime[b.Time] = b
	}
	merged := make([]Candle, 0, len(byTime))
	for _, b := range byTime {
		merged = append(merged, b)
	}
	for i := 0; i < len(merged); i++ {
		for j := i + 1; j < len(merged); j++ {
			if merged[j].Time < merged[i].Time {
				merged[i], merged[j] = merged[j], merged[i]
			}
		}
	}
	if len(merged) > 2 {
		merged = merged[len(merged)-2:]
	}
	return merged
}

func (h *StreamHub) broadcast(key string, bars []Candle) {
	if h.opts.Cache != nil {
		// Segarkan ekor cache REST bila seri yang sama sudah dimuat.
		h.opts.Cache.UpdateLatestBars(bars, key)
	}

	h.mu.Lock()
	ch, ok := h.channels[key]
	if !ok {
		h.mu.Unlock()
		return
	}
	ch.lastBars = append([]Candle(nil), bars...)
	subs := make([]chan Candle, 0, len(ch.subs))
	for _, subChan := range ch.subs {
		subs = append(subs, subChan)
	}
	h.mu.Unlock()

	for _, subChan := range subs {
		for _, bar := range bars {
			pushCandleNonBlocking(subChan, bar)
		}
	}
}

func (h *StreamHub) forceCloseChannel(key string) {
	h.mu.Lock()
	ch, ok := h.channels[key]
	if !ok {
		h.mu.Unlock()
		return
	}
	delete(h.channels, key)
	if ch.idleTimer != nil {
		ch.idleTimer.Stop()
		ch.idleTimer = nil
	}
	for id, subChan := range ch.subs {
		delete(ch.subs, id)
		close(subChan)
		if h.totalSubs > 0 {
			h.totalSubs--
		}
	}
	cancel := ch.cancel
	h.mu.Unlock()

	cancel()
}

func pushCandleNonBlocking(ch chan Candle, bar Candle) {
	select {
	case ch <- bar:
	default:
		select {
		case <-ch:
		default:
		}
		select {
		case ch <- bar:
		default:
		}
	}
}
