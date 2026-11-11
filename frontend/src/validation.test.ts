import { describe, expect, it } from 'vitest'
import { parseNumber, validateInputs } from './validation'
import { formatExpression } from './operations'

describe('parseNumber', () => {
  it.each([
    ['12', 12],
    ['-3.5', -3.5],
    ['.5', 0.5],
    ['1e3', 1000],
    ['  7  ', 7],
  ])('parses %j as %d', (raw, expected) => {
    expect(parseNumber(raw)).toBe(expected)
  })

  it.each(['', '   ', 'abc', '1.2.3', '1e400'])('rejects %j', (raw) => {
    expect(parseNumber(raw)).toBeUndefined()
  })
})

describe('validateInputs', () => {
  it('accepts two valid numbers for a binary operation', () => {
    expect(validateInputs('add', '2', '3')).toEqual({ a: 2, b: 3, errors: {} })
  })

  it('requires both operands for binary operations', () => {
    const { errors } = validateInputs('multiply', '', '')
    expect(errors.a).toBe('Enter a number')
    expect(errors.b).toBe('Enter a number')
  })

  it('flags non-numeric input', () => {
    expect(validateInputs('add', 'abc', '1').errors.a).toBe('Not a valid number')
    expect(validateInputs('add', '1', 'x').errors.b).toBe('Not a valid number')
  })

  it('blocks division by zero before calling the API', () => {
    expect(validateInputs('divide', '5', '0').errors.b).toBe('Cannot divide by zero')
  })

  it('ignores the second operand for square root', () => {
    expect(validateInputs('sqrt', '9', '')).toEqual({ a: 9, b: undefined, errors: {} })
  })

  it('blocks square root of a negative number', () => {
    expect(validateInputs('sqrt', '-4', '').errors.a).toMatch(/negative/)
  })
})

describe('formatExpression', () => {
  it('formats each kind of operation', () => {
    expect(formatExpression('add', 1, 2)).toBe('1 + 2')
    expect(formatExpression('divide', 10, 4)).toBe('10 ÷ 4')
    expect(formatExpression('power', 2, 8)).toBe('2 ^ 8')
    expect(formatExpression('sqrt', 16)).toBe('√16')
    expect(formatExpression('percentage', 200, 15)).toBe('15% of 200')
  })
})
