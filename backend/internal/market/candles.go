// Package market menyediakan data candle untuk dashboard.
//
// Sumbernya sengaja sama dengan yang dibaca engine: TradingView dengan prefiks
// broker pilihan admin (mis. FOREXCOM atau ICMARKETS). Dengan begitu garis
// sweep, CISD, entry, SL, dan TP yang digambar di chart jatuh pada lilin yang
// persis sama dengan yang dievaluasi engine.
package market

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Candle adalah satu lilin yang dikirim ke browser.
//
// Time memakai Unix detik UTC karena itu format yang diminta lightweight-charts.
type Candle struct {
	Time   int64   `json:"time"`
	Open   float64 `json:"open"`
	High   float64 `json:"high"`
	Low    float64 `json:"low"`
	Close  float64 `json:"close"`
	Volume float64 `json:"volume"`
}

// Provider mengambil candle dari sumber pasar.
type Provider interface {
	FetchCandles(ctx context.Context, symbol, feed, timeframe string, limit int) ([]Candle, error)
}

var tvMsgRegex = regexp.MustCompile(`~m~\d+~m~`)

// TradingViewProvider membaca candle dari websocket TradingView.
type TradingViewProvider struct {
	wsURL   string
	headers http.Header
}

// NewTradingViewProvider menyiapkan klien feed TradingView.
func NewTradingViewProvider() *TradingViewProvider {
	headers := http.Header{}
	headers.Set("Origin", "https://www.tradingview.com")
	headers.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	return &TradingViewProvider{
		wsURL:   "wss://data.tradingview.com/socket.io/websocket?from=chart%2F",
		headers: headers,
	}
}

// DefaultFeed adalah feed yang dipakai bila admin belum menentukan apa pun.
func DefaultFeed(symbol string) string {
	if isCryptoSymbol(strings.ToUpper(strings.TrimSpace(symbol))) {
		return "BINANCE"
	}
	return "FOREXCOM"
}

// feedAliases menerjemahkan simbol katalog ILA ke nama simbol yang benar-benar
// dipakai feed tersebut.
//
// Pemetaan ini WAJIB sama dengan yang dipakai engine (pkg/marketfeed di
// go-trade). Contoh: katalog memakai USTEC dan US500, sedangkan FOREX.com di
// TradingView menamainya NAS100 dan SPX500 — sementara IC Markets justru
// memakai USTEC dan US500, tapi menyebut GER40 sebagai DE40 dan USOIL sebagai
// XTIUSD. Tanpa alias, TradingView tidak pernah mengirim bar sehingga chart
// tidak pernah tampil.
//
// Semua pasangan di bawah sudah diuji langsung ke feed TradingView, bukan
// tebakan.
var feedAliases = map[string]map[string]string{
	"FOREXCOM": {
		"USTEC": "NAS100",
		"US100": "NAS100",
		"NDX":   "NAS100",
		"US500": "SPX500",
		"SPX":   "SPX500",
	},
	"ICMARKETS": {
		"GER40": "DE40",
		"USOIL": "XTIUSD",
		"WTI":   "XTIUSD",
	},
}

// MapSymbol memetakan simbol kanonik ke "FEED:SIMBOL".
func MapSymbol(symbol, feed string) string {
	sym := strings.ToUpper(strings.TrimSpace(symbol))
	if strings.Contains(sym, ":") {
		return sym
	}
	sym = strings.ReplaceAll(sym, "/", "")

	f := strings.ToUpper(strings.TrimSpace(feed))
	if f == "" {
		f = DefaultFeed(sym)
	}

	// Binance memakai pasangan USDT, sedangkan katalog ILA memakai akhiran USD.
	if f == "BINANCE" && strings.HasSuffix(sym, "USD") && !strings.HasSuffix(sym, "USDT") {
		sym += "T"
	}

	if alias, ok := feedAliases[f][sym]; ok {
		sym = alias
	}

	return f + ":" + sym
}

// MapResolution menerjemahkan timeframe ILA ke resolusi TradingView.
// String kosong berarti timeframe tidak dikenal.
func MapResolution(timeframe string) string {
	switch strings.ToUpper(strings.TrimSpace(timeframe)) {
	case "1M", "M1", "1":
		return "1"
	case "3M", "M3", "3":
		return "3"
	case "5M", "M5", "5":
		return "5"
	case "15M", "M15", "15":
		return "15"
	case "30M", "M30", "30":
		return "30"
	case "1H", "H1", "60M", "60":
		return "60"
	case "2H", "H2", "120M", "120":
		return "120"
	case "4H", "H4", "240M", "240":
		return "240"
	case "1D", "D1", "D":
		return "1D"
	case "1W", "W1", "W":
		return "1W"
	default:
		return ""
	}
}

