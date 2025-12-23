package converter

// ConvertFunc is a generic function that converts a value of type `From` to a
// value of type `To`.
//
// It returns the converted value and an error if the conversion fails.
type ConvertFunc[From, To any] func(from From) (To, error)

// ConvertSECAToArubaFunc is a specific conversion function that converts a
// SECA resource to an Aruba resource.
//
// TODO: restrict the Aruba types to a common interface (instead any).
type ConvertSECAToArubaFunc[S any, A any] ConvertFunc[S, A]

// ConvertArubaToSECAFunc is a specific conversion function that converts an
// Aruba resource to a SECA resource.
//
// TODO: restrict the Aruba types to a common interface (instead any).
type ConvertArubaToSECAFunc[S any, A any] ConvertFunc[A, S]

// Converter is an interface for types that can convert between SECA and Aruba
// resources.
type Converter[S any, A any] interface {
	// FromSECAToAruba converts a SECA resource to an Aruba resource.
	FromSECAToAruba(from S) (A, error)
	// FromArubaToSECA converts an Aruba resource to a SECA resource.
	FromArubaToSECA(from A) (S, error)
}

// GenericConverter is a generic implementation of the Converter interface.
//
// It uses two functions to perform the conversions.
type GenericConverter[S any, A any] struct {
	fromSECAToAruba ConvertSECAToArubaFunc[S, A]
	fromArubaToSECA ConvertArubaToSECAFunc[S, A]
}

// FromSECAToAruba converts a SECA resource to an Aruba resource using the
// provided conversion function.
func (c *GenericConverter[S, A]) FromSECAToAruba(from S) (A, error) {
	return c.fromSECAToAruba(from)
}

// FromArubaToSECA converts an Aruba resource to a SECA resource using the
// provided conversion function.
func (c *GenericConverter[S, A]) FromArubaToSECA(from A) (S, error) {
	return c.fromArubaToSECA(from)
}

var _ Converter[float64, int] = (*GenericConverter[float64, int])(nil)
