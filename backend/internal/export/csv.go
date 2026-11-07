package export

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"time"

	"ila/backend/internal/pagination"
	"ila/backend/internal/timezone"
)

// BOM adalah penanda urutan byte yang membuat Excel mengenali UTF-8.
// Tanpa ini, nama beraksen tampil rusak di Excel Windows Indonesia.
var BOM = []byte{0xEF, 0xBB, 0xBF}

// Builder menyusun berkas CSV dengan dua kolom waktu.
type Builder struct {
	buf     bytes.Buffer
	writer  *csv.Writer
	rowCap  int
	written int
}

// New membuat penyusun CSV dengan batas baris keras.
func New(rowCap int) *Builder {
	if rowCap <= 0 || rowCap > pagination.ExportMaxRows {
		rowCap = pagination.ExportMaxRows
	}
	b := &Builder{rowCap: rowCap}
	b.buf.Write(BOM)
	b.writer = csv.NewWriter(&b.buf)
	return b
}

// Header menulis baris judul.
func (b *Builder) Header(columns ...string) error {
	return b.writer.Write(columns)
}

// Row menulis satu baris. Baris melebihi batas ditolak agar server tidak
// tersedak oleh ekspor tanpa batas (plans/24 §6).
func (b *Builder) Row(values ...string) error {
	if b.written >= b.rowCap {
		return fmt.Errorf("ekspor melebihi %d baris; persempit filter atau rentang tanggal", b.rowCap)
	}
	b.written++
	return b.writer.Write(values)
}

// Finished menyelesaikan penulisan dan mengembalikan isi berkas.
func (b *Builder) Finished() []byte {
	b.writer.Flush()
	return b.buf.Bytes()
}

// Rows mengembalikan jumlah baris data yang sudah ditulis.
func (b *Builder) Rows() int {
	return b.written
}

// TimeColumns mengembalikan dua kolom waktu untuk satu titik waktu.
// Nil mengembalikan dua sel kosong agar baris tetap rata.
func TimeColumns(t *time.Time) (string, string) {
	if t == nil || t.IsZero() {
		return "", ""
	}
	return t.In(timezone.WIB).Format("2006-01-02 15:04:05"),
		t.In(timezone.NY).Format("2006-01-02 15:04:05")
}
