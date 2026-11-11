// Operations supported by the backend. Kept in one place so the UI, the
// validation and the API client agree on names and arity.
export type Operation =
  | 'add'
  | 'subtract'
  | 'multiply'
  | 'divide'
  | 'power'
  | 'sqrt'
  | 'percentage'

export interface OperationMeta {
  value: Operation
  label: string
  symbol: string
  unary: boolean
}

export const OPERATIONS: OperationMeta[] = [
  { value: 'add', label: 'Add', symbol: '+', unary: false },
  { value: 'subtract', label: 'Subtract', symbol: '−', unary: false },
  { value: 'multiply', label: 'Multiply', symbol: '×', unary: false },
  { value: 'divide', label: 'Divide', symbol: '÷', unary: false },
  { value: 'power', label: 'Power', symbol: 'xʸ', unary: false },
  { value: 'sqrt', label: 'Square root', symbol: '√', unary: true },
  { value: 'percentage', label: 'Percentage', symbol: '%', unary: false },
]

export function getOperation(op: Operation): OperationMeta {
  const meta = OPERATIONS.find((o) => o.value === op)
  if (!meta) throw new Error(`Unknown operation: ${op}`)
  return meta
}

// Human-readable expression for the result panel and history,
// e.g. "10 ÷ 4", "√16", "15% of 200".
export function formatExpression(op: Operation, a: number, b?: number): string {
  switch (op) {
    case 'sqrt':
      return `√${a}`
    case 'percentage':
      return `${b}% of ${a}`
    case 'power':
      return `${a} ^ ${b}`
    default:
      return `${a} ${getOperation(op).symbol} ${b}`
  }
}
