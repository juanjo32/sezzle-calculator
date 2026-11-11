// Package api exposes the calculator over HTTP/JSON.
package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/juanjo32/sezzle-calculator/backend/calculator"
)

// maxBodyBytes caps request size. A calculation payload is a few dozen bytes,
// so anything larger is either a mistake or abuse.
const maxBodyBytes = 4 << 10 // 4 KiB

// CalculateRequest is the body of POST /api/v1/calculate.
// Operands are pointers so we can tell "missing" apart from an explicit 0.
type CalculateRequest struct {
	Operation string   `json:"operation"`
	A         *float64 `json:"a"`
	B         *float64 `json:"b,omitempty"`
}

// CalculateResponse is returned on success.
type CalculateResponse struct {
	Operation string   `json:"operation"`
	A         float64  `json:"a"`
	B         *float64 `json:"b,omitempty"`
	Result    float64  `json:"result"`
}

// OperationInfo describes a supported operation for clients.
type OperationInfo struct {
	Name     string `json:"name"`
	Operands int    `json:"operands"`
}

// ErrorBody is the envelope for every error response.
type ErrorBody struct {
	Error APIError `json:"error"`
}

// APIError carries a stable machine-readable code and a human message.
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Error codes returned by the API. They are part of the public contract.
const (
	CodeInvalidJSON          = "INVALID_JSON"
	CodeUnsupportedMedia     = "UNSUPPORTED_MEDIA_TYPE"
	CodeInvalidInput         = "INVALID_INPUT"
	CodeUnsupportedOperation = "UNSUPPORTED_OPERATION"
	CodeDivisionByZero       = "DIVISION_BY_ZERO"
	CodeNegativeSqrt         = "NEGATIVE_SQRT"
	CodeNonFiniteResult      = "NON_FINITE_RESULT"
	CodePayloadTooLarge      = "PAYLOAD_TOO_LARGE"
	CodeInternal             = "INTERNAL_ERROR"
)

// NewRouter wires all routes. It uses the Go 1.22+ standard library mux,
// which supports method-based patterns and answers 405 automatically.
func NewRouter() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", handleHealth)
	mux.HandleFunc("GET /api/v1/operations", handleOperations)
	mux.HandleFunc("POST /api/v1/calculate", handleCalculate)
	return mux
}

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func handleOperations(w http.ResponseWriter, _ *http.Request) {
	ops := calculator.SupportedOperations()
	out := make([]OperationInfo, 0, len(ops))
	for _, op := range ops {
		n := 2
		if calculator.IsUnary(op) {
			n = 1
		}
		out = append(out, OperationInfo{Name: string(op), Operands: n})
	}
	writeJSON(w, http.StatusOK, map[string]any{"operations": out})
}

func handleCalculate(w http.ResponseWriter, r *http.Request) {
	req, apiErr, status := decodeRequest(w, r)
	if apiErr != nil {
		writeError(w, status, apiErr.Code, apiErr.Message)
		return
	}

	op := calculator.Operation(strings.ToLower(strings.TrimSpace(req.Operation)))
	result, err := calculator.Calculate(op, req.A, req.B)
	if err != nil {
		status, code := mapError(err)
		writeError(w, status, code, err.Error())
		return
	}

	resp := CalculateResponse{Operation: string(op), A: *req.A, Result: result}
	if !calculator.IsUnary(op) {
		resp.B = req.B
	}
	writeJSON(w, http.StatusOK, resp)
}

// decodeRequest parses and structurally validates the JSON body.
func decodeRequest(w http.ResponseWriter, r *http.Request) (CalculateRequest, *APIError, int) {
	var req CalculateRequest

	if ct := r.Header.Get("Content-Type"); ct != "" && !strings.HasPrefix(ct, "application/json") {
		return req, &APIError{CodeUnsupportedMedia, "Content-Type must be application/json"}, http.StatusUnsupportedMediaType
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(&req); err != nil {
		var maxErr *http.MaxBytesError
		var typeErr *json.UnmarshalTypeError
		switch {
		case errors.As(err, &maxErr):
			return req, &APIError{CodePayloadTooLarge, fmt.Sprintf("request body must not exceed %d bytes", maxBodyBytes)}, http.StatusRequestEntityTooLarge
		case errors.As(err, &typeErr):
			return req, &APIError{CodeInvalidInput, fmt.Sprintf("field %q must be a %s", typeErr.Field, typeErr.Type)}, http.StatusBadRequest
		case errors.Is(err, io.EOF):
			return req, &APIError{CodeInvalidJSON, "request body is empty"}, http.StatusBadRequest
		default:
			return req, &APIError{CodeInvalidJSON, "request body is not valid JSON: " + err.Error()}, http.StatusBadRequest
		}
	}
	// Reject trailing data such as `{...}{...}`.
	if dec.More() {
		return req, &APIError{CodeInvalidJSON, "request body must contain a single JSON object"}, http.StatusBadRequest
	}
	if strings.TrimSpace(req.Operation) == "" {
		return req, &APIError{CodeInvalidInput, `field "operation" is required`}, http.StatusBadRequest
	}
	return req, nil, 0
}

// mapError translates domain errors into HTTP status codes and API codes.
// Input problems are 400; mathematically undefined results are 422
// (the request is well-formed but cannot be processed).
func mapError(err error) (int, string) {
	switch {
	case errors.Is(err, calculator.ErrUnsupportedOperation):
		return http.StatusBadRequest, CodeUnsupportedOperation
	case errors.Is(err, calculator.ErrMissingOperand), errors.Is(err, calculator.ErrInvalidOperand):
		return http.StatusBadRequest, CodeInvalidInput
	case errors.Is(err, calculator.ErrDivisionByZero):
		return http.StatusUnprocessableEntity, CodeDivisionByZero
	case errors.Is(err, calculator.ErrNegativeSqrt):
		return http.StatusUnprocessableEntity, CodeNegativeSqrt
	case errors.Is(err, calculator.ErrNonFiniteResult):
		return http.StatusUnprocessableEntity, CodeNonFiniteResult
	default:
		return http.StatusInternalServerError, CodeInternal
	}
}

func writeError(w http.ResponseWriter, status int, code, msg string) {
	writeJSON(w, status, ErrorBody{Error: APIError{Code: code, Message: msg}})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
