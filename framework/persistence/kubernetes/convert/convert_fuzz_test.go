package convert

import "testing"

// FuzzStringRoundTrip verifies that StringToInterface→InterfaceToString is idempotent:
// once normalized by one pass, a second pass must produce identical output.
func FuzzStringRoundTrip(f *testing.F) {
	f.Add(`"hello"`)
	f.Add(`42`)
	f.Add(`-1`)
	f.Add(`3.14`)
	f.Add(`true`)
	f.Add(`false`)
	f.Add(`null`)
	f.Add(`{"key":"value"}`)
	f.Add(`[1,2,3]`)
	f.Add(`[[1],{"a":null}]`)
	f.Add(``)
	f.Add(`not-json`)
	f.Add("\x97") // invalid UTF-8: was unstable before normalization fix
	f.Add("-0.0") // float64 negative zero: marshals as "-0" but k8s parses "-0" as int64

	f.Fuzz(func(t *testing.T, input string) {
		str1 := InterfaceToString(StringToInterface(input))
		str2 := InterfaceToString(StringToInterface(str1))
		if str1 != str2 {
			t.Errorf("unstable: input=%q pass1=%q pass2=%q", input, str1, str2)
		}
	})
}
