# Full-Stack Calculator

A calculator with a **React + TypeScript** frontend and a **Go** REST microservice.
The frontend sends every operation to the backend, which validates the input,
computes the result and returns JSON.

- Operations: addition, subtraction, multiplication, division, exponentiation, square root and percentage
- Input validation on both layers, with clear, typed error codes
- Unit tests for both layers (backend: 100% of the domain package, ~99% of the HTTP layer; frontend: 100% of statements)
- Docker Compose setup that runs the whole stack with one command

```
.
├── backend/             Go service (standard library only, no external deps)
│   ├── calculator/      Pure arithmetic logic + table-driven tests
│   ├── api/             HTTP handlers, JSON validation, CORS and logging middleware + tests
│   ├── main.go          Server wiring, config from env vars, graceful shutdown
│   └── Dockerfile
├── frontend/            React 19 + TypeScript + Vite
│   ├── src/
│   │   ├── Calculator.tsx   UI component (form, result, history)
│   │   ├── api.ts           Typed API client
│   │   ├── validation.ts    Client-side input parsing and validation
│   │   ├── operations.ts    Operation metadata shared by UI and client
│   │   └── *.test.ts(x)     Vitest + Testing Library tests
│   ├── nginx.conf           Serves the bundle and proxies /api in Docker
│   └── Dockerfile
├── docker-compose.yml
├── Makefile
└── PROMPTS.md           AI prompts used while building this
```

---

## Running it

### Option 1: Docker (whole stack)

Requirements: Docker with Compose.

```bash
docker compose up --build
```

Open http://localhost:3000. nginx serves the frontend and forwards `/api/*` to the Go service.

### Option 2: Local development

Requirements: Go 1.22+ and Node.js 20+.

