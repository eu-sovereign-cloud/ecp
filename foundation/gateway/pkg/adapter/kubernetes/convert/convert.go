package convert

import (
	"fmt"

	"k8s.io/apimachinery/pkg/util/json"
)

// InterfaceToString converts an interface{} to its string representation by marshaling to JSON. This way, the original
// data types are preserved.
func InterfaceToString(input interface{}) string {
	if s, err := json.Marshal(input); err == nil {
		return string(s)
	}

	// Fallback case
	return fmt.Sprintf("%v", input)
}

// StringToInterface attempts to convert a string to an interface{} by unmarshaling it as JSON. This way, the original
// data types are preserved.
func StringToInterface(input string) interface{} {
	var res interface{}
	if err := json.Unmarshal([]byte(input), &res); err == nil {
		return res
	}

	// Fallback case
	return input
}
