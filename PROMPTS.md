# AI usage

I used Claude as a pair programmer for this assignment, as allowed by the
instructions. I reviewed, ran and adjusted every file, and I can explain each
design decision (see the README).

## Prompts

1. > Here is the take-home assignment from Sezzle: [pasted the full assignment].
   > Propose an architecture that is simple and easy to defend in an interview:
   > Go backend using only the standard library, React + TypeScript frontend,
   > unit tests on both layers, Docker Compose and a README with design decisions.

2. > Build the Go backend: a pure `calculator` package with sentinel errors and
   > table-driven tests (division by zero, negative square root, overflow, NaN,
   > missing operands), and an `api` package with a single
   > `POST /api/v1/calculate` endpoint, strict JSON decoding, consistent error
   > envelope, CORS and logging middleware, and graceful shutdown.

3. > Build the React frontend with Vite and TypeScript: typed API client,
   > client-side validation, operation selector, result panel and short history.
   > Make it responsive and accessible. Add Vitest + Testing Library tests for
   > the client, the validation and the component, including loading and error states.

4. > Add Dockerfiles (multi-stage Go to distroless, Vite build served by nginx
   > that proxies /api) and a docker-compose.yml.

5. > Write the README: setup, how to run each layer, curl examples, error codes,
   > coverage, design decisions, assumptions and next steps.

6. > Walk me through each file so I can explain every line, and point out
   > anything you would change. Then apply the changes I agreed with.

## What I changed or decided myself

- Ran the backend and frontend locally on Windows 11 (Go 1.27, Node 24),
  exercised the API with curl and checked the full stack with
  `docker compose up` instead of only trusting the generated setup. `make` is not available on Windows, so I added
  PowerShell notes to the README (`npm test -- --run`, `curl.exe`).
- The 415 response reused the `INVALID_JSON` code even though the body was fine
  and the problem was the Content-Type. I gave it its own code,
  `UNSUPPORTED_MEDIA_TYPE`, and updated the test and the README table.
- `SupportedOperations()` repeats the operations map by hand, so adding an
  operation to the map and forgetting the list would silently break
  `GET /operations`. I added a test that fails if the two drift apart.
- Rewrote a confusing doc comment on `Calculate` about when `b` is required.
- Kept `float64` on purpose: it matches JavaScript's `number` and is enough for a
  general calculator. For money I would use a decimal type (noted in the README).
