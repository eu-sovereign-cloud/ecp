package convert

import (
	"fmt"
)

func InterfaceToString(input interface{}) string {
	switch v := input.(type) {
	case string:
		return v
	case fmt.Stringer:
		return v.String()
	default:
		return fmt.Sprintf("%v", v)
	}
}
