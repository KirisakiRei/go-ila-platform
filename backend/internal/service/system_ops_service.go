package service

import (
	"context"
	"strings"
	"time"

	"ila/backend/internal/audit"
	"ila/backend/internal/dto"
	"ila/backend/internal/model"
	"ila/backend/internal/runner"
	"ila/backend/internal/settings"
	"ila/backend/pkg/apperror"
)

// MaintenanceConfirmWord adalah kata yang wajib diketik untuk mengubah mode
// selain normal. Kata ini sengaja menakutkan agar tidak dilakukan tanpa niat.
const MaintenanceConfirmWord = "MAINTENANCE"

// SystemStateStore adalah kebutuhan penyimpanan mode dan ringkasan.
type SystemStateStore interface {
	State(ctx context.Context) (*model.SystemState, error)
	SaveState(ctx context.Context, state *model.SystemState) error
	SupportNotes(ctx context.Context, userID string, limit int) ([]model.SupportNote, error)
	AddSupportNote(ctx context.Context, note *model.SupportNote) error
	BrokerHealth(ctx context.Context) (total, connected, errored, frozen int, err error)
	CountActiveAnnouncements(ctx context.Context) (int, error)
}

// EngineReader membaca status engine terakhir yang dilaporkan Runner.
type EngineReader interface {
	ListEngines(ctx context.Context) ([]model.EngineState, error)
}

// AdminSystemOps adalah layanan operasional: mode, kesehatan, catatan dukungan.
type AdminSystemOps interface {
	Overview(ctx context.Context) (*dto.SystemOverview, error)
	EngineDetail(ctx context.Context) ([]dto.EngineOverview, error)
	RunnerDetail(ctx context.Context) (*dto.RunnerOverview, error)
	BrokerDetail(ctx context.Context) (*dto.BrokerHealth, error)
	ChangeMode(ctx context.Context, actor AdminActor, req dto.MaintenanceRequest) (*dto.MaintenanceState, error)
	SupportNotes(ctx context.Context, userID string) ([]dto.SupportNoteItem, error)
	AddSupportNote(ctx context.Context, actor AdminActor, userID string, req dto.SupportNoteRequest) (*dto.SupportNoteItem, error)
}

// PublicStatusService menyajikan mode platform ke halaman publik.
type PublicStatusService interface {
	Status(ctx context.Context) (*dto.PublicStatus, error)
}

type systemOpsService struct {
	store   SystemStateStore
	engines EngineReader
	flags   *settings.Service
	audit   AdminAudit
	clock   func() time.Time
}

// NewSystemOpsService merangkai layanan operasional sistem.
func NewSystemOpsService(store SystemStateStore, engines EngineReader, flags *settings.Service, recorder AdminAudit) *systemOpsService {
	return &systemOpsService{
		store:   store,
		engines: engines,
		flags:   flags,
		audit:   recorder,
		clock:   func() time.Time { return time.Now().UTC() },
	}
}

// staleAfter adalah umur laporan yang masih dianggap hidup.
// Runner mengirim denyut tiap menit; tiga menit lewat berarti bermasalah.
const staleAfter = 3 * time.Minute

// Overview merangkum kondisi platform untuk halaman admin kesehatan.
func (s *systemOpsService) Overview(ctx context.Context) (*dto.SystemOverview, error) {
	state, err := s.store.State(ctx)
	if err != nil {
		return nil, err
	}

	engines, err := s.engines.ListEngines(ctx)
	if err != nil {
		return nil, err
	}

	now := s.clock()
	out := &dto.SystemOverview{
		Mode:           toMaintenanceState(state),
		Settings:       make([]dto.SettingRow, 0, len(settings.Catalog)),
		Engines:        make([]dto.EngineOverview, 0, len(engines)),
		ServerTime:     now,
		ServerTimePair: dto.NewTimePair(now),
	}

	for _, item := range s.flags.Snapshot(ctx) {
		out.Settings = append(out.Settings, dto.SettingRow{
			Key: item.Key, Label: item.Label, Help: item.Help, Kind: item.Kind,
			Value: item.Value, Source: item.Source, Danger: item.Danger,
			Min: item.Min, Max: item.Max,
		})
	}

	var lastSeen *time.Time
	runnerHealthy := false
	for _, engine := range engines {
		item := toEngineOverview(engine, now)
		out.Engines = append(out.Engines, item)
		if engine.LastSeenAt != nil && (lastSeen == nil || engine.LastSeenAt.After(*lastSeen)) {
			lastSeen = engine.LastSeenAt
		}
		if item.Connected {
			runnerHealthy = true
		}
	}
	out.LastEngineSeen = lastSeen
	// Runner dianggap hidup hanya bila ada engine yang baru melapor; status
	// engine tidak dihapus saat Runner mati, jadi waktu terakhir yang dipakai.
	out.Runner = dto.RunnerOverview{Connected: runnerHealthy, LastSeen: lastSeen, Stale: !runnerHealthy}

	total, connected, errored, frozen, err := s.store.BrokerHealth(ctx)
	if err != nil {
		return nil, err
	}
	out.BrokerHealth = dto.BrokerHealth{Total: total, Connected: connected, ErrorCount: errored, Frozen: frozen}

	count, err := s.store.CountActiveAnnouncements(ctx)
	if err != nil {
		return nil, err
	}
	out.Announcements = count
	return out, nil
}

