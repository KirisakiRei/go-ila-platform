package service

import (
	"time"

	"google.golang.org/protobuf/encoding/protojson"

	enginev1 "ila/backend/gen/engine/v1"
	"ila/backend/internal/dto"
)

// parseHtfSweep membaca rincian sweep HTF yang tersimpan sebagai JSON proto.
//
// Protojson menerima nama field camelCase maupun nama proto (snake_case),
// sehingga baris lama yang disimpan sebelum perubahan format tetap terbaca.
func parseHtfSweep(raw string) *dto.SignalHtfSweep {
	if raw == "" {
		return nil
	}

	var info enginev1.HTFSweepInfo
	if err := protojson.Unmarshal([]byte(raw), &info); err != nil {
		return nil
	}

	out := &dto.SignalHtfSweep{
		Detected:        info.GetDetected(),
		SweepInProgress: info.GetSweepInProgress(),
		Direction:       info.GetDirection(),
		FVG:             fvgFromProto(info.GetFvg()),
		SweepCandle:     candleFromProto(info.GetSweepCandle()),
		SweptCandle:     candleFromProto(info.GetSweptCandle()),
		StopLoss:        info.GetStopLoss(),
		TargetDOL:       info.GetTargetDol(),
		Details:         info.GetDetails(),
		SweepTime:       timeOrNil(info.GetSweepTime().AsTime(), info.GetSweepTime() != nil),
	}
	return out
}

// parseCISD membaca rincian konfirmasi CISD dari JSON proto.
func parseCISD(raw string) *dto.SignalCISD {
	if raw == "" {
		return nil
	}

	var info enginev1.CISDInfo
	if err := protojson.Unmarshal([]byte(raw), &info); err != nil {
		return nil
	}

	return &dto.SignalCISD{
		Detected:    info.GetDetected(),
		Direction:   info.GetDirection(),
		RefCandle:   candleFromProto(info.GetRefCandle()),
		BreakCandle: candleFromProto(info.GetBreakCandle()),
		CISDLevel:   info.GetCisdLevel(),
		ExtremeSL:   info.GetExtremeSl(),
		EntryPrice:  info.GetEntryPrice(),
		Details:     info.GetDetails(),
		BreakTime:   timeOrNil(info.GetBreakTime().AsTime(), info.GetBreakTime() != nil),
	}
}

func candleFromProto(c *enginev1.CandleInfo) *dto.SignalCandle {
	if c == nil || c.GetTimestamp() == nil {
		return nil
	}
	return &dto.SignalCandle{
		Timestamp: c.GetTimestamp().AsTime().UTC(),
		Open:      c.GetOpen(),
		High:      c.GetHigh(),
		Low:       c.GetLow(),
		Close:     c.GetClose(),
		IsClosed:  c.GetIsClosed(),
	}
}

func fvgFromProto(g *enginev1.FairValueGap) *dto.SignalFairValueGap {
	if g == nil {
		return nil
	}
	return &dto.SignalFairValueGap{
		Type:      g.GetType(),
		Top:       g.GetTop(),
		Bottom:    g.GetBottom(),
		Candle3At: timeOrNil(g.GetCandle3Time().AsTime(), g.GetCandle3Time() != nil),
		Mitigated: g.GetMitigated(),
	}
}

// timeOrNil mengembalikan nil bila timestamp tidak ada, agar JSON tidak memuat
// waktu nol yang membingungkan klien.
func timeOrNil(t time.Time, ada bool) *time.Time {
	if !ada {
		return nil
	}
	utc := t.UTC()
	return &utc
}
