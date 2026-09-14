# Disaster Watcher: Project Map and Code Review

Last reviewed: 2026-09-09

## Product in one paragraph

Disaster Watcher is a Go HTTP API for community disaster reporting. A user registers with a location, signs in, and submits a report. The service geocodes locations, stores the report and a durable job in PostgreSQL, verifies the report, finds users near it, sends notifications, retries failures with exponential backoff, and records exhausted work in a dead-letter queue (DLQ).

The repository currently contains the backend only. There is no web or mobile client here.

## Current architecture

```text
HTTP request
  -> Gin router
  -> JWT authentication
  -> report + job persisted in PostgreSQL
  -> verification channel (5 workers)
  -> report channel (5 workers)
  -> affected-user channel (5 notification workers)
  -> failed-email channel (5 retry workers)
  -> dead-letter channel + PostgreSQL DLQ
```

PostgreSQL is both the application database and intended durable queue. In-memory buffered Go channels connect worker stages. GORM handles persistence. Nominatim supplies forward and reverse geocoding. The email implementation is currently a random success/failure simulator; the SMTP implementation is commented out.

## Implemented feature inventory

### Accounts and security

- User registration with unique username/email fields.
- Password hashing with bcrypt.
- Login with a 24-hour HS256 JWT.
- JWT-protected API route group.
- Admin role and middleware code exists, but no admin route is wired.
- Per-IP token-bucket HTTP rate limiting.
- User locations are geocoded and cached at registration.

### Reports and discovery

- Authenticated creation of disaster reports.
- Report categories, priorities, status, title, description, location, and votes are modeled.
- Reports are geocoded and their coordinates cached.
- A user can list their own reports.
- A report can be retrieved by ID.
- Nearby/recent report lookup is present, intended to filter by age and distance.
- Additional list/delete/coordinate-search handlers exist but are not routed.

### Notification pipeline

- A PostgreSQL job is created for each report.
- Four goroutine worker-pool stages: verification, user extraction, notification, and retry.
- Buffered typed channels connect the stages.
- Global outbound-email token-bucket rate limiting.
- Exponential retry delays and configurable maximum retry count.
- DLQ model and persistence for exhausted notification retries.
- An idempotency-key model is partially implemented for user/job notification pairs.
- SIGINT/SIGTERM HTTP shutdown and worker cancellation are implemented.
- Pending/stalled-job recovery helpers exist, but are not active in startup.

### Operations and development

- Dockerfile and Docker Compose configuration for API + PostgreSQL.
- Structured logging is started with `slog`, though several other logging styles remain.
- A Prometheus proof of concept exists but is not started or routed.
- Unit/integration tests cover Haversine-related geocoding helpers and pending-job recovery.
- Make targets provide manual registration/login and load generation.

## Routes actually exposed

| Method | Path | Authentication | Purpose |
|---|---|---:|---|
| POST | `/user/register` | No | Create a user and cache their coordinates |
| POST | `/user/login` | No | Return a JWT |
| GET | `/api/user/:id` | No | Return public account fields for any user ID |
| POST | `/api/reports` | Yes | Create a report and enqueue verification |
| GET | `/api/reports` | Yes | List the authenticated user's reports |
| GET | `/api/get_current_user` | Yes | Intended profile endpoint; currently returns no useful user value |
| GET | `/api/report/:id` | Yes | Fetch any report by ID |
| GET | `/api/nearby_reports/:hours` | Yes | Intended nearby/recent report lookup; route and handler disagree about the hours input |

The README currently documents different paths and query conventions, so it should not yet be treated as an API contract.

## Critical correctness findings

These should be fixed before extending the product.

1. **Nearby notification logic is reversed.** `GetUsersAffectedByDisaster` enqueues users when distance is greater than 20 km, not less than or equal to 20 km. It also omits the report from `AffectedUsersMessage`, so downstream emails receive an empty report.
2. **Recent-nearby lookup cannot reliably work.** The route supplies `:hours`, while the handler reads `?hours=`. It queries a user with `useruser_id = ?` instead of the users table primary key `id`, causing a database error in a normal schema.
3. **Profile context is never set.** JWT middleware sets `userId` and `currentUserEmail`; the profile handler reads `currentUser`, so the endpoint returns `null`.
4. **Registration continues after malformed JSON.** The binding-error branch sends a 400 response but does not return, allowing the handler to continue with zero-value input.
5. **Report creation is not atomic.** A report may remain stored when geocoding, payload serialization, job creation, or enqueueing fails. A queue-full response leaves a pending job in PostgreSQL without an active recovery loop.
6. **Durable recovery is disabled.** Startup recovery calls are commented out, and `RecoverMidProcessingJobs` accidentally queries `pending`, not `processing`. Stalled recovery is never run.
7. **A job's status does not represent all deliveries.** The first successful recipient marks the entire report job `done`; one exhausted recipient marks it `failed`, even if other recipients are still running or succeeded.
8. **Idempotency is recorded incorrectly.** Initial delivery creates a `success` record even when sending failed. The model is not included in `AutoMigrate`, database errors are mostly ignored, and checking happens only during retry, after an initial send.
9. **Email is not real.** `SendEmail` randomly succeeds or fails. The README currently describes notification as if it were operational.
10. **The verification algorithm is a stub.** `Helper` always returns `Verified`; `UserTrustScore` always returns `-1`. This means every geocoded report proceeds regardless of content or trust.
11. **Worker snapshots go stale.** Users are loaded once at process startup. Anyone who registers afterward cannot be notified until restart.
12. **Shutdown can lose queued work.** Cancellation makes workers exit without draining all accepted channel messages; five seconds may end HTTP work while downstream persistence/status transitions remain incomplete.

