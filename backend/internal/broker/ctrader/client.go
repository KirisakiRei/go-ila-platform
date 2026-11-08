package ctrader

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/diegobernardes/ctrader"
	"github.com/diegobernardes/ctrader/openapi"
	"golang.org/x/exp/slog"
	"google.golang.org/protobuf/proto"

	"ila/backend/internal/broker"
)

var (
	errMissingAccessToken = errors.New("access_token cTrader kosong")
	errMissingAccountID   = errors.New("ctidTraderAccountId cTrader kosong")
)

type Client struct {
	accountID           string
	defaultClientID     string
	defaultClientSecret string

	mu        sync.Mutex
	client    *ctrader.Client
	authed    bool
	creds     broker.Credentials
	ctidAccID int64
	live      bool

	symbolMap map[string]int64
	idToSym   map[int64]string
	lotMult   map[string]int64
	digits    map[string]int
}

func New(accountID string, defaultClientID, defaultClientSecret string) *Client {
	if defaultClientID == "" {
		defaultClientID = os.Getenv("CTRADER_CLIENT_ID")
	}
	if defaultClientSecret == "" {
		defaultClientSecret = os.Getenv("CTRADER_CLIENT_SECRET")
	}

	return &Client{
		accountID:           accountID,
		defaultClientID:     defaultClientID,
		defaultClientSecret: defaultClientSecret,
		symbolMap: map[string]int64{
			"EURUSD": 1,
			"XAUUSD": 41,
			"GOLD":   41,
			"USTEC":  10014,
			"NAS100": 10014,
			"US100":  10014,
		},
		idToSym: map[int64]string{
			1:     "EURUSD",
			41:    "XAUUSD",
			10014: "USTEC",
		},
		lotMult: map[string]int64{
			"EURUSD": 10000000,
			"XAUUSD": 10000,
			"GOLD":   10000,
			"USTEC":  100,
			"NAS100": 100,
			"US100":  100,
		},
		digits: map[string]int{
			"EURUSD": 5,
			"XAUUSD": 2,
			"GOLD":   2,
			"USTEC":  2,
			"NAS100": 2,
			"US100":  2,
		},
	}
}

func (c *Client) Platform() string {
	return "ctrader"
}

func (c *Client) AccountID() string {
	return c.accountID
}

func (c *Client) Connected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.client != nil && c.authed
}

func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.authed = false
	if c.client != nil {
		err := c.client.Stop()
		c.client = nil
		return err
	}
	return nil
}