**Backend** (http://localhost:8080):

```bash
cd backend
go run .
```

**Frontend** (http://localhost:5173), in a second terminal:

```bash
cd frontend
npm install
npm run dev
```

Vite proxies `/api` to `localhost:8080`, so the browser makes same-origin requests.

### Configuration

| Variable      | Where    | Default                 | Description |
|---------------|----------|-------------------------|-------------|
| `PORT`        | backend  | `8080`                  | HTTP port |
| `CORS_ORIGIN` | backend  | `http://localhost:5173` | Allowed browser origin. Empty disables CORS headers. |
| `VITE_API_URL`| frontend | `""` (same origin)      | Base URL of the API, if not proxied |

---

## Tests and coverage

```bash
make test        # backend (with -race) + frontend
make coverage    # coverage reports for both layers
```

Or per layer:

```bash
# Backend
cd backend
go test ./... -race
go test ./calculator ./api -coverprofile=coverage.out
go tool cover -func=coverage.out          # summary in the terminal
go tool cover -html=coverage.out -o coverage.html

# Frontend
cd frontend
npm test
npm run coverage                           # text summary + HTML in frontend/coverage/
```

Coverage at the time of submission:

| Layer | Package / file | Statements |
|-------|----------------|-----------|
| Backend | `calculator` | 100% |
| Backend | `api` | 98.7% |
| Frontend | all source files | 100% (94% branches) |

`main.go` is intentionally not unit tested: it only wires configuration and the
server. Its behavior is covered by the handler tests plus the Docker smoke test.

---

## API

Base path: `/api/v1`. All responses are JSON.

### `POST /api/v1/calculate`

Request:

| Field       | Type   | Required | Notes |
|-------------|--------|----------|-------|
| `operation` | string | yes      | `add`, `subtract`, `multiply`, `divide`, `power`, `sqrt`, `percentage` (case-insensitive) |
| `a`         | number | yes      | First operand |
| `b`         | number | for binary operations | Ignored for `sqrt` |

`percentage` returns `b` percent of `a` (`a = 200, b = 15` → `30`).

**Examples**

```bash
curl -s -X POST http://localhost:8080/api/v1/calculate \
  -H 'Content-Type: application/json' \
  -d '{"operation":"divide","a":10,"b":4}'
# 200 {"operation":"divide","a":10,"b":4,"result":2.5}

curl -s -X POST http://localhost:8080/api/v1/calculate \
  -H 'Content-Type: application/json' \
  -d '{"operation":"sqrt","a":16}'
# 200 {"operation":"sqrt","a":16,"result":4}

curl -s -X POST http://localhost:8080/api/v1/calculate \
  -H 'Content-Type: application/json' \
  -d '{"operation":"divide","a":1,"b":0}'
# 422 {"error":{"code":"DIVISION_BY_ZERO","message":"division by zero"}}
```

### `GET /api/v1/operations`

Lists supported operations and how many operands each needs.

```bash
curl -s http://localhost:8080/api/v1/operations
# {"operations":[{"name":"add","operands":2}, ... ,{"name":"sqrt","operands":1}, ...]}
```

### `GET /healthz`

Liveness check: `{"status":"ok"}`.

### Errors

Every error has the same shape:

```json
{ "error": { "code": "DIVISION_BY_ZERO", "message": "division by zero" } }
```

| HTTP | Code | When |
|------|------|------|
| 400 | `INVALID_JSON` | Body is empty, malformed, has unknown fields or trailing data |
| 400 | `INVALID_INPUT` | Missing operation or operand, or an operand that is not a number |
| 400 | `UNSUPPORTED_OPERATION` | Operation name not recognized |
| 415 | `UNSUPPORTED_MEDIA_TYPE` | `Content-Type` is not `application/json` |
| 413 | `PAYLOAD_TOO_LARGE` | Body larger than 4 KiB |
| 422 | `DIVISION_BY_ZERO` | `b` is 0 in a division |
| 422 | `NEGATIVE_SQRT` | Square root of a negative number |
| 422 | `NON_FINITE_RESULT` | Overflow (e.g. `10^400`) or undefined result (e.g. `(-8)^0.5`) |
| 405 | — | Wrong HTTP method (handled by the Go router) |

---

## Design decisions

**One endpoint with an `operation` field, not one endpoint per operation.**
The operations share the same request shape, validation and error handling.
A single endpoint keeps the API surface small, and adding an operation only
touches the calculator package. `GET /operations` lets clients discover what is
available instead of hardcoding it.

**Domain logic separated from HTTP.** `calculator` is pure Go with no HTTP or
JSON imports. It returns sentinel errors (`ErrDivisionByZero`, ...) that the
`api` package maps to status codes. This keeps the math trivially testable and
reusable from another transport.

**400 vs 422.** Malformed or incomplete requests are client errors (400). A
well-formed request that is mathematically undefined, like dividing by zero, is
422 Unprocessable Entity. Clients can tell "you sent bad data" apart from
"this calculation has no answer".

**Operands are `*float64`.** A pointer distinguishes a missing field from an
explicit `0`, so `{"operation":"add","a":0,"b":5}` is valid while
`{"operation":"add","b":5}` is rejected.

**No `Inf` or `NaN` in responses.** JSON cannot represent them and they are
rarely what a user expects. The service checks every result and returns
`NON_FINITE_RESULT` on overflow or undefined values.

**`float64` precision.** IEEE-754 doubles are the natural choice for a general
calculator and match JavaScript's `number`. The trade-off is binary rounding
(`0.1 + 0.2 = 0.30000000000000004`). The backend returns the exact double and the
UI formats it to 12 significant digits. A financial calculator would use a
decimal type instead; that is out of scope here.

**Strict JSON decoding.** Unknown fields, trailing data, wrong types and bodies
over 4 KiB are rejected. It catches client bugs early and limits abuse.

**Validation on both layers.** The frontend validates for instant feedback
(empty fields, non-numbers, division by zero) without a round trip. The backend
validates everything again because it is the source of truth and can be called
by any client.

**Standard library only.** Go 1.22's `http.ServeMux` supports method-based
routes (`POST /api/v1/calculate`) and answers 405 automatically, so no router
dependency was needed. `log/slog` provides structured logs.

**Production basics.** Server timeouts, graceful shutdown on SIGTERM, a
health endpoint, configuration through environment variables, a distroless
non-root container and request logging.

**Frontend structure.** The API client, validation and operation metadata are
plain TypeScript modules, so they are tested without rendering. The component
handles UI state only. Requests use an `AbortController` so a newer calculation
cancels a slower previous one. The UI is accessible: labelled inputs,
`aria-invalid`, `aria-live` results and keyboard-friendly controls.

**Same-origin in every environment.** Vite (dev) and nginx (Docker) proxy
`/api`, so the browser never makes cross-origin requests. CORS is still
available on the backend for clients served from other origins.

## Assumptions

- Operands and results are within `float64` range; anything beyond is reported as an error.
- `percentage` means "b percent of a".
- History is per browser session only; there is no persistence.

## What I would add next

- Rate limiting and request IDs for tracing
- OpenAPI spec generated from the handlers
- End-to-end tests (Playwright) against the Docker stack
- CI pipeline running lint, tests and coverage on each push
