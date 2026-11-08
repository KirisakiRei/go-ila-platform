package ingest

import (
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	enginev1 "ila/backend/gen/engine/v1"
	"ila/backend/internal/model"
)

// ConvertSignal menerjemahkan payload engine menjadi entitas penyimpanan.
//
// Konversi dibuat terpisah dari alur eksekusi agar aturan trading tetap bisa
// diuji tanpa koneksi gRPC.
func ConvertSignal(in *enginev1.Signal) (*model.Signal, bool) {
	if in == nil || in.GetSignalId() == "" || in.GetSymbol() == "" {
		return nil, false
	}

	direction := directionName(in.GetDirection())
	if direction == "" {
		return nil, false
	}

	publishedAt := timestampOrZero(in.GetPublishedAt())
	expiresAt := timestampOrZero(in.GetExpiresAt())

	status := statusName(in.GetStatus())

	// Sinyal tanpa TTL tidak boleh dieksekusi: batas waktu adalah bagian
	// dari kontrak, bukan pelengkap opsional.
	if expiresAt.IsZero() {
		expiresAt = publishedAt
	}

	return &model.Signal{
		SignalID:        in.GetSignalId(),
		Symbol:          in.GetSymbol(),
		StrategyVersion: in.GetStrategyVersion(),
		EngineVersion:   in.GetEngineVersion(),
		Direction:       direction,
		EntryPrice:      in.GetEntryPrice(),
		StopLoss:        in.GetStopLoss(),
		TakeProfit:      in.GetTakeProfit(),
		RiskReward:      in.GetRiskReward(),
		Status:          status,
		Summary:         in.GetSummary(),
		NextAction:      in.GetNextAction(),
		PublishedAt:     publishedAt,
		ExpiresAt:       expiresAt,
	}, true
}

func directionName(d enginev1.Direction) string {
	switch d {
	case enginev1.Direction_BUY:
		return "BUY"
	case enginev1.Direction_SELL:
		return "SELL"
	default:
		return ""
	}
}

func statusName(s enginev1.SignalStatus) string {
	switch s {
	case enginev1.SignalStatus_READY_TO_ENTRY:
		return model.SignalStatusReadyToEntry
	case enginev1.SignalStatus_WAIT_HTF_SWEEP:
		return "WAIT_HTF_SWEEP"
	case enginev1.SignalStatus_WAIT_HTF_CANDLE_CLOSE:
		return "WAIT_HTF_CANDLE_CLOSE"
	case enginev1.SignalStatus_WAIT_LTF_CISD:
		return "WAIT_LTF_CISD"
	case enginev1.SignalStatus_INVALIDATED:
		return "INVALIDATED"

	// ILA 3.0: dua titik entry dipetakan ke status siap yang sama agar Runner
	// tidak perlu tahu perbedaan internal strategi. Detail aslinya tetap
	// tersimpan di sequence_state (plans/07).
	case enginev1.SignalStatus_READY_ENTRY_1, enginev1.SignalStatus_READY_ENTRY_2:
		return model.SignalStatusReadyToEntry
	case enginev1.SignalStatus_WAIT_ILA3_HTF_KEY_LEVEL:
		return "WAIT_ILA3_HTF_KEY_LEVEL"
	case enginev1.SignalStatus_WAIT_ILA3_MTF_REVERSAL:
		return "WAIT_ILA3_MTF_REVERSAL"
	case enginev1.SignalStatus_WAIT_ILA3_LTF_CISD_1:
		return "WAIT_ILA3_LTF_CISD_1"
	case enginev1.SignalStatus_WAIT_ILA3_MTF_FVG_RETR:
		return "WAIT_ILA3_MTF_FVG_RETR"
	case enginev1.SignalStatus_WAIT_ILA3_LTF_CISD_2:
		return "WAIT_ILA3_LTF_CISD_2"
	case enginev1.SignalStatus_ILA3_EXPANSION_1_ACTIVE:
		return "ILA3_EXPANSION_1_ACTIVE"
	case enginev1.SignalStatus_ILA3_EXPANSION_2_ACTIVE:
		return "ILA3_EXPANSION_2_ACTIVE"
	case enginev1.SignalStatus_ILA3_TARGET_REACHED:
		return "ILA3_TARGET_REACHED"
	default:
		return ""
	}
}

// UpdateStatus menerjemahkan aksi pembaruan engine menjadi status lokal.
func UpdateStatus(action enginev1.SignalAction) string {
	switch action {
	case enginev1.SignalAction_SIGNAL_ACTION_CANCEL:
		return model.SignalStatusCancelled
	case enginev1.SignalAction_SIGNAL_ACTION_EXPIRE:
		return model.SignalStatusExpired
	case enginev1.SignalAction_SIGNAL_ACTION_SUPERSEDE:
		return model.SignalStatusSuperseded
	default:
		return ""
	}
}

// MarketOpen menandai pasar terbuka menurut engine.
func MarketOpen(state enginev1.MarketState) bool {
	return state == enginev1.MarketState_MARKET_OPEN
}

func timestampOrZero(ts *timestamppb.Timestamp) time.Time {
	if ts == nil {
		return time.Time{}
	}
	return ts.AsTime().UTC()
}
