# ILA Platform

Platform SaaS untuk ILA Bot: backend API, runner, dan console web.

Sistem dipisah menjadi tiga peran:

| Peran | Tanggung jawab | Lokasi |
| --- | --- | --- |
| Engine | Menemukan setup trading (ILA 2.0 dan ILA 3.0) | repo terpisah, VPS 1 |
| Runner | Manajemen trade per pengguna, memegang kredensial broker | `backend/cmd/runner` |
| Console | Dashboard, onboarding akun, tier, notifikasi | `backend/cmd/backend-api` + `frontend` |

Engine tidak mengenal pengguna dan tidak memegang kredensial. Runner tidak
menghitung ulang strategi; ia hanya menjalankan sinyal yang dikirim engine.
Frontend tidak pernah menyentuh broker maupun engine secara langsung.

## Struktur

```
backend/
  cmd/            binary: backend-api, runner, migrate
  internal/       router → handler → service → repository → model
  pkg/            apperror, httpx, utils
frontend/         React + Vite + TypeScript + Tailwind + shadcn/ui
deploy/           Dockerfile dan docker-compose (4 container)
```

Aturan penulisan kode backend ada di [AGENTS.md](AGENTS.md).

## Menjalankan secara lokal

```bash
cp .env.example .env          # isi kredensial lokal
make install-frontend
make migrate
make run-api                  # terminal 1
make dev-frontend             # terminal 2
```

Frontend berjalan di `http://localhost:3000` dan meneruskan `/api` serta
`/health` ke backend di `http://localhost:8080`.

## Pemeriksaan sebelum commit

```bash
make check
```

Menjalankan `go vet`, `go test`, dan `eslint` frontend.

## Deployment

Empat container dijalankan dari `deploy/docker-compose.yml`:
`postgres`, `backend-api`, `runner`, dan `frontend` (nginx).
Total batas memori 512 MB.

```bash
make up      # nyalakan
make logs    # lihat log
make down    # hentikan
```

## Endpoint awal

| Metode | Path | Keterangan |
| --- | --- | --- |
| GET | `/health` | Liveness |
| GET | `/ready` | Kesiapan database |
| POST | `/api/v1/auth/register` | Daftar akun |
| POST | `/api/v1/auth/login` | Masuk, mengembalikan token |
| GET | `/api/v1/me` | Profil pengguna aktif |