## Security and reliability findings

- JWT middleware type-asserts the `exp` claim and can panic on a malformed but parseable token.
- The signing secret is not validated at startup; an empty `SECRET` still signs and verifies tokens.
- API responses expose internal database/network error strings.
- Registration checks only duplicate usernames before insert and does not validate required fields, email format, password strength, category, priority, location coordinates, or allowed status transitions.
- Report retrieval is authenticated but not ownership-checked; deletion code, if routed, has the same authorization concern.
- Rate limiting trusts client IP configuration and keeps visitor entries forever. Its cleanup method has locking and ticker-lifecycle defects and is not started.
- Geocoding uses an HTTP client with no timeout, no context propagation, no retry policy, and no cache; one external call can hold a request indefinitely.
- Database calls generally do not receive request/worker contexts.
- Several nil coordinate pointers can be dereferenced in unrouted or edge-case paths.
- Database schema management is split between an incomplete SQL file and `AutoMigrate`; neither is a dependable versioned migration story.
- The global `db.DB` makes isolation and tests difficult.

## Maintainability and repository findings

- Package boundaries are blurred: handlers, global database state, utilities, worker orchestration, and business rules depend directly on one another.
- Naming and JSON conventions are inconsistent (`UserId`, `Created_at`, `Longitude`, `Description`, `co-ordinates`).
- API response shapes and status-code choices are inconsistent.
- Comments frequently describe aspirations rather than current guarantees.
- Dead code and experiments remain (Redis helper, unused handlers, admin shell, Prometheus server, commented SMTP block, duplicate READMEs).
- Generated binaries, Redis data, build logs, and oddly named scratch files are tracked or present in the repository.
- The Makefile contains a committed bearer token and personal test payloads. Even if expired, secrets/tokens should never be committed; rotate the signing secret if that token ever targeted a real deployment.
- The test suite has hard-coded `/home/denzil/.../.env`, requires a live PostgreSQL database, mutates persistent data without cleanup, and makes real Nominatim requests.
- There are no handler, authentication, worker, retry, concurrency, migration, or end-to-end tests.

## Test baseline at review time

`go test -vet=off ./... -count=1 -timeout=90s` passes when given access to the configured local PostgreSQL instance and external geocoding service. Only `internal/utils` contains tests. A sandboxed/offline run fails during `TestMain` because the whole utility package unconditionally connects to that database.

Passing tests therefore mean "the few environment-coupled tests passed," not "the service is correct."

## Recommended renovation sequence

### Phase 0: establish a safe baseline

- Preserve or commit the current working tree intentionally; it contains substantial uncommitted changes.
- Remove tracked build/data/scratch artifacts and replace the committed token in the Makefile.
- Add `.env.example`, strict startup config validation, formatting/static checks, and a reproducible test command.
- Freeze and document the API contract before changing internals.

### Phase 1: repair the existing product

- Fix proximity, missing report propagation, nearby-route/query, user lookup, profile context, and registration early return.
- Make report + job creation transactional and ensure every persisted pending job is eventually claimed.
- Define job semantics explicitly: one report-processing job plus one delivery row per recipient is the cleanest model.
- Make delivery idempotency database-enforced and update statuses transactionally.
- Activate and test pending/stalled recovery with atomic job claiming.
- Introduce a real email adapter behind an interface, retaining a deterministic fake for tests.

### Phase 2: create clean architecture seams

- Add a typed config package and an `App` composition root.
- Replace the global database with repository interfaces and injected dependencies.
- Separate HTTP DTOs, domain models, and persistence records.
- Move report validation/verification and proximity policy into domain services.
- Wrap geocoding and email in context-aware interfaces with timeouts.
- Standardize errors, responses, logging fields, and naming.

### Phase 3: prove reliability

- Fast table-driven unit tests for validation, Haversine, verification, retry schedules, and state transitions.
- Handler tests using `httptest` and mocked services.
- PostgreSQL integration tests in isolated transactions or disposable containers.
- Deterministic worker tests for duplicate delivery, restart recovery, partial recipient failure, shutdown, and queue saturation.
- Run `go test -race ./...`, `go vet ./...`, a linter, migration checks, and container smoke tests in CI.

### Phase 4: production polish

- Health/readiness endpoints, real Prometheus metrics, structured request/job correlation IDs, and useful dashboards.
- Versioned migrations, database indexes, retention policies, and DLQ admin/replay workflows.
- Pagination and geospatial database queries instead of loading all users/reports into memory.
- Notification preferences, report confirmation/voting rules, moderation tools, and abuse controls.
- OpenAPI documentation and generated examples kept in CI against actual routes.

## Definition of “fine art” for this project

The goal is not the most abstractions. It is a small service whose guarantees are obvious: accepted reports are durably processed, eligible users receive at most one notification, failures are visible and replayable, cancellation is safe, API behavior is documented, and each rule can be tested without the internet or a developer's personal database.
