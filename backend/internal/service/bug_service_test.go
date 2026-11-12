package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"ila/backend/internal/model"
	"ila/backend/pkg/apperror"
)

// ---------------------------------------------------------------------------
// Tiruan
// ---------------------------------------------------------------------------

type bugStoreTiruan struct {
	reports     map[string]*model.BugReport
	attachments map[string]*model.BugReportAttachment
}

func newBugStoreTiruan() *bugStoreTiruan {
	return &bugStoreTiruan{
		reports:     map[string]*model.BugReport{},
		attachments: map[string]*model.BugReportAttachment{},
	}
}

func (s *bugStoreTiruan) Create(_ context.Context, r *model.BugReport) error {
	r.ID = fmt.Sprintf("rep-%d", len(s.reports)+1)
	s.reports[r.ID] = r
	return nil
}

func (s *bugStoreTiruan) AddAttachment(_ context.Context, a *model.BugReportAttachment) error {
	a.ID = fmt.Sprintf("att-%d", len(s.attachments)+1)
	s.attachments[a.ID] = a
	return nil
}

func (s *bugStoreTiruan) FindByID(_ context.Context, id string) (*model.BugReport, error) {
	r, ok := s.reports[id]
	if !ok {
		return nil, apperror.ErrNotFound
	}
	return r, nil
}

func (s *bugStoreTiruan) ListByUser(_ context.Context, userID string, _ int) ([]model.BugReport, error) {
	var out []model.BugReport
	for _, r := range s.reports {
		if r.UserID == userID {
			out = append(out, *r)
		}
	}
	return out, nil
}

func (s *bugStoreTiruan) List(context.Context, string, string, int) ([]model.BugReport, error) {
	var out []model.BugReport
	for _, r := range s.reports {
		out = append(out, *r)
	}
	return out, nil
}

func (s *bugStoreTiruan) AttachmentsOf(_ context.Context, reportID string) ([]model.BugReportAttachment, error) {
	var out []model.BugReportAttachment
	for _, a := range s.attachments {
		if a.ReportID == reportID {
			out = append(out, *a)
		}
	}
	return out, nil
}

func (s *bugStoreTiruan) FindAttachment(_ context.Context, reportID, attachmentID string) (*model.BugReportAttachment, error) {
	a, ok := s.attachments[attachmentID]
	if !ok || a.ReportID != reportID {
		return nil, apperror.ErrNotFound
	}
	return a, nil
}

func (s *bugStoreTiruan) UpdateStatus(_ context.Context, id, status, note string) error {
	r, ok := s.reports[id]
	if !ok {
		return apperror.ErrNotFound
	}
	r.Status = status
	r.AdminNote = note
	return nil
}

// ---------------------------------------------------------------------------
// Pembantu
// ---------------------------------------------------------------------------

func inputLaporan() BugInput {
	return BugInput{
		Category: model.BugKategoriTampilan,
		Title:    "Tombol hilang",
		Body:     "Tombol simpan tidak muncul di halaman pengaturan.",
	}
}

// pngPalsu menghasilkan byte yang diawali magic PNG.
func pngPalsu(ukuran int) []byte {
	head := []byte("\x89PNG\r\n\x1a\n")
	if ukuran <= len(head) {
		return head
	}
	return append(head, bytes.Repeat([]byte{'a'}, ukuran-len(head))...)
}

func svgBerbahaya() []byte {
	return []byte(`<?xml version="1.0"?><svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`)
}

// ---------------------------------------------------------------------------
// Tes
// ---------------------------------------------------------------------------

// TestLaporanTanpaLampiranTersimpan mengunci jalur dasar.
func TestLaporanTanpaLampiranTersimpan(t *testing.T) {
	store := newBugStoreTiruan()
	svc := NewBugService(store, t.TempDir(), nil)

	detail, err := svc.Create(context.Background(), "user-1", inputLaporan())
	if err != nil {
		t.Fatalf("Create error: %v", err)
	}
	if detail.Status != model.BugStatusBaru {
		t.Fatalf("status awal = %q; ingin %q", detail.Status, model.BugStatusBaru)
	}
	if len(store.reports) != 1 {
		t.Fatalf("laporan tersimpan = %d", len(store.reports))
	}
}

