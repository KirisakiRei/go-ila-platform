package runner

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sort"
	"testing"
)

func silent() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// Penyelarasan harus mengirim hanya selisih, bukan seluruh daftar tiap kali.
func TestDiffSymbolsSelisih(t *testing.T) {
	sent := toSymbolSet([]string{"XAUUSD", "EURUSD"})
	desired := []string{"XAUUSD", "GBPUSD"}

	add, remove := diffSymbols(sent, desired)

	if len(add) != 1 || add[0] != "GBPUSD" {
		t.Fatalf("tambah = %v, ingin [GBPUSD]", add)
	}
	if len(remove) != 1 || remove[0] != "EURUSD" {
		t.Fatalf("cabut = %v, ingin [EURUSD]", remove)
	}
}

// Tanpa perubahan tidak ada panggilan ke engine sama sekali.
func TestSyncTidakMengirimBilaSama(t *testing.T) {
	var dipanggil int
	svc := &Service{logger: silent()}
	svc.SetSymbolSync("ila-2.0", func(context.Context, []string, []string) error {
		dipanggil++
		return nil
	})

	if err := svc.SyncSubscriptions(context.Background(), "ila-2.0", []string{"XAUUSD"}); err != nil {
		t.Fatalf("sinkron pertama: %v", err)
	}
	if err := svc.SyncSubscriptions(context.Background(), "ila-2.0", []string{"XAUUSD"}); err != nil {
		t.Fatalf("sinkron kedua: %v", err)
	}

	if dipanggil != 1 {
		t.Fatalf("panggilan engine = %d, ingin 1 (kedua tidak berubah)", dipanggil)
	}
}

// Gagal mengirim tidak boleh dianggap sudah terkirim: percobaan berikutnya
// harus mengulang daftar yang sama.
func TestSyncGagalMengulang(t *testing.T) {
	var panggilan int
	var terakhirAdd []string

	svc := &Service{logger: silent()}
	svc.SetSymbolSync("ila-2.0", func(_ context.Context, add, _ []string) error {
		panggilan++
		terakhirAdd = add
		if panggilan == 1 {
			return errors.New("engine mati")
		}
		return nil
	})

	ctx := context.Background()
	if err := svc.SyncSubscriptions(ctx, "ila-2.0", []string{"XAUUSD"}); err == nil {
		t.Fatal("kegagalan harus dikembalikan")
	}
	if len(svc.SymbolsSent("ila-2.0")) != 0 {
		t.Fatal("gagal kirim tidak boleh dianggap terkirim")
	}

	if err := svc.SyncSubscriptions(ctx, "ila-2.0", []string{"XAUUSD"}); err != nil {
		t.Fatalf("percobaan kedua: %v", err)
	}
	if len(terakhirAdd) != 1 || terakhirAdd[0] != "XAUUSD" {
		t.Fatalf("percobaan kedua harus mengulang XAUUSD, dapat %v", terakhirAdd)
	}
}

// Saat stream tersambung ulang, daftar terkirim dikosongkan agar engine (yang
// mungkin sudah lupa) menerima seluruh kebutuhan sekali lagi.
func TestSetSymbolSyncMengulangSetelahReconnect(t *testing.T) {
	svc := &Service{logger: silent()}
	svc.SetSymbolSync("ila-2.0", func(context.Context, []string, []string) error { return nil })
	if err := svc.SyncSubscriptions(context.Background(), "ila-2.0", []string{"XAUUSD"}); err != nil {
		t.Fatalf("sinkron: %v", err)
	}

	svc.SetSymbolSync("ila-2.0", func(context.Context, []string, []string) error { return nil })
	if got := svc.SymbolsSent("ila-2.0"); len(got) != 0 {
		t.Fatalf("setelah reconnect daftar harus kosong, dapat %v", got)
	}
}

// Langganan yang dicabut pengguna harus benar-benar dicabut di engine.
func TestSyncMencabutSimbolYangTidakDibutuhkan(t *testing.T) {
	svc := &Service{logger: silent()}
	var removed []string
	svc.SetSymbolSync("ila-2.0", func(_ context.Context, _, remove []string) error {
		removed = append(removed, remove...)
		return nil
	})

	ctx := context.Background()
	if err := svc.SyncSubscriptions(ctx, "ila-2.0", []string{"XAUUSD", "EURUSD"}); err != nil {
		t.Fatalf("sinkron awal: %v", err)
	}
	if err := svc.SyncSubscriptions(ctx, "ila-2.0", []string{"XAUUSD"}); err != nil {
		t.Fatalf("sinkron kedua: %v", err)
	}

	sort.Strings(removed)
	if len(removed) != 1 || removed[0] != "EURUSD" {
		t.Fatalf("dicabut = %v, ingin [EURUSD]", removed)
	}
}

// Tanpa engine terhubung, penyelarasan tidak melakukan apa pun dan tidak panik.
func TestSyncTanpaEngineAman(t *testing.T) {
	svc := &Service{logger: silent()}
	if err := svc.SyncSubscriptions(context.Background(), "ila-2.0", []string{"XAUUSD"}); err != nil {
		t.Fatalf("tanpa engine harus aman: %v", err)
	}
}

// Dua engine berjalan bersamaan dan masing-masing punya daftar simbol sendiri.
// Sebelumnya state sinkronisasi dipakai bersama, sehingga menyelaraskan satu
// engine mencabut langganan engine lain dan engine kembali tidak pernah
// menerima pembaruan.
func TestSyncTerpisahPerStrategi(t *testing.T) {
	svc := &Service{logger: silent()}

	var kiriman20, kiriman30 []string
	svc.SetSymbolSync("ila-2.0", func(_ context.Context, add, remove []string) error {
		kiriman20 = append(kiriman20, add...)
		kiriman20 = append(kiriman20, remove...)
		return nil
	})
	svc.SetSymbolSync("ila-3.0", func(_ context.Context, add, remove []string) error {
		kiriman30 = append(kiriman30, add...)
		kiriman30 = append(kiriman30, remove...)
		return nil
	})

	ctx := context.Background()
	if err := svc.SyncSubscriptions(ctx, "ila-2.0", []string{"EURUSD", "XAUUSD"}); err != nil {
		t.Fatalf("sinkron ila-2.0: %v", err)
	}
	if err := svc.SyncSubscriptions(ctx, "ila-3.0", []string{"XAUUSD", "XAGUSD"}); err != nil {
		t.Fatalf("sinkron ila-3.0: %v", err)
	}

	// Kiriman engine ILA 2.0 tidak boleh berisi simbol khas ILA 3.0.
	sort.Strings(kiriman20)
	if len(kiriman20) != 2 || kiriman20[0] != "EURUSD" || kiriman20[1] != "XAUUSD" {
		t.Fatalf("engine ila-2.0 menerima %v, ingin [EURUSD XAUUSD]", kiriman20)
	}
	sort.Strings(kiriman30)
	if len(kiriman30) != 2 || kiriman30[0] != "XAGUSD" || kiriman30[1] != "XAUUSD" {
		t.Fatalf("engine ila-3.0 menerima %v, ingin [XAGUSD XAUUSD]", kiriman30)
	}

	// Siklus berikutnya yang sama persis tidak boleh mengirim apa pun lagi.
	if err := svc.SyncSubscriptions(ctx, "ila-2.0", []string{"EURUSD", "XAUUSD"}); err != nil {
		t.Fatalf("sinkron ulang ila-2.0: %v", err)
	}
	if len(kiriman20) != 2 {
		t.Fatalf("sinkron ulang tidak boleh mengirim lagi, dapat %v", kiriman20)
	}
}
