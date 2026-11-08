package model

import "github.com/google/uuid"

// ensureID mengisi ID UUID bila belum ada.
// Penjaga ini dipakai entitas yang tidak memakai Base tetapi tetap punya
// primary key berupa UUID. Tanpa ini, PostgreSQL menolak baris ber-ID kosong
// sementara SQLite diam-diam menerimanya — selisih yang baru terlihat di
// produksi. Satu tempat untuk seluruh tabel agar tidak terulang.
func ensureID(id *string) error {
	if *id == "" {
		*id = uuid.NewString()
	}
	return nil
}
