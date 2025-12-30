package converter

import (
	aruba_converter_port "github.com/eu-sovereign-cloud/ecp/foundation/plugin/aruba/pkg/port/converter"
)

// GenericConverter is a generic implementation of the Converter interface.
//
// It uses two functions to perform the conversions.
type GenericConverter[S, A any] struct {
	fromSECAToAruba aruba_converter_port.ConvertSECAToArubaFunc[S, A]
	fromArubaToSECA aruba_converter_port.ConvertArubaToSECAFunc[S, A]
}

var _ aruba_converter_port.Converter[float64, int] = (*GenericConverter[float64, int])(nil)

// NewGenericConverter creates and returns a new GenericConverter with its
// internal conversion functions initialized to nil. These can be set later
// using SetFromSECAToArubaFunc and SetFromArubaToSECAFunc.
func NewGenericConverter[S, A any]() *GenericConverter[S, A] {
	return &GenericConverter[S, A]{}
}

// NewGenericConverterWithFuncs creates and returns a new GenericConverter
// initialized with the provided SECA to Aruba and Aruba to SECA conversion
// functions.
func NewGenericConverterWithFuncs[S, A any](
	fromSECAToAruba aruba_converter_port.ConvertSECAToArubaFunc[S, A],
	fromArubaToSECA aruba_converter_port.ConvertArubaToSECAFunc[S, A],
) *GenericConverter[S, A] {
	return &GenericConverter[S, A]{
		fromSECAToAruba: fromSECAToAruba,
		fromArubaToSECA: fromArubaToSECA,
	}
}

// SetFromSECAToArubaFunc sets the conversion function responsible for
// converting a SECA resource of type S to an Aruba resource of type A.
func (c *GenericConverter[S, A]) SetFromSECAToArubaFunc(converterFunc aruba_converter_port.ConvertSECAToArubaFunc[S, A]) {
	c.fromSECAToAruba = converterFunc
}

// SetFromArubaToSECAFunc sets the conversion function responsible for
// converting an Aruba resource of type A to a SECA resource of type S.
func (c *GenericConverter[S, A]) SetFromArubaToSECAFunc(converterFunc aruba_converter_port.ConvertArubaToSECAFunc[S, A]) {
	c.fromArubaToSECA = converterFunc
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
