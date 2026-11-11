package api

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func doRequest(t *testing.T, h http.Handler, method, path, body string, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestCalculateSuccess(t *testing.T) {
	router := NewRouter()
	tests := []struct {
		name string
		body string
		want float64
	}{
		{"add", `{"operation":"add","a":2,"b":3}`, 5},
		{"divide", `{"operation":"divide","a":10,"b":4}`, 2.5},
		{"zero operand is valid", `{"operation":"multiply","a":0,"b":9}`, 0},
		{"operation is case-insensitive", `{"operation":" ADD ","a":1,"b":1}`, 2},
		{"sqrt with only a", `{"operation":"sqrt","a":81}`, 9},
		{"percentage", `{"operation":"percentage","a":200,"b":15}`, 30},
		{"power", `{"operation":"power","a":2,"b":8}`, 256},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := doRequest(t, router, http.MethodPost, "/api/v1/calculate", tc.body, nil)
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, body = %s", rec.Code, rec.Body)
			}
			if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
				t.Errorf("Content-Type = %q", ct)
			}
			var resp CalculateResponse
			if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
				t.Fatalf("invalid JSON response: %v", err)
			}
			if resp.Result != tc.want {
				t.Errorf("result = %v, want %v", resp.Result, tc.want)
			}
		})
	}
}

func TestSqrtResponseOmitsB(t *testing.T) {
	rec := doRequest(t, NewRouter(), http.MethodPost, "/api/v1/calculate", `{"operation":"sqrt","a":4,"b":7}`, nil)
	if strings.Contains(rec.Body.String(), `"b"`) {
		t.Errorf("unary response should not echo b: %s", rec.Body)
	}
}

func TestCalculateErrors(t *testing.T) {
	router := NewRouter()
	tests := []struct {
		name       string
		body       string
		headers    map[string]string
		wantStatus int
		wantCode   string
	}{
		{"division by zero", `{"operation":"divide","a":1,"b":0}`, nil, http.StatusUnprocessableEntity, CodeDivisionByZero},
		{"negative sqrt", `{"operation":"sqrt","a":-1}`, nil, http.StatusUnprocessableEntity, CodeNegativeSqrt},
		{"overflow", `{"operation":"power","a":10,"b":400}`, nil, http.StatusUnprocessableEntity, CodeNonFiniteResult},
		{"unsupported operation", `{"operation":"modulo","a":1,"b":2}`, nil, http.StatusBadRequest, CodeUnsupportedOperation},
		{"missing operation", `{"a":1,"b":2}`, nil, http.StatusBadRequest, CodeInvalidInput},
		{"missing a", `{"operation":"add","b":2}`, nil, http.StatusBadRequest, CodeInvalidInput},
		{"missing b", `{"operation":"add","a":2}`, nil, http.StatusBadRequest, CodeInvalidInput},
		{"null b", `{"operation":"add","a":2,"b":null}`, nil, http.StatusBadRequest, CodeInvalidInput},
		{"string operand", `{"operation":"add","a":"2","b":3}`, nil, http.StatusBadRequest, CodeInvalidInput},
		{"malformed JSON", `{"operation":"add",`, nil, http.StatusBadRequest, CodeInvalidJSON},
		{"empty body", ``, nil, http.StatusBadRequest, CodeInvalidJSON},
		{"unknown field", `{"operation":"add","a":1,"b":2,"c":3}`, nil, http.StatusBadRequest, CodeInvalidJSON},
		{"trailing data", `{"operation":"add","a":1,"b":2}{"x":1}`, nil, http.StatusBadRequest, CodeInvalidJSON},
		{"wrong content type", `{"operation":"add","a":1,"b":2}`, map[string]string{"Content-Type": "text/plain"}, http.StatusUnsupportedMediaType, CodeInvalidJSON},
		{"payload too large", `{"operation":"add","a":1,"b":2,"pad":"` + strings.Repeat("x", 5000) + `"}`, nil, http.StatusRequestEntityTooLarge, CodePayloadTooLarge},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := doRequest(t, router, http.MethodPost, "/api/v1/calculate", tc.body, tc.headers)
			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d, body = %s", rec.Code, tc.wantStatus, rec.Body)
			}
			var body ErrorBody
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("error response is not JSON: %v", err)
			}
			if body.Error.Code != tc.wantCode {
				t.Errorf("code = %q, want %q", body.Error.Code, tc.wantCode)
			}
			if body.Error.Message == "" {
				t.Error("error message should not be empty")
			}
		})
	}
}

func TestMethodNotAllowed(t *testing.T) {
	rec := doRequest(t, NewRouter(), http.MethodGet, "/api/v1/calculate", "", nil)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", rec.Code)
	}
}

func TestHealthAndOperations(t *testing.T) {
	router := NewRouter()

	rec := doRequest(t, router, http.MethodGet, "/healthz", "", nil)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "ok") {
		t.Errorf("healthz: %d %s", rec.Code, rec.Body)
	}

	rec = doRequest(t, router, http.MethodGet, "/api/v1/operations", "", nil)
	var body struct {
		Operations []OperationInfo `json:"operations"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Operations) != 7 {
		t.Fatalf("expected 7 operations, got %d", len(body.Operations))
	}
	for _, op := range body.Operations {
		if op.Name == "sqrt" && op.Operands != 1 {
			t.Error("sqrt should report 1 operand")
		}
		if op.Name == "add" && op.Operands != 2 {
			t.Error("add should report 2 operands")
		}
	}
}

func TestCORS(t *testing.T) {
	h := CORS("http://localhost:5173", NewRouter())

	rec := doRequest(t, h, http.MethodOptions, "/api/v1/calculate", "", nil)
	if rec.Code != http.StatusNoContent {
		t.Errorf("preflight status = %d", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:5173" {
		t.Errorf("allow-origin = %q", got)
	}

	rec = doRequest(t, CORS("", NewRouter()), http.MethodGet, "/healthz", "", nil)
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("CORS disabled but header set: %q", got)
	}
}

func TestLogging(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	h := Logging(logger, NewRouter())

	doRequest(t, h, http.MethodPost, "/api/v1/calculate", `{"operation":"divide","a":1,"b":0}`, nil)

	line, _ := io.ReadAll(&buf)
	if !strings.Contains(string(line), `"status":422`) || !strings.Contains(string(line), `"path":"/api/v1/calculate"`) {
		t.Errorf("unexpected log line: %s", line)
	}
}
