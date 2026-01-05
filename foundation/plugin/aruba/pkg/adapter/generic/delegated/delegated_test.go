package delegated

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
)

//go:generate mockgen -package delegated -destination=zz_mock_identifiable_test.go github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/port IdentifiableResource
//go:generate mockgen -package delegated -destination=zz_mock_resolver_test.go github.com/eu-sovereign-cloud/ecp/foundation/plugin/aruba/pkg/port/resolver DependenciesResolver
//go:generate mockgen -package delegated -destination=zz_mock_converter_test.go github.com/eu-sovereign-cloud/ecp/foundation/plugin/aruba/pkg/port/converter Converter

func TestGenericDelegated_Do(t *testing.T) {
	const (
		goodResourceWorkspace = "good resource workspace"
		badResourceWorkspace  = "bad resource workspace"

		goodResourceName = "good resource name"
		badResourceName  = "bad resource name"
	)

	var (
		errInvalidWorkspace = errors.New("invalid workspace")
		errInvalidName      = errors.New("invalid name")
	)

	type secaBundle struct {
		main       *MockIdentifiableResource
		dependency *MockIdentifiableResource
	}

	type arubaBundle struct {
		main       map[string]any
		dependency map[string]any
	}

	t.Run("should report an error when seca dependencies resolving fails", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		//
		// Given a SECA identifiable resource containig bad data
		resource := NewMockIdentifiableResource(ctrl)

		resource.EXPECT().GetWorkspace().Return(badResourceWorkspace).Times(1)

		//
		// And a SECA dependencies resolver which will return an error when
		// detect bad data
		secaResolver := NewMockDependenciesResolver[*MockIdentifiableResource, *secaBundle](ctrl)

		secaResolver.EXPECT().ResolveDependencies(
			gomock.AssignableToTypeOf(t.Context()),
			gomock.AssignableToTypeOf(resource),
		).DoAndReturn(func(ctx context.Context, main *MockIdentifiableResource) (*secaBundle, error) {
			if main.GetWorkspace() != goodResourceWorkspace {
				return nil, errInvalidWorkspace
			}

			return &secaBundle{main: main}, nil
		})

		//
		// And a delegated which uses this above mentioned converter
		delegated := GenericDelegated[*MockIdentifiableResource, *secaBundle, *arubaBundle]{
			secaResolver: secaResolver,
		}

		//
		// When we try to perform the delegated action
		err := delegated.Do(t.Context(), resource)

		//
		// Then it should return the conversion error properly wrapped
		require.ErrorIs(t, err, errInvalidWorkspace)
	})

	t.Run("should report an error when conversion fails", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		//
		// Given a SECA identifiable resource containig bad data
		resource := NewMockIdentifiableResource(ctrl)

		resource.EXPECT().GetWorkspace().Return(goodResourceWorkspace).Times(1)
		resource.EXPECT().GetName().Return(badResourceName).Times(1)

		//
		// And a SECA dependencies resolver
		secaResolver := NewMockDependenciesResolver[*MockIdentifiableResource, *secaBundle](ctrl)

		secaResolver.EXPECT().ResolveDependencies(
			gomock.AssignableToTypeOf(t.Context()),
			gomock.AssignableToTypeOf(resource),
		).DoAndReturn(func(ctx context.Context, main *MockIdentifiableResource) (*secaBundle, error) {
			if main.GetWorkspace() != goodResourceWorkspace {
				return nil, errInvalidWorkspace
			}

			return &secaBundle{main: main}, nil
		})

		//
		// And a converter which will return an error when detect bad data
		converter := NewMockConverter[*secaBundle, *arubaBundle](ctrl)

		converter.EXPECT().FromSECAToAruba(gomock.AssignableToTypeOf(&secaBundle{})).DoAndReturn(
			func(from *secaBundle) (*arubaBundle, error) {
				if from.main.GetName() != goodResourceName {
					return nil, errInvalidName
				}

				return &arubaBundle{main: map[string]any{"name": from.main.GetName()}}, nil
			},
		).Times(1)

		//
		// And a delegated which uses this above mentioned converter
		delegated := GenericDelegated[*MockIdentifiableResource, *secaBundle, *arubaBundle]{
			secaResolver: secaResolver,
			converter:    converter,
		}

		//
		// When we try to perform the delegated action
		err := delegated.Do(t.Context(), resource)

		//
		// Then it should return the conversion error properly wrapped
		require.ErrorIs(t, err, errInvalidName)
	})
}
