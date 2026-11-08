package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"ila/backend/internal/audit"
	"ila/backend/internal/dto"
	"ila/backend/internal/model"
	"ila/backend/pkg/apperror"
)

// CreditCodes membaca daftar voucher saldo beserta pemakaiannya.
func (s *adminBillingService) CreditCodes(ctx context.Context, status, q string) ([]dto.CreditCodeItem, error) {
	rows, err := s.store.ListCreditCodes(ctx, status, q, 100)
	if err != nil {
		return nil, err
	}
	now := s.now()
	out := make([]dto.CreditCodeItem, 0, len(rows))
	for _, row := range rows {
		used, err := s.store.CountCreditRedemptions(ctx, row.ID)
		if err != nil {
			return nil, err
		}
		out = append(out, toCreditCodeItem(row, used, now))
	}
	return out, nil
}

// CreateCreditCode membuat voucher saldo baru.
func (s *adminBillingService) CreateCreditCode(ctx context.Context, actor AdminActor, req dto.CreditCodeRequest) (*dto.CreditCodeItem, error) {
	code := strings.ToUpper(strings.TrimSpace(req.Code))
	if code == "" {
		return nil, validation("kode kredit wajib diisi")
	}

	startsAt, err := time.Parse(time.RFC3339, req.StartsAt)
	if err != nil {
		return nil, validation("tanggal mulai harus format RFC3339")
	}
	endsAt, err := time.Parse(time.RFC3339, req.EndsAt)
	if err != nil {
		return nil, validation("tanggal akhir harus format RFC3339")
	}
	if !endsAt.After(startsAt) {
		return nil, validation("tanggal akhir harus setelah tanggal mulai")
	}
	// Kode berlaku paling lama setahun: tujuan voucher adalah kampanye singkat,
	// bukan kewajiban terbuka tanpa batas.
	if endsAt.Sub(startsAt) > 366*24*time.Hour {
		return nil, validation("masa berlaku kode kredit maksimal satu tahun")
	}

	maxPerUser := req.MaxPerUser
	if maxPerUser <= 0 {
		maxPerUser = 1
	}
	maxRedemptions := req.MaxRedemptions
	if maxRedemptions <= 0 {
		maxRedemptions = 1
	}

	row := &model.CreditCode{
		Code: code, AmountIDR: req.AmountIDR,
		MaxRedemptions: maxRedemptions, MaxPerUser: maxPerUser,
		StartsAt: startsAt.UTC(), EndsAt: endsAt.UTC(),
		IsActive: true, CreatedBy: actor.ID, CreatedAt: s.now(),
	}
	if err := s.store.CreateCreditCode(ctx, row); err != nil {
		if errors.Is(err, apperror.ErrAlreadyExists) {
			return nil, errors.Join(apperror.ErrConflict, errors.New("kode kredit sudah dipakai"))
		}
		return nil, err
	}
	if err := s.audit.Record(ctx, AuditEntry{
		ActorID: actor.ID, Action: audit.CreditAdjust,
		TargetType: "credit_code", TargetID: row.ID, Reason: req.Reason,
		RequestID: actor.RequestID,
		After:     map[string]any{"code": row.Code, "amount_idr": row.AmountIDR, "max_redemptions": row.MaxRedemptions},
	}); err != nil {
		return nil, err
	}

	item := toCreditCodeItem(*row, 0, s.now())
	return &item, nil
}

// DisableCreditCode menghentikan kode tanpa menghapus riwayat penukarannya.
func (s *adminBillingService) DisableCreditCode(ctx context.Context, actor AdminActor, id, reason string) error {
	row, err := s.store.FindCreditCodeByID(ctx, id)
	if err != nil {
		return err
	}
	if !row.IsActive {
		return nil
	}
	if err := s.store.SetCreditCodeActive(ctx, id, false); err != nil {
		return err
	}
	return s.audit.Record(ctx, AuditEntry{
		ActorID: actor.ID, Action: audit.CreditAdjust,
		TargetType: "credit_code", TargetID: id, Reason: reason, RequestID: actor.RequestID,
		Before: map[string]any{"is_active": true},
		After:  map[string]any{"is_active": false},
	})
}

