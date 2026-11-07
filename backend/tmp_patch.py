# Skrip bantu sekali pakai.
import io
import sys

path = 'cmd/runner/main.go'
src = io.open(path, encoding='utf-8', errors='replace').read()

start = src.index('// runEngineLoop menjaga koneksi ke engine tetap hidup.')
mid = src.index('// streamOnce membuka satu sesi ke engine sampai terputus.')

new_loop = '''// engineTarget adalah satu proses engine yang harus dijangkau Runner.
type engineTarget struct {
\tStrategy           string
\tAddr               string
\tCertFile           string
\tKeyFile            string
\tCAFile             string
\tServerNameOverride string
}

// engineTargets menyusun daftar engine yang aktif.
// ILA 3.0 hanya ikut bila tahapnya mengizinkan engine berjalan.
func engineTargets(cfg *config.Config) []engineTarget {
\tout := make([]engineTarget, 0, 2)

\tif cfg.EngineGRPCAddr != "" {
\t\tout = append(out, engineTarget{
\t\t\tStrategy:           cfg.EngineStrategy,
\t\t\tAddr:               cfg.EngineGRPCAddr,
\t\t\tCertFile:           cfg.EngineMTLSCert,
\t\t\tKeyFile:            cfg.EngineMTLSKey,
\t\t\tCAFile:             cfg.EngineMTLSCA,
\t\t\tServerNameOverride: cfg.EngineServerName,
\t\t})
\t}

\tila3 := runner.NewSettings(cfg.ILA3Availability)
\tif cfg.Engine3GRPCAddr != "" && ila3.ILA3Running() {
\t\tout = append(out, engineTarget{
\t\t\tStrategy:           "ila-3.0",
\t\t\tAddr:               cfg.Engine3GRPCAddr,
\t\t\tCertFile:           cfg.Engine3MTLSCert,
\t\t\tKeyFile:            cfg.Engine3MTLSKey,
\t\t\tCAFile:             cfg.Engine3MTLSCA,
\t\t\tServerNameOverride: cfg.Engine3ServerName,
\t\t})
\t}

\treturn out
}

// runEngineLoop menjaga koneksi ke satu engine tetap hidup.
//
// Pola reconnect mengikuti plans/05 §8: setelah terhubung, minta snapshot,
// lalu berlangganan ulang untuk semua simbol kebutuhan. Sinyal lama yang
// terlewat TIDAK dieksekusi karena pemeriksaan TTL berjalan di jalur sinyal.
func runEngineLoop(
\tctx context.Context,
\ttarget engineTarget,
\tsvc *runner.Service,
\tstore repository.RunnerRepository,
\tlogger *slog.Logger,
) {
\tbackoff := minBackoff
\tstateID := engineStateID(target.Strategy)

\tfor ctx.Err() == nil {
\t\terr := streamOnce(ctx, target, svc, store, logger)
\t\trecordEngineState(ctx, store, stateID, target.Strategy, err, logger)

\t\tif ctx.Err() != nil {
\t\t\treturn
\t\t}

\t\tlogger.Warn("koneksi engine terputus, mencoba ulang",
\t\t\t"strategy", target.Strategy,
\t\t\t"backoff", backoff.String(), "error", errorText(err))

\t\tselect {
\t\tcase <-ctx.Done():
\t\t\treturn
\t\tcase <-time.After(backoff):
\t\t}

\t\tbackoff *= 2
\t\tif backoff > maxBackoff {
\t\t\tbackoff = maxBackoff
\t\t}
\t}
}

'''

src = src[:start] + new_loop + src[mid:]
io.open(path, 'w', encoding='utf-8', errors='replace').write(src)
print('ok')
