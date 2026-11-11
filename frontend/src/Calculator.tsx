import { useRef, useState, type FormEvent } from 'react'
import { ApiError, calculate } from './api'
import { OPERATIONS, formatExpression, getOperation, type Operation } from './operations'
import { formatResult, validateInputs } from './validation'

interface HistoryEntry {
  id: number
  expression: string
  result: number
}

const MAX_HISTORY = 5

export default function Calculator() {
  const [operation, setOperation] = useState<Operation>('add')
  const [rawA, setRawA] = useState('')
  const [rawB, setRawB] = useState('')
  const [fieldErrors, setFieldErrors] = useState<{ a?: string; b?: string }>({})
  const [result, setResult] = useState<{ expression: string; value: number } | null>(null)
  const [serverError, setServerError] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)
  const [history, setHistory] = useState<HistoryEntry[]>([])
  const nextId = useRef(1)
  const inFlight = useRef<AbortController | null>(null)

  const unary = getOperation(operation).unary

  function selectOperation(op: Operation) {
    setOperation(op)
    setFieldErrors({})
    setServerError(null)
  }

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    setServerError(null)

    const { a, b, errors } = validateInputs(operation, rawA, rawB)
    setFieldErrors(errors)
    if (errors.a || errors.b || a === undefined) return

    // Cancel a previous request if the user submits again quickly.
    inFlight.current?.abort()
    const controller = new AbortController()
    inFlight.current = controller

    setLoading(true)
    try {
      const res = await calculate(operation, a, unary ? undefined : b, controller.signal)
      const expression = formatExpression(operation, a, b)
      setResult({ expression, value: res.result })
      setHistory((h) => [{ id: nextId.current++, expression, result: res.result }, ...h].slice(0, MAX_HISTORY))
    } catch (err) {
      if (err instanceof DOMException && err.name === 'AbortError') return
      setResult(null)
      setServerError(err instanceof ApiError ? err.message : 'Something went wrong')
    } finally {
      if (inFlight.current === controller) {
        inFlight.current = null
        setLoading(false)
      }
    }
  }

  function handleClear() {
    setRawA('')
    setRawB('')
    setFieldErrors({})
    setResult(null)
    setServerError(null)
  }

  return (
    <section className="calculator" aria-labelledby="calc-title">
      <h1 id="calc-title">Calculator</h1>

      <form onSubmit={handleSubmit} noValidate>
        <fieldset className="operations">
          <legend>Operation</legend>
          {OPERATIONS.map((op) => (
            <button
              key={op.value}
              type="button"
              className={op.value === operation ? 'op active' : 'op'}
              aria-pressed={op.value === operation}
              aria-label={op.label}
              title={op.label}
              onClick={() => selectOperation(op.value)}
            >
              {op.symbol}
            </button>
          ))}
        </fieldset>

        <div className="inputs">
          <div className="field">
            <label htmlFor="operand-a">{unary ? 'Number' : 'First number'}</label>
            <input
              id="operand-a"
              type="text"
              inputMode="decimal"
              value={rawA}
              onChange={(e) => setRawA(e.target.value)}
              aria-invalid={Boolean(fieldErrors.a)}
              aria-describedby={fieldErrors.a ? 'error-a' : undefined}
              placeholder="0"
            />
            {fieldErrors.a && (
              <small id="error-a" className="field-error">
                {fieldErrors.a}
              </small>
            )}
          </div>

          {!unary && (
            <div className="field">
              <label htmlFor="operand-b">{operation === 'percentage' ? 'Percent' : 'Second number'}</label>
              <input
                id="operand-b"
                type="text"
                inputMode="decimal"
                value={rawB}
                onChange={(e) => setRawB(e.target.value)}
                aria-invalid={Boolean(fieldErrors.b)}
                aria-describedby={fieldErrors.b ? 'error-b' : undefined}
                placeholder="0"
              />
              {fieldErrors.b && (
                <small id="error-b" className="field-error">
                  {fieldErrors.b}
                </small>
              )}
            </div>
          )}
        </div>

        <div className="actions">
          <button type="submit" className="primary" disabled={loading}>
            {loading ? 'Calculating…' : 'Calculate'}
          </button>
          <button type="button" className="secondary" onClick={handleClear}>
            Clear
          </button>
        </div>
      </form>

      <div className="output" aria-live="polite">
        {serverError && (
          <p role="alert" className="server-error">
            {serverError}
          </p>
        )}
        {result && !serverError && (
          <p className="result">
            <span className="expression">{result.expression} =</span>
            <strong data-testid="result">{formatResult(result.value)}</strong>
          </p>
        )}
      </div>

      {history.length > 0 && (
        <div className="history">
          <h2>Recent</h2>
          <ul>
            {history.map((h) => (
              <li key={h.id}>
                <span>{h.expression}</span>
                <span>= {formatResult(h.result)}</span>
              </li>
            ))}
          </ul>
        </div>
      )}
    </section>
  )
}