// ChangeMode mengubah mode platform. Posisi berjalan tidak pernah disentuh:
// mode hanya mengatur penerimaan entry baru (plans/18 §4.3).
func (s *systemOpsService) ChangeMode(ctx context.Context, actor AdminActor, req dto.MaintenanceRequest) (*dto.MaintenanceState, error) {
	mode := strings.TrimSpace(req.Mode)
	if mode != model.ModeNormal && mode != model.ModeReadOnly && mode != model.ModeMaintenance {
		return nil, validation("mode tidak dikenal")
	}
	if mode != model.ModeNormal && strings.TrimSpace(req.Confirm) != MaintenanceConfirmWord {
		return nil, validation("ketik MAINTENANCE untuk konfirmasi")
	}
	if err := ValidateReason(audit.SystemModeChange, req.Reason); err != nil {
		return nil, err
	}

	var expected *time.Time
	if strings.TrimSpace(req.ExpectedEnd) != "" {
		parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(req.ExpectedEnd))
		if err != nil {
			return nil, validation("perkiraan selesai harus berformat RFC3339")
		}
		utc := parsed.UTC()
		expected = &utc
	}

	before, err := s.store.State(ctx)
	if err != nil {
		return nil, err
	}

	row := &model.SystemState{
		ID:          1,
		Mode:        mode,
		Reason:      strings.TrimSpace(req.Reason),
		ExpectedEnd: expected,
		ChangedBy:   actor.ID,
		ChangedAt:   s.clock(),
	}
	if err := s.store.SaveState(ctx, row); err != nil {
		return nil, err
	}

	if err := s.audit.Record(ctx, AuditEntry{
		ActorID: actor.ID, Action: audit.SystemModeChange,
		TargetType: "system_state", TargetID: "1",
		Reason: req.Reason, RequestID: actor.RequestID,
		Before: map[string]any{"mode": before.Mode},
		After:  map[string]any{"mode": mode},
	}); err != nil {
		return nil, err
	}

	result := toMaintenanceState(row)
	return &result, nil
}

// EngineDetail mengembalikan kondisi tiap engine tanpa ringkasan lain.
// Dipakai halaman /admin/engine agar admin bisa fokus pada satu hal.
func (s *systemOpsService) EngineDetail(ctx context.Context) ([]dto.EngineOverview, error) {
	engines, err := s.engines.ListEngines(ctx)
	if err != nil {
		return nil, err
	}
	now := s.clock()
	out := make([]dto.EngineOverview, 0, len(engines))
	for _, engine := range engines {
		out = append(out, toEngineOverview(engine, now))
	}
	return out, nil
}

// RunnerDetail menyimpulkan kesehatan Runner dari laporan engine terakhir.
// Runner menulis status lewat engine yang dilayaninya, jadi tidak ada tabel
// terpisah untuk denyutnya.
func (s *systemOpsService) RunnerDetail(ctx context.Context) (*dto.RunnerOverview, error) {
	engines, err := s.engines.ListEngines(ctx)
	if err != nil {
		return nil, err
	}
	now := s.clock()
	var lastSeen *time.Time
	healthy := false
	for _, engine := range engines {
		if engine.LastSeenAt == nil {
			continue
		}
		if lastSeen == nil || engine.LastSeenAt.After(*lastSeen) {
			lastSeen = engine.LastSeenAt
		}
		if now.Sub(*engine.LastSeenAt) <= staleAfter {
			healthy = true
		}
	}
	return &dto.RunnerOverview{Connected: healthy, LastSeen: lastSeen, Stale: !healthy}, nil
}

// BrokerDetail menghitung akun broker sehat, gagal, dan dibekukan.
func (s *systemOpsService) BrokerDetail(ctx context.Context) (*dto.BrokerHealth, error) {
	total, connected, errored, frozen, err := s.store.BrokerHealth(ctx)
	if err != nil {
		return nil, err
	}
	return &dto.BrokerHealth{Total: total, Connected: connected, ErrorCount: errored, Frozen: frozen}, nil
}

// toEngineOverview mengubah baris engine menjadi bentuk tampilan.
// Laporan yang lebih lama dari staleAfter ditandai basi walau statusnya
// masih tertulis tersambung.
func toEngineOverview(engine model.EngineState, now time.Time) dto.EngineOverview {
	stale := engine.LastSeenAt == nil || now.Sub(*engine.LastSeenAt) > staleAfter
	return dto.EngineOverview{
		Strategy:      engine.Strategy,
		Connected:     engine.Connected && !stale,
		EngineVersion: engine.EngineVersion,
		MarketState:   engine.MarketState,
		LastSeenAt:    engine.LastSeenAt,
		LastError:     engine.LastError,
		Stale:         stale,
	}
}