// Invoices membaca tagihan seluruh pengguna untuk panel admin.
func (s *adminBillingService) Invoices(ctx context.Context, status, q string, limit int) ([]dto.AdminInvoiceItem, error) {
	rows, err := s.store.ListInvoicesByStatus(ctx, status, q, limit)
	if err != nil {
		return nil, err
	}
	out := make([]dto.AdminInvoiceItem, 0, len(rows))
	for _, row := range rows {
		item := dto.AdminInvoiceItem{
			ID: row.ID, Number: row.Number, UserID: row.UserID, PlanCode: row.PlanCode,
			AmountIDR: row.AmountIDR, DiscountIDR: row.DiscountIDR, CreditIDR: row.CreditIDR,
			TotalIDR: row.TotalIDR, Status: row.Status, CouponCode: row.CouponCode,
			Gateway: row.Gateway, PeriodStart: dto.NewTimePair(row.PeriodStart),
			PeriodEnd: dto.NewTimePair(row.PeriodEnd), CreatedAt: dto.NewTimePair(row.CreatedAt),
			ExpiresAt: dto.NewTimePair(row.ExpiresAt),
		}
		if row.PaidAt != nil {
			pair := dto.NewTimePair(*row.PaidAt)
			item.PaidAt = &pair
		}
		out = append(out, item)
	}
	return out, nil
}

// MarkInvoicePaid mencatat pembayaran manual (transfer bank) sebagai lunas.
//
// Ini satu-satunya jalur di mana admin bisa memberi hak paket tanpa webhook,
// jadi alasannya wajib diisi dan seluruh aksinya tercatat di audit.
func (s *adminBillingService) MarkInvoicePaid(ctx context.Context, actor AdminActor, id string, req dto.ManualPaymentRequest) error {
	if err := ValidateReason(audit.InvoiceMarkPaid, req.Note); err != nil {
		return err
	}
	invoice, err := s.store.FindInvoiceByID(ctx, id)
	if err != nil {
		return err
	}
	if invoice.Status == model.InvoicePaid {
		return errors.Join(apperror.ErrConflict, errors.New("invoice sudah lunas"))
	}
	if invoice.Status == model.InvoiceRefunded || invoice.Status == model.InvoiceCancelled {
		return errors.Join(apperror.ErrConflict, errors.New("invoice tidak dapat dilunasi pada status ini"))
	}
	// Nominal manual tidak boleh kurang dari tagihan: kekurangan bayar tetap
	// ditolak seperti pada webhook.
	if req.AmountIDR < invoice.TotalIDR {
		return validation("nominal pembayaran kurang dari total tagihan")
	}

	now := s.now()
	txnID := "manual-" + invoice.Number
	if _, err := s.store.SavePayment(ctx, &model.Payment{
		InvoiceID: invoice.ID, Gateway: "manual",
		GatewayTxnID: txnID, AmountIDR: req.AmountIDR,
		Status: model.InvoicePaid, SignatureValid: false,
		RawPayload: truncateNote(req.Note), ReceivedAt: now,
	}); err != nil {
		return err
	}

	start := invoice.PeriodStart
	if start.IsZero() {
		start = now
	}
	end := invoice.PeriodEnd
	if !end.After(start) {
		end = start.AddDate(0, 1, 0)
	}
	invoice.Status = model.InvoicePaid
	invoice.PaidAt = &now
	invoice.Gateway = "manual"
	invoice.GatewayRef = txnID
	invoice.PeriodStart = start
	invoice.PeriodEnd = end
	invoice.UpdatedAt = now
	if err := s.store.UpdateInvoice(ctx, invoice); err != nil {
		return err
	}
	if err := s.subsaver.ActivatePlan(ctx, invoice.UserID, invoice.PlanCode, start, end); err != nil {
		return err
	}

	return s.audit.Record(ctx, AuditEntry{
		ActorID: actor.ID, Action: audit.InvoiceMarkPaid,
		TargetType: "invoice", TargetID: invoice.ID, Reason: req.Note, RequestID: actor.RequestID,
		Before: map[string]any{"status": model.InvoicePending},
		After: map[string]any{
			"status": model.InvoicePaid, "amount_idr": req.AmountIDR,
			"number": invoice.Number, "plan": invoice.PlanCode, "user_id": invoice.UserID,
		},
	})
}