// TestSVGDitolak: SVG bisa memuat skrip, jadi TIDAK boleh diterima walau
// dikirim dengan Content-Type image/svg+xml.
func TestSVGDitolak(t *testing.T) {
	store := newBugStoreTiruan()
	svc := NewBugService(store, t.TempDir(), nil)

	in := inputLaporan()
	in.Attachment = &BugUpload{FileName: "jahat.svg", Reader: bytes.NewReader(svgBerbahaya())}

	if _, err := svc.Create(context.Background(), "user-1", in); err == nil {
		t.Fatal("SVG diterima sebagai lampiran; seharusnya ditolak")
	}
	if len(store.attachments) != 0 {
		t.Fatal("lampiran SVG tercatat di database")
	}
}

// TestHTMLDitolak memastikan halaman HTML yang disamarkan juga ditolak.
func TestHTMLDitolak(t *testing.T) {
	svc := NewBugService(newBugStoreTiruan(), t.TempDir(), nil)
	in := inputLaporan()
	in.Attachment = &BugUpload{
		FileName: "foto.png", // nama menipu: isinya HTML
		Reader:   bytes.NewReader([]byte("<html><script>alert(1)</script></html>")),
	}
	if _, err := svc.Create(context.Background(), "user-1", in); err == nil {
		t.Fatal("berkas HTML dengan nama .png diterima; tipe harus dinilai dari isi")
	}
}

// TestPathTraversalTidakMenyentuhJalur: nama berkas kiriman tidak boleh
// menentukan lokasi simpan.
func TestPathTraversalTidakMenyentuhJalur(t *testing.T) {
	dir := t.TempDir()
	store := newBugStoreTiruan()
	svc := NewBugService(store, dir, nil)

	in := inputLaporan()
	in.Attachment = &BugUpload{
		FileName: "../../../../etc/passwd.png",
		Reader:   bytes.NewReader(pngPalsu(64)),
	}
	if _, err := svc.Create(context.Background(), "user-1", in); err != nil {
		t.Fatalf("Create error: %v", err)
	}

	// Berkas harus berada di dalam folder laporan, dan namanya UUID.
	for _, att := range store.attachments {
		if strings.Contains(att.StoredAs, "/") || strings.Contains(att.StoredAs, "..") {
			t.Fatalf("nama simpan tidak aman: %q", att.StoredAs)
		}
		path := filepath.Join(dir, att.ReportID, att.StoredAs)
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("berkas lampiran tidak ada di tempat yang diharapkan: %v", err)
		}
	}

	// Tidak ada yang tertulis di luar folder lampiran.
	luar := filepath.Join(filepath.Dir(dir), "passwd.png")
	if _, err := os.Stat(luar); err == nil {
		t.Fatal("berkas tertulis di luar folder lampiran (path traversal berhasil)")
	}
}

// TestLampiranTerlaluBesarDitolak membatasi ukuran sebelum memakan memori.
func TestLampiranTerlaluBesarDitolak(t *testing.T) {
	store := newBugStoreTiruan()
	svc := NewBugService(store, t.TempDir(), nil)

	in := inputLaporan()
	in.Attachment = &BugUpload{
		FileName: "besar.png",
		Reader:   bytes.NewReader(pngPalsu(BugMaxLampiranByte + 2048)),
	}

	_, err := svc.Create(context.Background(), "user-1", in)
	if !errors.Is(err, ErrLampiranTerlaluBesar) {
		t.Fatalf("error = %v; ingin ErrLampiranTerlaluBesar", err)
	}
	if len(store.attachments) != 0 {
		t.Fatal("lampiran kebesaran tetap tercatat")
	}
}

// TestLampiranPNGSahTersimpan memastikan jalur normal berjalan.
func TestLampiranPNGSahTersimpan(t *testing.T) {
	dir := t.TempDir()
	store := newBugStoreTiruan()
	svc := NewBugService(store, dir, nil)

	in := inputLaporan()
	in.Attachment = &BugUpload{FileName: "bukti.png", Reader: bytes.NewReader(pngPalsu(512))}

	detail, err := svc.Create(context.Background(), "user-1", in)
	if err != nil {
		t.Fatalf("Create error: %v", err)
	}
	if len(detail.Attachments) != 1 {
		t.Fatalf("lampiran = %d; ingin 1", len(detail.Attachments))
	}
	if detail.Attachments[0].MimeType != "image/png" {
		t.Fatalf("mime = %q; ingin image/png", detail.Attachments[0].MimeType)
	}
}

