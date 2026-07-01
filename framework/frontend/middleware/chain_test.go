package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestChain verifies that Chain reverses the provided middleware slice so that
// callers can supply middlewares in natural execution order while oapi-codegen's
// inside-out wrapping still applies them in that order.
func TestChain(t *testing.T) {
	t.Parallel()

	// Each middleware appends its label to a shared string builder.
	var sb strings.Builder

	maker := func(label string) func(http.Handler) http.Handler {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				sb.WriteString(label + "→")
				next.ServeHTTP(w, r)
			})
		}
	}

	type alias func(http.Handler) http.Handler
	chain := Chain[alias](maker("A"), maker("B"), maker("C"))

	if len(chain) != 3 {
		t.Fatalf("len = %d, want 3", len(chain))
	}

	// Simulate oapi-codegen's inside-out wrapping: apply from index 0 downward so
	// the LAST element runs outermost (first executed). Chain must have reversed
	// the order so that A ends up outermost.
	handler := http.Handler(okHandler)
	for _, mw := range chain {
		handler = mw(handler)
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	handler.ServeHTTP(w, r)

	// After Chain reverses [A,B,C] → [C,B,A], oapi-codegen wraps:
	//   handler = A(B(C(okHandler)))
	// Execution: A runs first, then B, then C.
	got := sb.String()
	want := "A→B→C→"
	if got != want {
		t.Errorf("execution order = %q, want %q", got, want)
	}
}

func TestChain_Empty(t *testing.T) {
	t.Parallel()
	type alias func(http.Handler) http.Handler
	chain := Chain[alias]()
	if len(chain) != 0 {
		t.Errorf("expected empty chain, got len=%d", len(chain))
	}
}

func TestChain_SingleElement(t *testing.T) {
	t.Parallel()
	type alias func(http.Handler) http.Handler
	called := false
	mw := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			next.ServeHTTP(w, r)
		})
	}
	chain := Chain[alias](mw)
	if len(chain) != 1 {
		t.Fatalf("len = %d, want 1", len(chain))
	}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	chain[0](okHandler).ServeHTTP(w, r)
	if !called {
		t.Error("middleware was not called")
	}
}
