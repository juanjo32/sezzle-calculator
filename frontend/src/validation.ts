import type { Operation } from './operations'
import { getOperation } from './operations'

export interface ValidationResult {
  a?: number
  b?: number
  errors: { a?: string; b?: string }
}

// Accepts numbers like "12", "-3.5", ".5", "1e3". Rejects empty strings,
// "abc", "1.2.3" and values that overflow to Infinity.
export function parseNumber(raw: string): number | undefined {
  const trimmed = raw.trim()
  if (trimmed === '') return undefined
  const n = Number(trimmed)
  return Number.isFinite(n) ? n : undefined
}

// Avoids showing floating-point noise like 0.30000000000000004.
export function formatResult(n: number): string {
  return Number.parseFloat(n.toPrecision(12)).toString()
}

// Client-side validation gives instant feedback. The backend validates again:
// the frontend is a convenience, the API is the source of truth.
export function validateInputs(op: Operation, rawA: string, rawB: string): ValidationResult {
  const errors: ValidationResult['errors'] = {}
  const unary = getOperation(op).unary

  const a = parseNumber(rawA)
  if (rawA.trim() === '') errors.a = 'Enter a number'
  else if (a === undefined) errors.a = 'Not a valid number'
  else if (op === 'sqrt' && a < 0) errors.a = 'Cannot take the square root of a negative number'

  let b: number | undefined
  if (!unary) {
    b = parseNumber(rawB)
    if (rawB.trim() === '') errors.b = 'Enter a number'
    else if (b === undefined) errors.b = 'Not a valid number'
    else if (op === 'divide' && b === 0) errors.b = 'Cannot divide by zero'
  }

  return { a, b, errors }
}
