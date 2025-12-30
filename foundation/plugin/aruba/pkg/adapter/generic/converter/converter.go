package converter

import (
	aruba_converter_port "github.com/eu-sovereign-cloud/ecp/foundation/plugin/aruba/pkg/port/converter"
)

// GenericConverter is a generic implementation of the Converter interface.
//
// It uses two functions to perform the conversions.
type GenericConverter[S any, A any] struct {
	fromSECAToAruba aruba_converter_port.ConvertSECAToArubaFunc[S, A]
	fromArubaToSECA aruba_converter_port.ConvertArubaToSECAFunc[S, A]
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

var _ aruba_converter_port.Converter[float64, int] = (*GenericConverter[float64, int])(nil)