func (c *Client) Connect(ctx context.Context, creds broker.Credentials) error {
	token := strings.TrimSpace(creds.Token)
	if token == "" {
		return fmt.Errorf("%w: %v", broker.ErrInvalidCredentials, errMissingAccessToken)
	}

	ctidStr := strings.TrimSpace(creds.Extra["ctid_trader_account_id"])
	if ctidStr == "" {
		ctidStr = strings.TrimSpace(creds.Login)
	}
	if ctidStr == "" {
		return fmt.Errorf("%w: %v", broker.ErrInvalidCredentials, errMissingAccountID)
	}

	ctidAccID, err := strconv.ParseInt(ctidStr, 10, 64)
	if err != nil {
		return fmt.Errorf("%w: ctidTraderAccountId '%s' tidak sah: %v", broker.ErrInvalidCredentials, ctidStr, err)
	}

	clientID := strings.TrimSpace(creds.Extra["client_id"])
	if clientID == "" {
		clientID = c.defaultClientID
	}
	clientSecret := strings.TrimSpace(creds.Extra["client_secret"])
	if clientSecret == "" {
		clientSecret = c.defaultClientSecret
	}
	if clientID == "" || clientSecret == "" {
		return fmt.Errorf("%w: cTrader ClientID atau ClientSecret belum disiapkan", broker.ErrInvalidCredentials)
	}

	live := creds.Extra["live"] == "true"

	c.mu.Lock()
	if c.client != nil {
		_ = c.client.Stop()
		c.client = nil
		c.authed = false
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	client := &ctrader.Client{
		ApplicationClientID: clientID,
		ApplicationSecret:   clientSecret,
		Logger:              logger,
		Deadline:            10 * time.Second,
		Live:                live,
		HandlerEvent:        func(proto.Message) {},
	}

	if err := client.Start(); err != nil {
		c.mu.Unlock()
		return fmt.Errorf("%w: gagal memulai cTrader client: %v", broker.ErrNotConnected, err)
	}

	// Otentikasi Akun cTrader
	_, err = ctrader.Command[*openapi.ProtoOAAccountAuthReq, *openapi.ProtoOAAccountAuthRes](
		ctx,
		client,
		&openapi.ProtoOAAccountAuthReq{
			CtidTraderAccountId: &ctidAccID,
			AccessToken:         &token,
		},
	)
	if err != nil {
		_ = client.Stop()
		c.mu.Unlock()
		return fmt.Errorf("%w: autentikasi akun cTrader gagal: %v", broker.ErrInvalidCredentials, err)
	}

	c.client = client
	c.authed = true
	c.creds = creds
	c.ctidAccID = ctidAccID
	c.live = live
	c.mu.Unlock()

	return nil
}

func (c *Client) Account(ctx context.Context) (broker.AccountSnapshot, error) {
	c.mu.Lock()
	client := c.client
	authed := c.authed
	ctidAccID := c.ctidAccID
	c.mu.Unlock()

	if client == nil || !authed {
		return broker.AccountSnapshot{}, broker.ErrNotConnected
	}

	res, err := ctrader.Command[*openapi.ProtoOATraderReq, *openapi.ProtoOATraderRes](
		ctx,
		client,
		&openapi.ProtoOATraderReq{
			CtidTraderAccountId: &ctidAccID,
		},
	)
	if err != nil {
		c.mu.Lock()
		c.authed = false
		c.mu.Unlock()
		return broker.AccountSnapshot{}, fmt.Errorf("mengambil info trader: %w", err)
	}

	trader := res.GetTrader()
	moneyDigits := trader.GetMoneyDigits()
	div := math.Pow10(int(moneyDigits))
	balance := float64(trader.GetBalance()) / div

	positions, _ := c.ListPositions(ctx)
	equity := balance
	for _, p := range positions {
		equity += p.Profit
	}

	return broker.AccountSnapshot{
		AccountID:  strconv.FormatInt(trader.GetTraderLogin(), 10),
		Currency:   "USD",
		Balance:    math.Round(balance*100) / 100,
		Equity:     math.Round(equity*100) / 100,
		FreeMargin: math.Round(equity*100) / 100,
		Leverage:   int(trader.GetLeverageInCents() / 100),
	}, nil
}

func (c *Client) Symbol(ctx context.Context, symbol string) (broker.SymbolSpec, error) {
	norm := strings.ToUpper(strings.TrimSpace(symbol))
	if norm == "GOLD" {
		norm = "XAUUSD"
	}
	if norm == "NAS100" || norm == "US100" {
		norm = "USTEC"
	}

	spec, ok := defaultSpecs[norm]
	if !ok {
		return broker.SymbolSpec{}, fmt.Errorf("%w: simbol %s tidak dikenal", broker.ErrUnsupported, symbol)
	}
	return spec, nil
}

var defaultSpecs = map[string]broker.SymbolSpec{
	"EURUSD": {
		Symbol:       "EURUSD",
		Digits:       5,
		PipSize:      0.0001,
		ValuePerLot:  10.0,
		MinLot:       0.01,
		MaxLot:       100.0,
		LotStep:      0.01,
		ContractSize: 100000.0,
	},
	"XAUUSD": {
		Symbol:       "XAUUSD",
		Digits:       2,
		PipSize:      0.01,
		ValuePerLot:  1.0,
		MinLot:       0.01,
		MaxLot:       100.0,
		LotStep:      0.01,
		ContractSize: 100.0,
	},
	"USTEC": {
		Symbol:       "USTEC",
		Digits:       2,
		PipSize:      1.0,
		ValuePerLot:  1.0,
		MinLot:       0.10,
		MaxLot:       50.0,
		LotStep:      0.10,
		ContractSize: 1.0,
	},
}

func (c *Client) CurrentPrice(ctx context.Context, symbol string) (float64, error) {
	positions, err := c.ListPositions(ctx)
	if err == nil {
		for _, p := range positions {
			if strings.EqualFold(p.Symbol, symbol) && p.OpenPrice > 0 {
				return p.OpenPrice, nil
			}
		}
	}
	return 0, nil
}

func (c *Client) OpenPosition(ctx context.Context, req broker.OpenRequest) (broker.OpenResult, error) {
	c.mu.Lock()
	client := c.client
	authed := c.authed
	ctidAccID := c.ctidAccID
	c.mu.Unlock()

	if client == nil || !authed {
		return broker.OpenResult{}, broker.ErrNotConnected
	}

	symNorm := strings.ToUpper(strings.TrimSpace(req.Symbol))
	symID, ok := c.symbolMap[symNorm]
	if !ok {
		return broker.OpenResult{}, fmt.Errorf("%w: simbol %s tidak didukung cTrader", broker.ErrUnsupported, req.Symbol)
	}

	mult := c.lotMult[symNorm]
	if mult == 0 {
		mult = 10000000
	}
	rawVolume := int64(math.Round(req.Lots * float64(mult)))
	if symID == 10014 { // USTEC step volume 10 (0.10 lot)
		rawVolume = (rawVolume / 10) * 10
		if rawVolume < 10 {
			rawVolume = 10
		}
	}

	orderType := openapi.ProtoOAOrderType_MARKET
	tradeSide := openapi.ProtoOATradeSide_BUY
	if strings.ToUpper(req.Side) == "SELL" {
		tradeSide = openapi.ProtoOATradeSide_SELL
	}

	orderReq := &openapi.ProtoOANewOrderReq{
		CtidTraderAccountId: &ctidAccID,
		SymbolId:            &symID,
		OrderType:           &orderType,
		TradeSide:           &tradeSide,
		Volume:              &rawVolume,
		Comment:             &req.Comment,
	}

	digits := c.digits[symNorm]
	if digits == 0 {
		digits = 2
	}
	pow := math.Pow10(digits)
	roundPrice := func(p float64) float64 {
		return math.Round(p*pow) / pow
	}

	if req.StopLoss > 0 {
		sl := roundPrice(req.StopLoss)
		orderReq.StopLoss = &sl
	}
	if req.TakeProfit > 0 {
		tp := roundPrice(req.TakeProfit)
		orderReq.TakeProfit = &tp
	}

	respMsg, err := ctrader.Command[*openapi.ProtoOANewOrderReq, proto.Message](
		ctx,
		client,
		orderReq,
	)
	if err != nil {
		c.mu.Lock()
		c.authed = false
		c.mu.Unlock()
		return broker.OpenResult{}, fmt.Errorf("eksekusi order cTrader gagal: %w", err)
	}

	var execEvent *openapi.ProtoOAExecutionEvent
	switch v := respMsg.(type) {
	case *openapi.ProtoOAExecutionEvent:
		execEvent = v
	case *openapi.ProtoOAOrderErrorEvent:
		return broker.OpenResult{}, fmt.Errorf("cTrader order error [%s]: %s", v.GetErrorCode(), v.GetDescription())
	default:
		return broker.OpenResult{}, fmt.Errorf("cTrader respons tidak terduga: %T", respMsg)
	}

	posID := int64(0)
	execPrice := 0.0

	if execEvent.GetPosition() != nil {
		posID = execEvent.GetPosition().GetPositionId()
		execPrice = execEvent.GetPosition().GetPrice()
	} else if execEvent.GetDeal() != nil {
		posID = execEvent.GetDeal().GetPositionId()
		execPrice = execEvent.GetDeal().GetExecutionPrice()
	}

	if posID == 0 {
		return broker.OpenResult{}, fmt.Errorf("cTrader execution event tidak mengembalikan ID posisi (ErrorCode: %s)", execEvent.GetErrorCode())
	}

	// Konfirmasi SL dan TP di sisi server
	if req.StopLoss > 0 || req.TakeProfit > 0 {
		amendReq := &openapi.ProtoOAAmendPositionSLTPReq{
			CtidTraderAccountId: &ctidAccID,
			PositionId:          &posID,
		}
		if req.StopLoss > 0 {
			sl := roundPrice(req.StopLoss)
			amendReq.StopLoss = &sl
		}
		if req.TakeProfit > 0 {
			tp := roundPrice(req.TakeProfit)
			amendReq.TakeProfit = &tp
		}
		_, _ = ctrader.Command[*openapi.ProtoOAAmendPositionSLTPReq, proto.Message](ctx, client, amendReq)
	}

	return broker.OpenResult{
		Ticket:        strconv.FormatInt(posID, 10),
		ExecutedPrice: execPrice,
		Lots:          req.Lots,
		StopLoss:      req.StopLoss,
		TakeProfit:    req.TakeProfit,
	}, nil
}

func (c *Client) ListPositions(ctx context.Context) ([]broker.PositionState, error) {
	c.mu.Lock()
	client := c.client
	authed := c.authed
	ctidAccID := c.ctidAccID
	c.mu.Unlock()

	if client == nil || !authed {
		return nil, broker.ErrNotConnected
	}

	res, err := ctrader.Command[*openapi.ProtoOAReconcileReq, *openapi.ProtoOAReconcileRes](
		ctx,
		client,
		&openapi.ProtoOAReconcileReq{
			CtidTraderAccountId: &ctidAccID,
		},
	)
	if err != nil {
		c.mu.Lock()
		c.authed = false
		c.mu.Unlock()
		return nil, fmt.Errorf("cTrader reconcile gagal: %w", err)
	}

	var list []broker.PositionState
	for _, p := range res.GetPosition() {
		symID := p.GetTradeData().GetSymbolId()
		symName := c.idToSym[symID]
		if symName == "" {
			symName = fmt.Sprintf("SYM_%d", symID)
		}

		mult := c.lotMult[symName]
		if mult == 0 {
			mult = 10000000
		}
		lots := float64(p.GetTradeData().GetVolume()) / float64(mult)
		lots = math.Round(lots*100) / 100

		side := "BUY"
		if p.GetTradeData().GetTradeSide() == openapi.ProtoOATradeSide_SELL {
			side = "SELL"
		}

		comment := ""
		if p.GetTradeData().Comment != nil {
			comment = *p.GetTradeData().Comment
		}

		list = append(list, broker.PositionState{
			Ticket:     strconv.FormatInt(p.GetPositionId(), 10),
			Symbol:     symName,
			Side:       side,
			Lots:       lots,
			OpenPrice:  p.GetPrice(),
			StopLoss:   p.GetStopLoss(),
			TakeProfit: p.GetTakeProfit(),
			Profit:     0,
			Comment:    comment,
		})
	}

	return list, nil
}

func (c *Client) ClosedSince(ctx context.Context, unixSeconds int64) ([]broker.ClosedPosition, error) {
	c.mu.Lock()
	client := c.client
	authed := c.authed
	ctidAccID := c.ctidAccID
	c.mu.Unlock()

	if client == nil || !authed {
		return nil, broker.ErrNotConnected
	}

	fromTs := unixSeconds * 1000
	toTs := time.Now().UnixMilli()

	res, err := ctrader.Command[*openapi.ProtoOADealListReq, *openapi.ProtoOADealListRes](
		ctx,
		client,
		&openapi.ProtoOADealListReq{
			CtidTraderAccountId: &ctidAccID,
			FromTimestamp:       &fromTs,
			ToTimestamp:         &toTs,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("mengambil deal tertutup: %w", err)
	}

	var closed []broker.ClosedPosition
	for _, d := range res.GetDeal() {
		if d.GetClosePositionDetail() == nil {
			continue
		}
		grossProfit := float64(d.GetClosePositionDetail().GetGrossProfit()) / 100.0
		reason := broker.CloseReasonOther
		closed = append(closed, broker.ClosedPosition{
			Ticket:    strconv.FormatInt(d.GetPositionId(), 10),
			ExitPrice: d.GetExecutionPrice(),
			Profit:    grossProfit,
			Reason:    reason,
			ClosedAt:  d.GetExecutionTimestamp() / 1000,
		})
	}

	return closed, nil
}