func randomSessionID(prefix string) string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	return prefix + hex.EncodeToString(b)
}

func formatMsg(funcName string, params []any) string {
	payload, _ := json.Marshal(map[string]any{"m": funcName, "p": params})
	return fmt.Sprintf("~m~%d~m~%s", len(payload), payload)
}

// tvErrorText menyusun keterangan galat dari payload protokol TradingView.
func tvErrorText(raw map[string]any) string {
	params, _ := raw["p"].([]any)
	parts := make([]string, 0, len(params))
	for _, param := range params {
		if text, ok := param.(string); ok && text != "" {
			parts = append(parts, text)
		}
	}
	if len(parts) == 0 {
		return "tanpa keterangan"
	}
	return strings.Join(parts, " ")
}

// FetchCandles mengambil candle terbaru dari TradingView.
func (p *TradingViewProvider) FetchCandles(ctx context.Context, symbol, feed, timeframe string, limit int) ([]Candle, error) {
	resolution := MapResolution(timeframe)
	if resolution == "" {
		return nil, fmt.Errorf("timeframe tidak dikenal: %q", timeframe)
	}
	if limit <= 0 {
		limit = 200
	}

	tvSymbol := MapSymbol(symbol, feed)

	dialer := websocket.DefaultDialer
	conn, _, err := dialer.DialContext(ctx, p.wsURL, p.headers)
	if err != nil {
		return nil, fmt.Errorf("gagal menyambung ke feed TradingView: %w", err)
	}
	defer conn.Close()

	chartSession := randomSessionID("cs_")

	_ = conn.WriteMessage(websocket.TextMessage, []byte(formatMsg("set_auth_token", []any{"unauthorized_user_token"})))
	_ = conn.WriteMessage(websocket.TextMessage, []byte(formatMsg("chart_create_session", []any{chartSession, ""})))

	symSpec, _ := json.Marshal(map[string]string{"symbol": tvSymbol, "adjustment": "splits"})
	_ = conn.WriteMessage(websocket.TextMessage, []byte(formatMsg("resolve_symbol", []any{chartSession, "sds_sym_1", "=" + string(symSpec)})))
	_ = conn.WriteMessage(websocket.TextMessage, []byte(formatMsg("create_series", []any{chartSession, "sds_1", "s1", "sds_sym_1", resolution, limit, ""})))

	// Batas waktu per baca, bukan batas total: selama feed masih mengirim
	// pesan, koneksi tidak diputus.
	const batasBaca = 12 * time.Second

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		_ = conn.SetReadDeadline(time.Now().Add(batasBaca))

		_, msgBytes, err := conn.ReadMessage()
		if err != nil {
			return nil, fmt.Errorf("membaca feed TradingView (%s): %w", tvSymbol, err)
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

			// Simbol yang tidak ada di feed dilaporkan sebagai pesan protokol,
			// bukan sebagai bar kosong. Tanpa deteksi ini permintaan menggantung
			// sampai batas waktu lalu berakhir sebagai galat 500 tanpa sebab jelas.
			switch method {
			case "symbol_error":
				return nil, fmt.Errorf("simbol %s tidak tersedia pada feed (%s)", tvSymbol, tvErrorText(raw))
			case "series_error", "critical_error", "protocol_error":
				return nil, fmt.Errorf("feed menolak permintaan %s (%s)", tvSymbol, tvErrorText(raw))
			}

			if method != "timescale_update" && method != "du" {
				continue
			}

			candles, ok := parseBars(raw)
			if !ok || len(candles) == 0 {
				continue
			}
			if len(candles) > limit {
				candles = candles[len(candles)-limit:]
			}
			return candles, nil
		}
	}
}

// parseBars mengubah payload timescale_update menjadi candle terurut menaik.
//
// Lightweight-charts menolak data yang tidak menaik atau punya waktu ganda,
// jadi pengurutan dan deduplikasi dilakukan di sini, bukan di browser.
func parseBars(raw map[string]any) ([]Candle, bool) {
	params, ok := raw["p"].([]any)
	if !ok || len(params) < 2 {
		return nil, false
	}
	dataObj, ok := params[1].(map[string]any)
	if !ok {
		return nil, false
	}
	sdsObj, ok := dataObj["sds_1"].(map[string]any)
	if !ok {
		return nil, false
	}
	barsList, ok := sdsObj["s"].([]any)
	if !ok || len(barsList) == 0 {
		return nil, false
	}

	byTime := make(map[int64]Candle, len(barsList))
	for _, item := range barsList {
		barMap, ok := item.(map[string]any)
		if !ok {
			continue
		}
		values, ok := barMap["v"].([]any)
		if !ok || len(values) < 5 {
			continue
		}

		tsSec, _ := values[0].(float64)
		open, _ := values[1].(float64)
		high, _ := values[2].(float64)
		low, _ := values[3].(float64)
		closePrice, _ := values[4].(float64)
		volume := 0.0
		if len(values) >= 6 {
			volume, _ = values[5].(float64)
		}
		if tsSec <= 0 {
			continue
		}

		// Lilin yang belum ditutup tetap dikirim: chart perlu lilin berjalan
		// agar pergerakan harga terlihat hidup.
		byTime[int64(tsSec)] = Candle{
			Time:   int64(tsSec),
			Open:   open,
			High:   high,
			Low:    low,
			Close:  closePrice,
			Volume: volume,
		}
	}
	if len(byTime) == 0 {
		return nil, false
	}

	out := make([]Candle, 0, len(byTime))
	for _, candle := range byTime {
		out = append(out, candle)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Time < out[j].Time })
	return out, true
}

