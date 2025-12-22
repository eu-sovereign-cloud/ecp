package converter

import (
	seca_gateway_port "github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/port"
)

// ConvertFunc is a generic function that converts a value of type `From` to a
// value of type `To`.
//
// It returns the converted value and an error if the conversion fails.
type ConvertFunc[From, To any] func(from From) (To, error)

// ConvertSECAToArubaFunc is a specific conversion function that converts a
// SECA resource to an Aruba resource.
//
// TODO: restrict the Aruba types to a common interface (instead any).
type ConvertSECAToArubaFunc[S seca_gateway_port.IdentifiableResource, A any] ConvertFunc[S, A]

// ConvertArubaToSECAFunc is a specific conversion function that converts an
// Aruba resource to a SECA resource.
//
// TODO: restrict the Aruba types to a common interface (instead any).
type ConvertArubaToSECAFunc[S seca_gateway_port.IdentifiableResource, A any] ConvertFunc[A, S]

// Converter is an interface for types that can convert between SECA and Aruba
// resources.
type Converter[S seca_gateway_port.IdentifiableResource, A any] interface {
	// FromSECAToAruba converts a SECA resource to an Aruba resource.
	FromSECAToAruba(from S) (A, error)
	// FromArubaToSECA converts an Aruba resource to a SECA resource.
	FromArubaToSECA(from A) (S, error)
}
