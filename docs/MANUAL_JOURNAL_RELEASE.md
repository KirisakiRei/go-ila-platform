# Manual journal, position setup charts, and Telegram cycle labels

## Scope

- Manual journal is an owner-scoped, independent module at `/app/jurnal`. It does not create broker positions, orders, approval tokens, signals, or trading quota records.
- Telegram trading alerts and edited callback messages retain setup/cycle context. Missing context is explicitly unknown.
- Position charts display source HTF/LTF with actual position prices. Legacy positions retain automatic-timeframe fallback.

## Deployment prerequisites

1. Back up the PostgreSQL database and configured private upload volume.
2. Run the existing migration command against the target environment **before starting the new backend**: `go run ./cmd/migrate` from `backend`, or the existing deployment migration job.
3. Confirm these new tables exist: `manual_journal_accounts`, `manual_journal_capital`, `manual_journal_entries`, `manual_journal_screenshots`, `manual_journal_import_tokens`, `manual_journal_reviews`, `manual_journal_review_settings`. Use model TableName definitions as authoritative if deploying custom migrations. Review settings require generation columns, including `settings_generation` on reviews.
4. Deploy backend API, runner (Telegram publishers), and frontend through the existing deployment process. No engine strategy change is required.
5. Ensure the configured `UploadDir` volume is writable and persistent. Journal files live in its private `manual-journal` subdirectory; never expose it with static serving.
6. Smoke-test authenticated CRUD, account ownership, screenshot access, calendar/table filtering, CSV preview/confirm, chart fallback, and Telegram edits with test accounts.
7. Review AI uses existing configured AI provider/model. Without configured AI, deterministic review remains available. Verify privacy preferences before live provider testing.

No migration, production deployment, or external-provider smoke test was executed as part of local implementation. Roll back application images if necessary; do not drop the new tables or upload files containing user data.

## Data and calculations

- Calendar dates, filters, review rollover, and displayed times use WIB (UTC+7).
- Draft/open/unknown P&L are not realized results; zero P&L is breakeven.
- Win rate excludes breakeven from its denominator. Profit factor is unavailable without losses.
- Planned RR, realized R, hold duration, and gain percentage require their corresponding inputs. Net P&L is manually supplied, not inferred universally from prices/lot size.
- Financial API values are decimal strings; backend calculations use exact rational arithmetic with explicit output rounding.
- Realized balance is initial capital + deposits - withdrawals + known closed net P&L. Unknown closed P&L makes balance incomplete. It is not broker equity, and correctness depends on complete manual records without double-counting initial capital.
- Performance curve and absolute drawdown use cumulative realized P&L, excluding cashflows. Monetary reports reject mixed currencies.
- Tags/emotions/mistakes/checklists accept one item per line in the UI. Multi-tag breakdown groups overlap.

## Limits

- Reports/insights: at most 10,000 matching entries, with an explicit error rather than truncation.
- CSV import/export: at most 500 rows and 1 MiB. Narrow filters for larger exports. CSV transfers entries, not accounts, cashflows, screenshots, or a full backup.
- Import preview tokens are owner-bound, payload-bound, expire after 15 minutes, and are consumed atomically with insertion. Retry of the same confirmation cannot duplicate rows; a deliberately new preview can import identical content again.
- Historical checklist snapshots survive template changes and CSV roundtrips.
- Deleting an entry transactionally tombstones its screenshot metadata. File cleanup retries in a separate background task; metadata is only purged after file deletion succeeds.
- Chart requests currently use the latest 500 candles; no historical-range endpoint or archive was added. Missing entry history and source-feed differences are explicitly labeled.
- Position chart mobile uses HTF/LTF tabs; desktop uses two panels.

## Automatic review

- Persistent per-account/day/revision cache, background polling, bounded provider attempts, leases, and stale-result checks.
- Save/update changes are detected by fingerprint; refresh is eventually consistent with the worker interval (normally one minute).
- Review uses cumulative history through the current WIB date, not a previous-day-only recap. Frontend pending reviews refresh every five seconds, ready reviews every minute.
- Provider input is aggregate numeric metrics only; free-text notes, labels, chart images, and raw journals are not sent.
- AI requires explicit user-wide opt-in (default off); per-account preference can further disable it. Rules continue without consent or a configured provider. Generation checks prevent stale work from reviving withdrawn permission. A provider request already in flight cannot be recalled.
- Recommendations never change broker risk settings or place orders.
- Worker scan cursor resumes between ticks in a process; it resets on restart. Large-history fingerprint scans and report limits need monitoring as data grows.

## Validation and remaining checks

Backend tests, build, and vet pass. Frontend tests and production build pass. Scope-specific lint passes; full-project lint has six existing failures in `TradingRadarChart.tsx`, `PengaturanPairModal.tsx`, and `ila2-progress.ts`. The existing conditional-hook error in TradingRadarChart is outside the timeframe-prop changes.

Still required before production acceptance: PostgreSQL migration/concurrency verification, real browser/mobile/keyboard checks, authenticated E2E, Telegram delivery, and optional live AI testing. The frontend build retains its existing large-chunk warning.
