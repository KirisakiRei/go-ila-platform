package knowledge

import (
	"embed"
	"fmt"
	"strings"
	"sync"
)

//go:embed SOUL.md SKILLS.md MEMORY.md ila2/*.md
var files embed.FS

// ila2Order menjaga urutan baca resmi sesuai ila2/README.md agar aturan yang
// saling merujuk selalu berada pada konteks yang berurutan.
var ila2Order = []string{
	"ila2/README.md",
	"ila2/key-level-irl.md",
	"ila2/key-level-erl.md",
	"ila2/c2-closure-irl.md",
	"ila2/c2-closure-erl.md",
	"ila2/cisd.md",
	"ila2/time-cycle.md",
	"ila2/alur-entry.md",
	"ila2/diagram.md",
}

var (
	loadOnce   sync.Once
	cachedBase string
	cachedErr  error
)

// BaseSystemPrompt mengembalikan gabungan persona nanobot (SOUL, SKILLS, MEMORY)
// dan seluruh dokumen baku New ILA 2.0 dari berkas yang ter-embed di binary.
func BaseSystemPrompt() (string, error) {
	loadOnce.Do(func() {
		var b strings.Builder
		sections := append([]string{"SOUL.md", "SKILLS.md", "MEMORY.md"}, ila2Order...)
		for i, path := range sections {
			raw, err := files.ReadFile(path)
			if err != nil {
				cachedErr = fmt.Errorf("membaca knowledge %s: %w", path, err)
				return
			}
			if i > 0 {
				b.WriteString("\n\n---\n\n")
			}
			b.WriteString("<!-- FILE: ")
			b.WriteString(path)
			b.WriteString(" -->\n")
			b.WriteString(strings.TrimSpace(string(raw)))
		}
		cachedBase = b.String()
	})
	return cachedBase, cachedErr
}
