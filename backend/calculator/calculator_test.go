package calculator

import (
	"errors"
	"math"
	"testing"
)

func ptr(f float64) *float64 { return &f }

func TestCalculate(t *testing.T) {
	tests := []struct {
		name    string
		op      Operation
		a, b    *float64
		want    float64
		wantErr error
	}{
		// Basic operations
		{name: "add", op: Add, a: ptr(2), b: ptr(3), want: 5},
		{name: "add negatives", op: Add, a: ptr(-2.5), b: ptr(-0.5), want: -3},
		{name: "subtract", op: Subtract, a: ptr(10), b: ptr(4), want: 6},
		{name: "subtract to negative", op: Subtract, a: ptr(4), b: ptr(10), want: -6},
		{name: "multiply", op: Multiply, a: ptr(6), b: ptr(7), want: 42},
		{name: "multiply by zero", op: Multiply, a: ptr(123), b: ptr(0), want: 0},
		{name: "divide", op: Divide, a: ptr(10), b: ptr(4), want: 2.5},
		{name: "divide zero by number", op: Divide, a: ptr(0), b: ptr(5), want: 0},

		// Advanced operations
		{name: "power", op: Power, a: ptr(2), b: ptr(10), want: 1024},
		{name: "power negative exponent", op: Power, a: ptr(2), b: ptr(-1), want: 0.5},
		{name: "power zero exponent", op: Power, a: ptr(0), b: ptr(0), want: 1},
		{name: "sqrt", op: Sqrt, a: ptr(16), want: 4},
		{name: "sqrt zero", op: Sqrt, a: ptr(0), want: 0},
		{name: "sqrt ignores b", op: Sqrt, a: ptr(9), b: ptr(100), want: 3},
		{name: "percentage", op: Percentage, a: ptr(200), b: ptr(15), want: 30},
		{name: "percentage over 100", op: Percentage, a: ptr(50), b: ptr(150), want: 75},

		// Edge cases and validation
		{name: "division by zero", op: Divide, a: ptr(1), b: ptr(0), wantErr: ErrDivisionByZero},
		{name: "division by negative zero", op: Divide, a: ptr(1), b: ptr(math.Copysign(0, -1)), wantErr: ErrDivisionByZero},
		{name: "sqrt of negative", op: Sqrt, a: ptr(-4), wantErr: ErrNegativeSqrt},
		{name: "unsupported operation", op: "modulo", a: ptr(1), b: ptr(2), wantErr: ErrUnsupportedOperation},
		{name: "missing a", op: Add, b: ptr(2), wantErr: ErrMissingOperand},
		{name: "missing b on binary op", op: Add, a: ptr(2), wantErr: ErrMissingOperand},
		{name: "NaN operand", op: Add, a: ptr(math.NaN()), b: ptr(1), wantErr: ErrInvalidOperand},
		{name: "infinite operand b", op: Add, a: ptr(1), b: ptr(math.Inf(1)), wantErr: ErrInvalidOperand},
		{name: "overflow multiply", op: Multiply, a: ptr(math.MaxFloat64), b: ptr(10), wantErr: ErrNonFiniteResult},
		{name: "overflow power", op: Power, a: ptr(10), b: ptr(400), wantErr: ErrNonFiniteResult},
		{name: "negative base fractional exponent", op: Power, a: ptr(-8), b: ptr(0.5), wantErr: ErrNonFiniteResult},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Calculate(tc.op, tc.a, tc.b)

			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("expected error %v, got %v (result %v)", tc.wantErr, err, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if math.Abs(got-tc.want) > 1e-9 {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestSupportedAndUnary(t *testing.T) {
	for _, op := range SupportedOperations() {
		if !Supported(op) {
			t.Errorf("%s should be supported", op)
		}
	}
	// The advertised list is maintained by hand, so make sure it stays in sync
	// with the operations map: same size and no duplicates.
	seen := map[Operation]bool{}
	for _, op := range SupportedOperations() {
		if seen[op] {
			t.Errorf("%s is listed twice", op)
		}
		seen[op] = true
	}
	if len(seen) != len(operations) {
		t.Errorf("SupportedOperations lists %d operations, map has %d", len(seen), len(operations))
	}
	if Supported("modulo") {
		t.Error("modulo should not be supported")
	}
	if !IsUnary(Sqrt) {
		t.Error("sqrt should be unary")
	}
	if IsUnary(Add) {
		t.Error("add should be binary")
	}
}