func (s *systemOpsService) SupportNotes(ctx context.Context, userID string) ([]dto.SupportNoteItem, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, apperror.ErrNotFound
	}
	rows, err := s.store.SupportNotes(ctx, userID, 100)
	if err != nil {
		return nil, err
	}
	out := make([]dto.SupportNoteItem, 0, len(rows))
	for _, row := range rows {
		out = append(out, toSupportNoteItem(row))
	}
	return out, nil
}

func (s *systemOpsService) AddSupportNote(ctx context.Context, actor AdminActor, userID string, req dto.SupportNoteRequest) (*dto.SupportNoteItem, error) {
	body := strings.TrimSpace(req.Body)
	if userID == "" || body == "" {
		return nil, validation("isi catatan wajib diisi")
	}
	channel := strings.TrimSpace(req.Channel)
	if channel == "" {
		channel = "internal"
	}
	note := &model.SupportNote{
		UserID:   userID,
		ActorID:  actor.ID,
		Channel:  channel,
		Body:     body,
		IsPinned: req.IsPinned,
	}
	if err := s.store.AddSupportNote(ctx, note); err != nil {
		return nil, err
	}
	// Jejak audit wajib: catatan ini menyangkut pelanggan, bukan milik sendiri.
	if err := s.audit.Record(ctx, AuditEntry{
		ActorID: actor.ID, Action: audit.SupportNoteAdd,
		TargetType: "user", TargetID: userID,
		RequestID: actor.RequestID,
		After:     map[string]any{"channel": channel},
	}); err != nil {
		return nil, err
	}
	item := toSupportNoteItem(*note)
	return &item, nil
}

// Status mengembalikan mode platform untuk halaman publik & dashboard.
func (s *systemOpsService) Status(ctx context.Context) (*dto.PublicStatus, error) {
	state, err := s.store.State(ctx)
	if err != nil {
		return nil, err
	}
	now := s.clock()
	out := &dto.PublicStatus{
		Mode:       state.Mode,
		Message:    maintenanceMessage(state.Mode),
		ServerTime: now,
		TimePair:   dto.NewTimePair(now),
		Market:     marketWindow(now),
		// Gagal membaca flag diperlakukan sebagai tertutup: registry yang
		// terbuka tiba-tiba lebih mahal daripada satu penolakan pendaftaran.
		InviteOnly: s.flags.Bool(ctx, settings.KeyInviteOnly, true),
	}
	if state.Mode != model.ModeNormal {
		out.ExpectedEnd = state.ExpectedEnd
	}
	return out, nil
}

// marketWindow menghitung jadwal pasar memakai aturan yang sama dengan
// Runner, sehingga dashboard dan eksekusi tidak pernah berbeda pendapat.
// Perubahan berikutnya dicari per jam: cukup untuk tampilan dan jauh lebih
// murah daripada memeriksa per menit sepanjang minggu.
func marketWindow(now time.Time) dto.MarketWindow {
	open := runner.ExpectedOpen(now)
	window := dto.MarketWindow{Open: open, Label: "Pasar buka"}
	if !open {
		window.Label = "Pasar tutup"
	}

	cursor := now.UTC().Truncate(time.Hour).Add(time.Hour)
	limit := now.Add(8 * 24 * time.Hour)
	for cursor.Before(limit) {
		if runner.ExpectedOpen(cursor) != open {
			window.NextChange = cursor
			window.NextChangePair = dto.NewTimePair(cursor)
			if open {
				window.NextLabel = "Pasar tutup"
			} else {
				window.NextLabel = "Pasar buka"
			}
			break
		}
		cursor = cursor.Add(time.Hour)
	}
	return window
}

func toMaintenanceState(state *model.SystemState) dto.MaintenanceState {
	if state == nil {
		return dto.MaintenanceState{Mode: model.ModeNormal, Label: maintenanceLabel(model.ModeNormal)}
	}
	return dto.MaintenanceState{
		Mode:        state.Mode,
		Label:       maintenanceLabel(state.Mode),
		Reason:      state.Reason,
		ExpectedEnd: state.ExpectedEnd,
		ChangedAt:   state.ChangedAt,
		ChangedBy:   state.ChangedBy,
	}
}

func maintenanceLabel(mode string) string {
	switch mode {
	case model.ModeReadOnly:
		return "Hanya baca"
	case model.ModeMaintenance:
		return "Pemeliharaan"
	default:
		return "Normal"
	}
}

func maintenanceMessage(mode string) string {
	switch mode {
	case model.ModeReadOnly:
		return "Sinyal baru ditahan sementara. Posisi yang sudah berjalan tetap berjalan."
	case model.ModeMaintenance:
		return "Sistem sedang dalam pemeliharaan. Posisi yang sudah berjalan tetap berjalan."
	default:
		return ""
	}
}

func toSupportNoteItem(note model.SupportNote) dto.SupportNoteItem {
	return dto.SupportNoteItem{
		ID:        note.ID,
		UserID:    note.UserID,
		ActorID:   note.ActorID,
		Channel:   note.Channel,
		Body:      note.Body,
		IsPinned:  note.IsPinned,
		CreatedAt: note.CreatedAt,
	}
}
