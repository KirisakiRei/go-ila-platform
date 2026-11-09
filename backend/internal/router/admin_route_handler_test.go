package router

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

/*
Setiap registrasi route admin wajib menyertakan handler, bukan hanya guard.

Bug yang dijaga di sini pernah lolos ke produksi:

	admin.GET("/roles", guard(http.MethodGet, "/roles", authz.RoleView))

Penjaga permission menjadi handler terakhir. Ia memanggil c.Next(), tidak ada
yang menulis balasan, dan Gin mengirim `200 OK` dengan body 0 byte. Panel admin
membaca body kosong itu sebagai "Balasan server tidak dapat dibaca", sehingga
halaman RBAC dan dokumen legal tidak bisa dibuka meski statusnya 200.

Pengujian ini membaca sumber router.go dan memeriksa bentuk pemanggilan
registrasinya. Pemeriksaan statis dipilih karena router penuh baru bisa
dirangkai dengan basis data, sedangkan cacat yang dicari terlihat dari bentuk
kodenya.
*/

// allowedRouteCallNames adalah pemanggilan yang menandakan registrasi route
// sudah memiliki handler.
//
// - xx.GET/POST/...("/path", handler...)   : minimal dua argumen
// - .Register(group, guard)                : handler mendaftarkan route sendiri
var adminRouteMethods = map[string]bool{
	"GET": true, "POST": true, "PUT": true, "PATCH": true, "DELETE": true,
	"HEAD": true, "OPTIONS": true, "Any": true,
}

func TestRegistrasiRouteAdminSelaluPunyaHandler(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "router.go", nil, 0)
	if err != nil {
		t.Fatalf("membaca router.go: %v", err)
	}

	diperiksa := 0

	ast.Inspect(file, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || !adminRouteMethods[sel.Sel.Name] {
			return true
		}

		// Hanya pemanggilan pada grup admin: penerima bernama `admin`, atau
		// `g`/`me` yang dipakai handler.Register. `me` adalah grup pelanggan,
		// jadi yang diperiksa cukup `admin` dan `g`.
		recv := identName(sel.X)
		if recv != "admin" && recv != "g" {
			return true
		}

		diperiksa++

		// Argumen pertama adalah path.
		if len(call.Args) == 0 {
			t.Errorf("%s: registrasi tanpa argumen", posisi(fset, call))
			return true
		}

		// Pemanggilan ber-handler langsung seperti admin.GET("/2fa", h.Status)
		// punya dua argumen dan sah. Yang dicari adalah argumen terakhir berupa
		// guard(): di situlah handler hilang.
		if isGuardCall(call.Args[len(call.Args)-1]) {
			t.Errorf("%s: route %s berakhir pada guard; handler tidak terpasang sehingga balasan akan kosong",
				posisi(fset, call), recv+"."+sel.Sel.Name)
		}

		return true
	})

	if diperiksa == 0 {
		t.Fatal("tidak ada registrasi route admin yang ditemukan; pemeriksaan tidak berjalan")
	}
	t.Logf("%d registrasi route admin diperiksa", diperiksa)
}

// isGuardCall menandai pemanggilan berbentuk guard(...).
func isGuardCall(expr ast.Expr) bool {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return false
	}
	if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
		return sel.Sel.Name == "guard"
	}
	return identName(call.Fun) == "guard"
}

// identName mengembalikan nama pengenal pada sebuah ekspresi.
func identName(expr ast.Expr) string {
	id, ok := expr.(*ast.Ident)
	if !ok {
		return ""
	}
	return id.Name
}

// posisi memberi lokasi kode untuk pesan kesalahan.
func posisi(fset *token.FileSet, node ast.Node) string {
	p := fset.Position(node.Pos())
	return strings.TrimSpace(p.String())
}
