import type { Operation } from './operations'

// Empty by default: requests go to the same origin (/api/...), which is
// proxied by Vite in development and by nginx in Docker.
const API_BASE = import.meta.env.VITE_API_URL ?? ''

export interface CalculateResponse {
  operation: Operation
  a: number
  b?: number
  result: number
}

// Error carrying the backend's machine-readable code, so the UI can
// react to specific cases if needed.
export class ApiError extends Error {
  readonly code: string
  readonly status: number

  constructor(code: string, message: string, status: number) {
    super(message)
    this.name = 'ApiError'
    this.code = code
    this.status = status
  }
}

export async function calculate(
  operation: Operation,
  a: number,
  b?: number,
  signal?: AbortSignal,
): Promise<CalculateResponse> {
  let response: Response
  try {
    response = await fetch(`${API_BASE}/api/v1/calculate`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(b === undefined ? { operation, a } : { operation, a, b }),
      signal,
    })
  } catch (err) {
    if (err instanceof DOMException && err.name === 'AbortError') throw err
    throw new ApiError('NETWORK_ERROR', 'Could not reach the calculator service', 0)
  }

  const body = await response.json().catch(() => null)

  if (!response.ok) {
    const code = body?.error?.code ?? 'UNKNOWN_ERROR'
    const message = body?.error?.message ?? `Request failed with status ${response.status}`
    throw new ApiError(code, message, response.status)
  }
  if (!body || typeof body.result !== 'number') {
    throw new ApiError('INVALID_RESPONSE', 'Unexpected response from the server', response.status)
  }
  return body as CalculateResponse
}