// RefundPayment mencatat pengembalian dana manual.
//
// Refund tidak menurunkan tier secara otomatis: keputusan mencabut akses
// adalah keputusan bisnis terpisah dan dilakukan lewat ubah tier.
func (s *adminBillingService) RefundPayment(ctx context.Context, actor AdminActor, invoiceID string, req dto.RefundRequest) error {
	if err := ValidateReason(audit.RefundCreate, req.Reason); err != nil {
		return err
	}
	invoice, err := s.store.FindInvoiceByID(ctx, invoiceID)
	if err != nil {
		return err
	}
	if invoice.Status != model.InvoicePaid && invoice.Status != model.InvoiceRefunded {
		return errors.Join(apperror.ErrConflict, errors.New("hanya invoice lunas yang dapat direfund"))
	}

	paymentID := strings.TrimSpace(req.PaymentID)
	if paymentID == "" {
		return validation("pembayaran yang direfund wajib dipilih")
	}

	// Total refund tidak boleh melebihi nilai yang benar-benar dibayar.
	refunded, err := s.store.RefundedAmount(ctx, paymentID)
	if err != nil {
		return err
	}
	if refunded+req.AmountIDR > invoice.TotalIDR {
		return validation("total refund melebihi nilai pembayaran")
	}

	refund := &model.Refund{
		PaymentID: paymentID, AmountIDR: req.AmountIDR, Reason: req.Reason,
		ActorID: actor.ID, GatewayRef: strings.TrimSpace(req.GatewayRef),
		Status: "pending", CreatedAt: s.now(),
	}
	if err := s.store.AddRefund(ctx, refund); err != nil {
		return err
	}

	if refunded+req.AmountIDR == invoice.TotalIDR {
		invoice.Status = model.InvoiceRefunded
		invoice.UpdatedAt = s.now()
		if err := s.store.UpdateInvoice(ctx, invoice); err != nil {
			return err
		}
	}

	return s.audit.Record(ctx, AuditEntry{
		ActorID: actor.ID, Action: audit.RefundCreate,
		TargetType: "invoice", TargetID: invoice.ID, Reason: req.Reason, RequestID: actor.RequestID,
		After: map[string]any{
			"refund_idr": refund.AmountIDR, "payment_id": paymentID,
			"gateway_ref": refund.GatewayRef, "invoice_status": invoice.Status,
		},
	})
}

// toCreditCodeItem menentukan status voucher dari waktu dan pemakaiannya.
func toCreditCodeItem(row model.CreditCode, used int64, now time.Time) dto.CreditCodeItem {
	status := "active"
	switch {
	case !row.IsActive:
		status = "disabled"
	case now.After(row.EndsAt):
		status = "expired"
	case now.Before(row.StartsAt):
		status = "scheduled"
	case row.MaxRedemptions > 0 && used >= int64(row.MaxRedemptions):
		status = "exhausted"
	}
	return dto.CreditCodeItem{
		ID: row.ID, Code: row.Code, AmountIDR: row.AmountIDR,
		MaxRedemptions: row.MaxRedemptions, MaxPerUser: row.MaxPerUser,
		StartsAt: dto.NewTimePair(row.StartsAt), EndsAt: dto.NewTimePair(row.EndsAt),
		IsActive: row.IsActive, Redeemed: used, Status: status,
	}
}

// truncateNote memotong catatan bebas sebelum disimpan sebagai payload.
func truncateNote(note string) string {
	if len(note) > 500 {
		return note[:500]
	}
	return note
}
