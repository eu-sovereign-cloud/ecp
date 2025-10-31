package validation

const (
	DefaultLimit = 1000
	MinLimit     = 1
	MaxLimit     = 10000
)

// GetLimit returns a validated limit value from an optional integer pointer.
// If the provided limit is nil, it returns the default limit.
// If the provided limit is greater than the maximum allowed limit, it returns the maximum limit.
func GetLimit(limit *int) int {
	if limit == nil {
		return DefaultLimit
	}
	if *limit < MinLimit {
		return MinLimit
	}
	if *limit > MaxLimit {
		return MaxLimit
	}
	return *limit
}
