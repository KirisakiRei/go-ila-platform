package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"ila/backend/internal/model"
)

// AssistantRepository mengelola sesi tunggal per pengguna, pesan percakapan,
// penandaan compaction, dan memori awet ILA AI.
type AssistantRepository interface {
	EnsureSchema(ctx context.Context) error
	GetOrCreateSession(ctx context.Context, userID string) (*model.AISession, error)
	GetUserMemory(ctx context.Context, userID string) (*model.AIUserMemory, error)
	ListActiveMessages(ctx context.Context, sessionID string, limit int) ([]model.AIMessage, error)
	ListDisplayMessages(ctx context.Context, sessionID string, beforeSeq int64, limit int) ([]model.AIMessage, bool, error)
	CountActiveMessages(ctx context.Context, sessionID string) (int64, error)
	AppendMessages(ctx context.Context, sessionID, userID string, lastMessageAt time.Time, msgs []model.AIMessage) ([]model.AIMessage, error)
	ApplyCompaction(ctx context.Context, sessionID, userID string, upToSeq int64, rollingSummary, memoryMarkdown string, at time.Time) error
}

type assistantRepository struct {
	db *gorm.DB
}

// NewAssistantRepository membuat repositori ILA AI.
func NewAssistantRepository(db *gorm.DB) AssistantRepository {
	return &assistantRepository{db: db}
}

func (r *assistantRepository) EnsureSchema(ctx context.Context) error {
	return r.db.WithContext(ctx).AutoMigrate(
		&model.AISession{},
		&model.AIMessage{},
		&model.AIUserMemory{},
	)
}

func (r *assistantRepository) GetOrCreateSession(ctx context.Context, userID string) (*model.AISession, error) {
	var sess model.AISession
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&sess).Error
	if err == nil {
		return &sess, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("membaca sesi asisten: %w", err)
	}

	sess = model.AISession{UserID: userID}
	if createErr := r.db.WithContext(ctx).Create(&sess).Error; createErr != nil {
		// Tangani balapan inisialisasi pertama antar-tab dengan membaca ulang.
		if retryErr := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&sess).Error; retryErr == nil {
			return &sess, nil
		}
		return nil, fmt.Errorf("membuat sesi asisten: %w", createErr)
	}
	return &sess, nil
}

func (r *assistantRepository) GetUserMemory(ctx context.Context, userID string) (*model.AIUserMemory, error) {
	var mem model.AIUserMemory
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&mem).Error
	if err == nil {
		return &mem, nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &model.AIUserMemory{UserID: userID}, nil
	}
	return nil, fmt.Errorf("membaca memori asisten: %w", err)
}

func (r *assistantRepository) ListActiveMessages(ctx context.Context, sessionID string, limit int) ([]model.AIMessage, error) {
	if limit <= 0 || limit > 200 {
		limit = 200
	}
	var rows []model.AIMessage
	err := r.db.WithContext(ctx).
		Where("session_id = ? AND is_compacted = ?", sessionID, false).
		Order("seq ASC").
		Limit(limit).
		Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("membaca pesan aktif asisten: %w", err)
	}
	return rows, nil
}

// ListDisplayMessages mengembalikan pesan tampilan UI (user & assistant
// dengan konten non-kosong) terurut kronologis ASC.
func (r *assistantRepository) ListDisplayMessages(ctx context.Context, sessionID string, beforeSeq int64, limit int) ([]model.AIMessage, bool, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}

	q := r.db.WithContext(ctx).
		Where("session_id = ? AND role IN ? AND content <> ''", sessionID, []string{"user", "assistant"})
	if beforeSeq > 0 {
		q = q.Where("seq < ?", beforeSeq)
	}

	var desc []model.AIMessage
	if err := q.Order("seq DESC").Limit(limit + 1).Find(&desc).Error; err != nil {
		return nil, false, fmt.Errorf("membaca riwayat chat asisten: %w", err)
	}

	hasMore := len(desc) > limit
	if hasMore {
		desc = desc[:limit]
	}

	out := make([]model.AIMessage, len(desc))
	for i := range desc {
		out[len(desc)-1-i] = desc[i]
	}
	return out, hasMore, nil
}

