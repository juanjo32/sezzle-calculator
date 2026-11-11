// Package calculator contains the pure arithmetic logic of the service.
//
// It has no knowledge of HTTP or JSON: every function receives float64
// operands and returns either a finite result or a typed error. Keeping the
// domain logic isolated makes it trivial to unit test and to reuse from any
// transport (REST today, gRPC or a CLI tomorrow).
package calculator

import (
	"errors"
	"fmt"
	"math"
)

// Operation identifies a supported calculator operation.
type Operation string

const (
	Add        Operation = "add"
	Subtract   Operation = "subtract"
	Multiply   Operation = "multiply"
	Divide     Operation = "divide"
	Power      Operation = "power"
	Sqrt       Operation = "sqrt"
	Percentage Operation = "percentage"
)

// Sentinel errors. Callers use errors.Is to map them to API error codes.
var (
	ErrDivisionByZero       = errors.New("division by zero")
	ErrNegativeSqrt         = errors.New("square root of a negative number")
	ErrUnsupportedOperation = errors.New("unsupported operation")
	ErrMissingOperand       = errors.New("missing operand")
	ErrInvalidOperand       = errors.New("operand must be a finite number")
	ErrNonFiniteResult      = errors.New("result is not a finite number")
)

// spec describes how many operands an operation needs and how to compute it.
type spec struct {
	unary bool
	fn    func(a, b float64) (float64, error)
}

var operations = map[Operation]spec{
	Add:      {fn: func(a, b float64) (float64, error) { return a + b, nil }},
	Subtract: {fn: func(a, b float64) (float64, error) { return a - b, nil }},
	Multiply: {fn: func(a, b float64) (float64, error) { return a * b, nil }},
	Divide: {fn: func(a, b float64) (float64, error) {
		if b == 0 {
			return 0, ErrDivisionByZero
		}
		return a / b, nil
	}},
	Power: {fn: func(a, b float64) (float64, error) { return math.Pow(a, b), nil }},
	Sqrt: {unary: true, fn: func(a, _ float64) (float64, error) {
		if a < 0 {
			return 0, ErrNegativeSqrt
		}
		return math.Sqrt(a), nil
	}},
	// Percentage returns b percent of a, e.g. percentage(200, 15) = 30.
	Percentage: {fn: func(a, b float64) (float64, error) { return a * b / 100, nil }},
}

// IsUnary reports whether op takes a single operand. Unknown operations
// return false; use Supported to validate the operation first.
func IsUnary(op Operation) bool {
	return operations[op].unary
}

// Supported reports whether op is a known operation.
func Supported(op Operation) bool {
	_, ok := operations[op]
	return ok
}

// SupportedOperations returns the list of operations in a stable order,
// used by the API to advertise what it can do.
func SupportedOperations() []Operation {
	return []Operation{Add, Subtract, Multiply, Divide, Power, Sqrt, Percentage}
}

// Calculate validates the inputs and evaluates op.
//
// a is always required. b is required for binary operations and must be nil
// for unary ones only in the sense that it is ignored; callers may pass it.
// The result is guaranteed to be finite: overflow (e.g. 1e308 * 10) or
// undefined results (e.g. (-8)^0.5) are reported as ErrNonFiniteResult
// instead of leaking +Inf or NaN, which JSON cannot represent.
func Calculate(op Operation, a, b *float64) (float64, error) {
	s, ok := operations[op]
	if !ok {
		return 0, fmt.Errorf("%w: %q", ErrUnsupportedOperation, op)
	}
	if a == nil {
		return 0, fmt.Errorf("%w: a", ErrMissingOperand)
	}
	if !isFinite(*a) {
		return 0, fmt.Errorf("%w: a", ErrInvalidOperand)
	}

	var bv float64
	if !s.unary {
		if b == nil {
			return 0, fmt.Errorf("%w: b", ErrMissingOperand)
		}
		if !isFinite(*b) {
			return 0, fmt.Errorf("%w: b", ErrInvalidOperand)
		}
		bv = *b
	}

	result, err := s.fn(*a, bv)
	if err != nil {
		return 0, err
	}
	if !isFinite(result) {
		return 0, ErrNonFiniteResult
	}
	return result, nil
}

func isFinite(f float64) bool {
	return !math.IsNaN(f) && !math.IsInf(f, 0)
}
