package model_test

import (
	"strings"
	"sync"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

// GORM mengubah "AmountIDR" menjadi "amount_id_r" bila kolom tidak ditulis
// eksplisit, sementara kueri memakai "amount_idr". SQLite mengabaikan beda ini,
// PostgreSQL tidak — sehingga kesalahan hanya muncul saat pindah database.
// Tes ini memastikan kolom rupiah bernama ..._idr.
func TestKolomRupiahTidakTerpecah(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("membuka database uji: %v", err)
	}

	naming := db.NamingStrategy
	cache := &sync.Map{}
	for _, value := range allModels() {
		parsed, err := schema.Parse(value, cache, naming)
		if err != nil {
			t.Fatalf("membaca skema %T: %v", value, err)
		}
		for _, field := range parsed.Fields {
			if strings.Contains(field.DBName, "_id_r") {
				t.Fatalf("%s.%s menjadi kolom %q, seharusnya berakhiran _idr",
					parsed.Table, field.Name, field.DBName)
			}
		}
	}
}
