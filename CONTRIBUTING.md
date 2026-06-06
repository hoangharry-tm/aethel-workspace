# Contributing to Aethel Workspace

Thank you for your interest in contributing. This document covers everything you need to get started.

---

## 1. Development Setup

**Prerequisites:**
- Go 1.24+
- Node.js 22+
- pnpm 9+
- PostgreSQL 16+
- Docker 24+ and Docker Compose v2

**Steps:**

```bash
git clone https://github.com/hoangharry-tm/aethel-workspace.git
cd aethel-workspace

# Configure environment
cp .env.example .env
# Edit .env: set POSTGRES_PASSWORD and AETHEL_JWT_SECRET

# First-time setup
bash aethel-scripts/setup-dev.sh

# Start all services (PostgreSQL + Go backend + Nuxt frontend)
make dev
```

The frontend is available at `http://localhost:3000`, the backend at `http://localhost:8080`, and API docs at `http://localhost:8080/api/docs`.

---

## 2. Branch Naming

| Type | Pattern | Example |
|------|---------|---------|
| Feature | `feat/short-description` | `feat/bulk-dispatch-import` |
| Bug fix | `fix/short-description` | `fix/csrf-cookie-samesite` |
| Chore | `chore/short-description` | `chore/upgrade-chi-v5` |
| Docs | `docs/short-description` | `docs/it-customization-guide` |

- Branch from `dev`, open PRs targeting `dev`
- `main` receives only release merges from `dev` — do not open feature PRs against `main`

---

## 3. Commit Message Format

We follow [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/).

```
<type>(<scope>): <short summary>

[optional body]

[optional footer]
```

**Types:** `feat`, `fix`, `chore`, `docs`, `test`, `refactor`, `perf`, `ci`

**Examples:**
```
feat(dispatch): add priority-based routing rule engine
fix(auth): remove hardcoded JWT fallback secret
chore(deps): upgrade chi to v5.2.1
test(workflow): add hash chain tamper detection benchmark
```

**Breaking changes** — add a `BREAKING CHANGE:` footer:
```
feat(config): replace server-database.yaml schema

BREAKING CHANGE: The `database.host` field is now `database.connection.host`.
Existing blueprints must be updated before upgrading.
```

---

## 4. Pull Request Requirements

Before opening a PR, verify:

- [ ] `go test ./...` passes (backend)
- [ ] `pnpm test` passes (frontend unit + component tests)
- [ ] `go vet ./...` passes with zero warnings
- [ ] `pnpm exec vue-tsc --noEmit` passes with zero TypeScript errors
- [ ] `go test -race -short ./...` passes with zero data races
- [ ] CI is green on your branch
- [ ] PR description references a GitHub issue (`Closes #N` or `Related to #N`)
- [ ] At least one reviewer has approved

Keep PRs focused — one logical change per PR. If your feature requires infrastructure changes, split them into separate PRs.

---

## 5. Code Style

**Go:**
- `gofmt` is enforced by CI — run `gofmt -w .` before pushing
- `golangci-lint run` for additional checks (configured in `.golangci.yml`)
- No commented-out code blocks — remove or file an issue instead
- No inline SQL in `repos/` — all queries go through the `QueryRegistry` (`r.q.Get("group.name").Stmt`)

**Vue / TypeScript:**
- ESLint via `pnpm lint` (`eslint.config.mjs` extends Nuxt defaults)
- No `console.log` in production code paths
- Use semantic CSS tokens (`text-body`, `text-muted`, `text-accent`) — never reference Tailwind palette names (`text-slate-500`, `text-indigo-600`) directly in components

**General:**
- No `TODO` comments without a linked GitHub issue
- Error messages are lowercase with no trailing period (Go convention)

---

## 6. Testing Expectations

**Unit tests** (`internal/service/`)
- Use interface-based in-memory mocks (see `auth_service_test.go` for the pattern)
- No external mock libraries (no `go-sqlmock`, no `testify/mock`)
- No `t.Parallel()` — tests are serial by convention for predictability

**Integration tests** (`internal/integration/`)
- Must hit a real PostgreSQL instance — no mocked DB
- Use `//go:build integration` build tag
- Run with: `go test -tags integration ./integration/... -db-url="$DATABASE_URL"`

**Benchmark tests**
- Use `b.ResetTimer()` after setup
- Use `b.ReportAllocs()` for allocation-sensitive benchmarks
- DB benchmarks use `//go:build integration` and skip gracefully when `DATABASE_URL` is unset

New API endpoints require at least one integration test covering the happy path and a 4xx error case.

---

## 7. License

By submitting a pull request, you confirm that:
- Your contribution is your own original work, or you have the right to submit it
- You agree to license your contribution under the [Apache License 2.0](./LICENSE)

---

## Questions?

Open a [GitHub Discussion](https://github.com/hoangharry-tm/aethel-workspace/discussions) or email tonminhhoang.work@gmail.com.
