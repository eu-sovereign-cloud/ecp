package delegated

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
)

//go:generate mockgen -package delegated -destination=zz_mock_identifiable_test.go github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/port IdentifiableResource
//go:generate mockgen -package delegated -destination=zz_mock_converter_test.go github.com/eu-sovereign-cloud/ecp/foundation/plugin/aruba/pkg/port/converter Converter

func TestGenericDelegated_Do(t *testing.T) {
	t.Run("should report an error when conversion fails", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		//
		// Given a SECA identifiable resource containig bad data
		resource := NewMockIdentifiableResource(ctrl)

		resource.EXPECT().GetName().Return("bad name").Times(1)

		//
		// And a converter which will return an error when detect bad data
		converter := NewMockConverter[*MockIdentifiableResource, map[string]any](ctrl)

		errInvalidName := errors.New("invalid name")
		converter.EXPECT().FromSECAToAruba(gomock.AssignableToTypeOf(resource)).DoAndReturn(
			func(from *MockIdentifiableResource) (map[string]any, error) {
				if from.GetName() != "good name" {
					return nil, errInvalidName
				}

				return map[string]any{"name": from.GetName()}, nil
			},
		).Times(1)

		//
		// And a delegated which uses this above mentioned converter
		delegated := GenericDelegated[*MockIdentifiableResource, map[string]any]{converter: converter}

		//
		// When we try to perform the delegated action
		err := delegated.Do(t.Context(), resource)

		//
		// Then it should return the conversion error properly wrapped
		require.ErrorIs(t, err, errInvalidName)
	})
}
