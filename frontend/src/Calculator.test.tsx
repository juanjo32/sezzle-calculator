import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import Calculator from './Calculator'
import { formatResult } from './validation'
import * as api from './api'

vi.mock('./api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('./api')>()
  return { ...actual, calculate: vi.fn() }
})

const calculateMock = vi.mocked(api.calculate)

function fillAndSubmit(user: ReturnType<typeof userEvent.setup>, a: string, b?: string) {
  return (async () => {
    // user.type('') throws, so empty strings mean "leave the field blank".
    if (a) await user.type(screen.getByLabelText(/^(first number|number)$/i), a)
    if (b) await user.type(screen.getByLabelText(/^(second number|percent)$/i), b)
    await user.click(screen.getByRole('button', { name: 'Calculate' }))
  })()
}

beforeEach(() => {
  calculateMock.mockReset()
})

describe('Calculator', () => {
  it('calculates and shows the result and history', async () => {
    calculateMock.mockResolvedValue({ operation: 'add', a: 2, b: 3, result: 5 })
    const user = userEvent.setup()
    render(<Calculator />)

    await fillAndSubmit(user, '2', '3')

    expect(await screen.findByTestId('result')).toHaveTextContent('5')
    expect(screen.getByText('2 + 3 =')).toBeInTheDocument()
    expect(calculateMock).toHaveBeenCalledWith('add', 2, 3, expect.any(AbortSignal))
    expect(screen.getByRole('heading', { name: 'Recent' })).toBeInTheDocument()
  })

  it('shows validation errors and does not call the API', async () => {
    const user = userEvent.setup()
    render(<Calculator />)

    await fillAndSubmit(user, 'abc', '')

    expect(screen.getByText('Not a valid number')).toBeInTheDocument()
    expect(screen.getByText('Enter a number')).toBeInTheDocument()
    expect(calculateMock).not.toHaveBeenCalled()
  })

  it('blocks division by zero on the client', async () => {
    const user = userEvent.setup()
    render(<Calculator />)

    await user.click(screen.getByRole('button', { name: 'Divide' }))
    await fillAndSubmit(user, '5', '0')

    expect(screen.getByText('Cannot divide by zero')).toBeInTheDocument()
    expect(calculateMock).not.toHaveBeenCalled()
  })

  it('hides the second input for square root and sends a single operand', async () => {
    calculateMock.mockResolvedValue({ operation: 'sqrt', a: 16, result: 4 })
    const user = userEvent.setup()
    render(<Calculator />)

    await user.click(screen.getByRole('button', { name: 'Square root' }))
    expect(screen.queryByLabelText(/second number/i)).not.toBeInTheDocument()

    await fillAndSubmit(user, '16')

    expect(await screen.findByTestId('result')).toHaveTextContent('4')
    expect(calculateMock).toHaveBeenCalledWith('sqrt', 16, undefined, expect.any(AbortSignal))
  })

  it('displays errors returned by the backend', async () => {
    calculateMock.mockRejectedValue(new api.ApiError('NON_FINITE_RESULT', 'result is not a finite number', 422))
    const user = userEvent.setup()
    render(<Calculator />)

    await user.click(screen.getByRole('button', { name: 'Power' }))
    await fillAndSubmit(user, '10', '400')

    expect(await screen.findByRole('alert')).toHaveTextContent('result is not a finite number')
    expect(screen.queryByTestId('result')).not.toBeInTheDocument()
  })

  it('shows a generic message for unexpected errors', async () => {
    calculateMock.mockRejectedValue(new Error('boom'))
    const user = userEvent.setup()
    render(<Calculator />)

    await fillAndSubmit(user, '1', '1')

    expect(await screen.findByRole('alert')).toHaveTextContent('Something went wrong')
  })

  it('disables the button while a request is in flight', async () => {
    let resolve!: (v: api.CalculateResponse) => void
    calculateMock.mockReturnValue(new Promise((r) => (resolve = r)))
    const user = userEvent.setup()
    render(<Calculator />)

    await fillAndSubmit(user, '1', '2')
    expect(screen.getByRole('button', { name: 'Calculating…' })).toBeDisabled()

    resolve({ operation: 'add', a: 1, b: 2, result: 3 })
    await waitFor(() => expect(screen.getByRole('button', { name: 'Calculate' })).toBeEnabled())
  })

  it('clears inputs, result and errors', async () => {
    calculateMock.mockResolvedValue({ operation: 'add', a: 2, b: 3, result: 5 })
    const user = userEvent.setup()
    render(<Calculator />)

    await fillAndSubmit(user, '2', '3')
    await screen.findByTestId('result')
    await user.click(screen.getByRole('button', { name: 'Clear' }))

    expect(screen.queryByTestId('result')).not.toBeInTheDocument()
    expect(screen.getByLabelText(/first number/i)).toHaveValue('')
  })

  it('keeps only the five most recent calculations', async () => {
    const user = userEvent.setup()
    render(<Calculator />)

    for (let i = 1; i <= 6; i++) {
      calculateMock.mockResolvedValueOnce({ operation: 'add', a: i, b: 0, result: i })
      await user.clear(screen.getByLabelText(/first number/i))
      await user.clear(screen.getByLabelText(/second number/i))
      await fillAndSubmit(user, String(i), '0')
      await screen.findByText(`${i} + 0 =`)
    }

    expect(screen.getAllByRole('listitem')).toHaveLength(5)
    expect(screen.queryByText('1 + 0')).not.toBeInTheDocument()
  })
})

describe('formatResult', () => {
  it('removes floating-point noise', () => {
    expect(formatResult(0.1 + 0.2)).toBe('0.3')
    expect(formatResult(2.5)).toBe('2.5')
    expect(formatResult(1e21)).toBe('1e+21')
  })
})