// TestPemilikLainTidakBisaBukaLampiran mengunci akses silang.
func TestPemilikLainTidakBisaBukaLampiran(t *testing.T) {
	dir := t.TempDir()
	store := newBugStoreTiruan()
	svc := NewBugService(store, dir, nil)

	in := inputLaporan()
	in.Attachment = &BugUpload{FileName: "b.png", Reader: bytes.NewReader(pngPalsu(256))}
	detail, _ := svc.Create(context.Background(), "pemilik", in)
	attID := detail.Attachments[0].ID

	// Pemilik boleh.
	if _, err := svc.OpenAttachment(context.Background(), BugActor{UserID: "pemilik"}, detail.ID, attID); err != nil {
		t.Fatalf("pemilik ditolak: %v", err)
	}

	// Admin boleh (untuk triage).
	if _, err := svc.OpenAttachment(context.Background(), BugActor{UserID: "admin-1", Admin: true}, detail.ID, attID); err != nil {
		t.Fatalf("admin ditolak: %v", err)
	}

	// Orang lain tidak boleh, dan galatnya 404 bukan 403 supaya keberadaan
	// laporan orang lain tidak bisa ditebak.
	_, err := svc.OpenAttachment(context.Background(), BugActor{UserID: "orang-lain"}, detail.ID, attID)
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Fatalf("error = %v; ingin ErrNotFound (bukan 403)", err)
	}
}

// TestStatusTidakDikenalDitolak menjaga kolom status tetap bermakna.
func TestStatusTidakDikenalDitolak(t *testing.T) {
	svc := NewBugService(newBugStoreTiruan(), t.TempDir(), nil)
	err := svc.UpdateStatus(context.Background(), AdminActor{ID: "a"}, "rep-1", "entah", "")
	if err == nil {
		t.Fatal("status tak dikenal diterima")
	}
}

// TestKategoriTidakDikenalDitolak.
func TestKategoriTidakDikenalDitolak(t *testing.T) {
	svc := NewBugService(newBugStoreTiruan(), t.TempDir(), nil)
	in := inputLaporan()
	in.Category = "kategori-karangan"
	if _, err := svc.Create(context.Background(), "u", in); err == nil {
		t.Fatal("kategori tak dikenal diterima")
	}
}

// TestContentDispositionSelaluAttachment: tanpa attachment, berkas bisa
// dirender peramban dan menjadi jalur XSS.
func TestContentDispositionSelaluAttachment(t *testing.T) {
	kasing := []string{"bukti.png", "jahat.html", "../../etc/passwd", "a\"b.png"}
	for _, nama := range kasing {
		got := ContentDisposition(nama)
		if !strings.HasPrefix(got, "attachment;") {
			t.Fatalf("%q -> %q; harus diawali attachment", nama, got)
		}
		if strings.Contains(got, "..") {
			t.Fatalf("%q -> %q; masih memuat ..", nama, got)
		}
		if strings.Count(got, `"`) != 2 {
			t.Fatalf("%q -> %q; kutip di dalam nama merusak header", nama, got)
		}
	}
}

// TestPenyimpananBelumDikonfigurasiDitolak: salah konfigurasi tidak boleh
// diam-diam menulis berkas ke folder acak.
func TestPenyimpananBelumDikonfigurasiDitolak(t *testing.T) {
	svc := NewBugService(newBugStoreTiruan(), "", nil)
	in := inputLaporan()
	in.Attachment = &BugUpload{FileName: "b.png", Reader: bytes.NewReader(pngPalsu(256))}

	if _, err := svc.Create(context.Background(), "u", in); err == nil {
		t.Fatal("lampiran diterima padahal folder penyimpanan kosong")
	}
}

// TestWaktuLaporanTerbentuk memastikan field tanggal tidak kosong.
func TestWaktuLaporanTerbentuk(t *testing.T) {
	store := newBugStoreTiruan()
	svc := NewBugService(store, t.TempDir(), nil)
	detail, _ := svc.Create(context.Background(), "u", inputLaporan())
	if detail.CreatedAt == "" {
		t.Fatal("created_at kosong")
	}
	if _, err := time.Parse(time.RFC3339, detail.CreatedAt); err != nil {
		t.Fatalf("created_at bukan RFC3339: %q", detail.CreatedAt)
	}
}
