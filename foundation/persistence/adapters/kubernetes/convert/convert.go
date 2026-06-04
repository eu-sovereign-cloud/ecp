package convert

import (
	"fmt"
	"math"
	"strings"
	"unicode/utf8"

	"k8s.io/apimachinery/pkg/util/json"
)

// InterfaceToString converts an any to its string representation by marshaling to JSON. This way, the original
// data types are preserved.
func InterfaceToString(input any) string {
	// json.Marshal(-0.0) → "-0", but k8s JSON parses "-0" as int64(0), not float64.
	// Normalize to avoid this type-changing instability across round-trips.
	if f, ok := input.(float64); ok && f == 0 && math.Signbit(f) {
		input = float64(0)
	}
	// json.Marshal emits \uXXXX escapes for invalid UTF-8 bytes, but once those
	// are unmarshaled they become actual UTF-8 bytes that marshal differently.
	// Normalize upfront so the output is always the direct-bytes form.
	if s, ok := input.(string); ok && !utf8.ValidString(s) {
		input = strings.ToValidUTF8(s, "�")
	}
	if s, err := json.Marshal(input); err == nil {
		return string(s)
	}

	// Fallback case
	return fmt.Sprintf("%v", input)
}

// StringToInterface attempts to convert a string to an any by unmarshaling it as JSON. This way, the original
// data types are preserved.
func StringToInterface(input string) any {
	var res any
	if err := json.Unmarshal([]byte(input), &res); err == nil {
		return res
	}

	// Fallback case: plain string. Normalize invalid UTF-8 so that a subsequent
	// InterfaceToString→StringToInterface cycle is stable (invalid bytes would
	// otherwise marshal as the � escape but unmarshal as the actual U+FFFD
	// bytes, producing different output on the second pass).
	if !utf8.ValidString(input) {
		return strings.ToValidUTF8(input, "�")
	}
	return input
}
