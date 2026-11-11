import { afterEach, describe, expect, it, vi } from 'vitest'
import { ApiError, calculate } from './api'

function mockFetch(impl: () => Promise<Partial<Response>>) {
  const fn = vi.fn(impl)
  vi.stubGlobal('fetch', fn)
  return fn
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('calculate', () => {
  it('posts the operation and returns the parsed result', async () => {
    const fetchMock = mockFetch(async () => ({
      ok: true,
      status: 200,
      json: async () => ({ operation: 'add', a: 2, b: 3, result: 5 }),
    }))

    const res = await calculate('add', 2, 3)

    expect(res.result).toBe(5)
    const [url, init] = fetchMock.mock.calls[0] as unknown as [string, RequestInit]
    expect(url).toBe('/api/v1/calculate')
    expect(init.method).toBe('POST')
    expect(JSON.parse(init.body as string)).toEqual({ operation: 'add', a: 2, b: 3 })
  })

  it('omits b for unary operations', async () => {
    const fetchMock = mockFetch(async () => ({
      ok: true,
      status: 200,
      json: async () => ({ operation: 'sqrt', a: 9, result: 3 }),
    }))

    await calculate('sqrt', 9)

    const [, init] = fetchMock.mock.calls[0] as unknown as [string, RequestInit]
    expect(JSON.parse(init.body as string)).toEqual({ operation: 'sqrt', a: 9 })
  })

  it('throws ApiError with the backend code and message', async () => {
    mockFetch(async () => ({
      ok: false,
      status: 422,
      json: async () => ({ error: { code: 'DIVISION_BY_ZERO', message: 'division by zero' } }),
    }))

    await expect(calculate('divide', 1, 0)).rejects.toMatchObject({
      name: 'ApiError',
      code: 'DIVISION_BY_ZERO',
      message: 'division by zero',
      status: 422,
    })
  })

  it('handles non-JSON error responses', async () => {
    mockFetch(async () => ({
      ok: false,
      status: 502,
      json: async () => {
        throw new SyntaxError('bad json')
      },
    }))

    await expect(calculate('add', 1, 1)).rejects.toMatchObject({ code: 'UNKNOWN_ERROR', status: 502 })
  })

  it('reports network failures as NETWORK_ERROR', async () => {
    mockFetch(async () => {
      throw new TypeError('Failed to fetch')
    })

    const err = await calculate('add', 1, 1).catch((e) => e)
    expect(err).toBeInstanceOf(ApiError)
    expect(err.code).toBe('NETWORK_ERROR')
  })

  it('rejects responses without a numeric result', async () => {
    mockFetch(async () => ({ ok: true, status: 200, json: async () => ({ result: 'oops' }) }))

    await expect(calculate('add', 1, 1)).rejects.toMatchObject({ code: 'INVALID_RESPONSE' })
  })

  it('rethrows aborts so callers can ignore them', async () => {
    mockFetch(async () => {
      throw new DOMException('aborted', 'AbortError')
    })

    await expect(calculate('add', 1, 1)).rejects.toMatchObject({ name: 'AbortError' })
  })
})