func (r *assistantRepository) CountActiveMessages(ctx context.Context, sessionID string) (int64, error) {
	var total int64
	if err := r.db.WithContext(ctx).
		Model(&model.AIMessage{}).
		Where("session_id = ? AND is_compacted = ?", sessionID, false).
		Count(&total).Error; err != nil {
		return 0, fmt.Errorf("menghitung pesan aktif asisten: %w", err)
	}
	return total, nil
}

func (r *assistantRepository) AppendMessages(ctx context.Context, sessionID, userID string, lastMessageAt time.Time, msgs []model.AIMessage) ([]model.AIMessage, error) {
	if len(msgs) == 0 {
		return nil, nil
	}
	saved := make([]model.AIMessage, len(msgs))
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var maxSeq int64
		if err := tx.Model(&model.AIMessage{}).
			Where("session_id = ?", sessionID).
			Select("COALESCE(MAX(seq), 0)").
			Scan(&maxSeq).Error; err != nil {
			return fmt.Errorf("membaca seq terakhir: %w", err)
		}

		for i := range msgs {
			maxSeq++
			m := msgs[i]
			m.SessionID = sessionID
			m.UserID = userID
			m.Seq = maxSeq
			if m.CreatedAt.IsZero() {
				m.CreatedAt = lastMessageAt
			}
			if err := tx.Create(&m).Error; err != nil {
				return fmt.Errorf("menyimpan pesan asisten: %w", err)
			}
			saved[i] = m
		}

		if err := tx.Model(&model.AISession{}).
			Where("id = ? AND user_id = ?", sessionID, userID).
			Updates(map[string]any{
				"last_message_at": lastMessageAt,
				"updated_at":      lastMessageAt,
			}).Error; err != nil {
			return fmt.Errorf("memperbarui waktu pesan sesi: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return saved, nil
}

func (r *assistantRepository) ApplyCompaction(ctx context.Context, sessionID, userID string, upToSeq int64, rollingSummary, memoryMarkdown string, at time.Time) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if upToSeq > 0 {
			if err := tx.Model(&model.AIMessage{}).
				Where("session_id = ? AND seq <= ? AND is_compacted = ?", sessionID, upToSeq, false).
				Update("is_compacted", true).Error; err != nil {
				return fmt.Errorf("menandai pesan ter-compact: %w", err)
			}
		}

		sessUpdates := map[string]any{
			"rolling_summary":   rollingSummary,
			"last_compacted_at": at,
			"updated_at":        at,
		}
		if upToSeq > 0 {
			sessUpdates["compacted_before_seq"] = upToSeq
		}
		if err := tx.Model(&model.AISession{}).
			Where("id = ? AND user_id = ?", sessionID, userID).
			Updates(sessUpdates).Error; err != nil {
			return fmt.Errorf("memperbarui ringkasan sesi: %w", err)
		}

		var mem model.AIUserMemory
		findErr := tx.Where("user_id = ?", userID).First(&mem).Error
		if errors.Is(findErr, gorm.ErrRecordNotFound) {
			mem = model.AIUserMemory{
				UserID:         userID,
				MemoryMarkdown: memoryMarkdown,
				UpdatedAt:      at,
			}
			if err := tx.Create(&mem).Error; err != nil {
				return fmt.Errorf("membuat memori pengguna: %w", err)
			}
			return nil
		}
		if findErr != nil {
			return fmt.Errorf("membaca memori pengguna: %w", findErr)
		}

		if err := tx.Model(&model.AIUserMemory{}).
			Where("user_id = ?", userID).
			Updates(map[string]any{
				"memory_markdown": memoryMarkdown,
				"updated_at":      at,
			}).Error; err != nil {
			return fmt.Errorf("memperbarui memori pengguna: %w", err)
		}
		return nil
	})
}