func isCryptoSymbol(sym string) bool {
	switch sym {
	case "BTCUSD", "BTCUSDT", "ETHUSD", "ETHUSDT",
		"BNBUSD", "BNBUSDT", "SOLUSD", "SOLUSDT",
		"XRPUSD", "XRPUSDT", "ADAUSD", "ADAUSDT",
		"DOGEUSD", "DOGEUSDT", "PAXGUSD", "PAXGUSDT":
		return true
	default:
		return false
	}
}

// Service menyajikan candle dengan cache pendek.
//
// Cache mencegah satu halaman yang terus diperbarui menembak TradingView untuk
// setiap pengguna, sementara umurnya cukup singkat agar lilin berjalan tetap
// terasa hidup.
type Service struct {
	provider Provider
	ttl      time.Duration

	mu     sync.Mutex
	cache  map[string]cacheEntry
	locks  map[string]*sync.Mutex
	clock  func() time.Time
	maxBar int
}

type cacheEntry struct {
	candles []Candle
	expires time.Time
}

// NewService merangkai layanan candle dengan umur cache tertentu.
func NewService(provider Provider, ttl time.Duration) *Service {
	if ttl <= 0 {
		ttl = 10 * time.Second
	}
	return &Service{
		provider: provider,
		ttl:      ttl,
		cache:    make(map[string]cacheEntry),
		locks:    make(map[string]*sync.Mutex),
		clock:    func() time.Time { return time.Now() },
		maxBar:   500,
	}
}

// Candles mengembalikan candle untuk satu simbol dan timeframe beserta feed
// efektif yang dipakai, sehingga klien bisa menampilkan label broker.
func (s *Service) Candles(ctx context.Context, symbol, feed, timeframe string, limit int) ([]Candle, string, error) {
	if strings.TrimSpace(symbol) == "" {
		return nil, "", fmt.Errorf("simbol wajib diisi")
	}
	if MapResolution(timeframe) == "" {
		return nil, "", fmt.Errorf("timeframe tidak didukung: %q", timeframe)
	}
	if limit <= 0 {
		limit = 200
	}
	if limit > s.maxBar {
		limit = s.maxBar
	}

	feed = strings.ToUpper(strings.TrimSpace(feed))
	if feed == "" {
		feed = DefaultFeed(symbol)
	}

	key := strings.ToUpper(strings.TrimSpace(symbol)) + "|" + feed + "|" +
		strings.ToUpper(strings.TrimSpace(timeframe)) + "|" + fmt.Sprint(limit)

	lock := s.lockFor(key)
	lock.Lock()
	defer lock.Unlock()

	s.mu.Lock()
	entry, ok := s.cache[key]
	s.mu.Unlock()
	if ok && s.clock().Before(entry.expires) {
		return entry.candles, feed, nil
	}

	candles, err := s.provider.FetchCandles(ctx, symbol, feed, timeframe, limit)
	if err != nil {
		// Data lama masih lebih berguna daripada layar kosong saat feed sesaat
		// gagal; umurnya diperpendek agar cepat dicoba ulang.
		if ok && len(entry.candles) > 0 {
			s.mu.Lock()
			s.cache[key] = cacheEntry{candles: entry.candles, expires: s.clock().Add(s.ttl / 2)}
			s.mu.Unlock()
			return entry.candles, feed, nil
		}
		return nil, feed, err
	}

	s.mu.Lock()
	s.cache[key] = cacheEntry{candles: candles, expires: s.clock().Add(s.ttl)}
	s.mu.Unlock()

	return candles, feed, nil
}

func (s *Service) lockFor(key string) *sync.Mutex {
	s.mu.Lock()
	defer s.mu.Unlock()
	lock, ok := s.locks[key]
	if !ok {
		lock = &sync.Mutex{}
		s.locks[key] = lock
	}
	return lock
}
