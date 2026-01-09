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
//go:generate mockgen -package delegated -destination=zz_mock_mutator_test.go github.com/eu-sovereign-cloud/ecp/foundation/plugin/aruba/pkg/port/mutator Mutator
//go:generate mockgen -package delegated -destination=zz_mock_repository_test.go github.com/eu-sovereign-cloud/ecp/foundation/plugin/aruba/pkg/port/repository Writer,Watcher

func TestGenericDelegated_Do(t *testing.T) {
	const (
		goodResourceWorkspace = "good resource workspace"
		badResourceWorkspace  = "bad resource workspace"

		goodResourceName = "good resource name"
		badResourceName  = "bad resource name"

		resourceTagKey       = "a_tag_key"
		goodResourceTagValue = "good resource tag value"
		badResourceTagValue  = "bad resource tag value"
	)

	var (
		errInvalidWorkspace = errors.New("invalid workspace")
		errInvalidName      = errors.New("invalid name")
		errInvalidTag       = errors.New("invalid tag")
	)

	type secaBundleType struct {
		main       *MockIdentifiableResource
		dependency *MockIdentifiableResource
	}

	type arubaBundleType struct {
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
		secaResolver := NewMockDependenciesResolver[*MockIdentifiableResource, *secaBundleType](ctrl)

		secaResolver.EXPECT().ResolveDependencies(
			gomock.AssignableToTypeOf(t.Context()),
			gomock.AssignableToTypeOf(resource),
		).DoAndReturn(func(ctx context.Context, main *MockIdentifiableResource) (*secaBundleType, error) {
			if main.GetWorkspace() != goodResourceWorkspace {
				return nil, errInvalidWorkspace
			}

			return &secaBundleType{main: main}, nil
		}).Times(1)

		//
		// And a delegated which uses these above mentioned elements
		delegated := GenericDelegated[*MockIdentifiableResource, *secaBundleType, *arubaBundleType]{
			resolveSECAFunc: secaResolver.ResolveDependencies,
		}

		//
		// When we try to perform the delegated action
		err := delegated.Do(t.Context(), resource)

		//
		// Then it should return the dependency resolution error properly
		// wrapped
		require.ErrorIs(t, err, errInvalidWorkspace)
	})

	t.Run("should report an error when conversion fails", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		//
		// Given a SECA identifiable resource
		resource := NewMockIdentifiableResource(ctrl)

		resource.EXPECT().GetWorkspace().Return(goodResourceWorkspace).Times(1)
		resource.EXPECT().GetName().Return(badResourceName).Times(1)

		//
		// And a SECA dependencies resolver
		secaResolver := NewMockDependenciesResolver[*MockIdentifiableResource, *secaBundleType](ctrl)

		secaResolver.EXPECT().ResolveDependencies(
			gomock.AssignableToTypeOf(t.Context()),
			gomock.AssignableToTypeOf(resource),
		).DoAndReturn(func(ctx context.Context, main *MockIdentifiableResource) (*secaBundleType, error) {
			if main.GetWorkspace() != goodResourceWorkspace {
				return nil, errInvalidWorkspace
			}

			return &secaBundleType{main: main}, nil
		}).Times(1)

		//
		// And a converter which will return an error when detect bad data
		converter := NewMockConverter[*secaBundleType, *arubaBundleType](ctrl)

		converter.EXPECT().FromSECAToAruba(gomock.AssignableToTypeOf(&secaBundleType{})).DoAndReturn(
			func(from *secaBundleType) (*arubaBundleType, error) {
				if from.main.GetName() != goodResourceName {
					return nil, errInvalidName
				}

				return &arubaBundleType{main: map[string]any{"name": from.main.GetName()}}, nil
			},
		).Times(1)

		//
		// And a delegated which uses these above mentioned elements
		delegated := GenericDelegated[*MockIdentifiableResource, *secaBundleType, *arubaBundleType]{
			resolveSECAFunc: secaResolver.ResolveDependencies,
			convertFunc:     converter.FromSECAToAruba,
		}

		//
		// When we try to perform the delegated action
		err := delegated.Do(t.Context(), resource)

		//
		// Then it should return the conversion error properly wrapped
		require.ErrorIs(t, err, errInvalidName)
	})

	t.Run("should report an error when aruba dependencies resolving fails", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		//
		// Given a SECA identifiable resource
		resource := NewMockIdentifiableResource(ctrl)

		resource.EXPECT().GetWorkspace().Return(goodResourceWorkspace).Times(1)
		resource.EXPECT().GetName().Return(goodResourceName).Times(2)

		//
		// And a SECA dependencies resolver
		secaResolver := NewMockDependenciesResolver[*MockIdentifiableResource, *secaBundleType](ctrl)

		secaResolver.EXPECT().ResolveDependencies(
			gomock.AssignableToTypeOf(t.Context()),
			gomock.AssignableToTypeOf(resource),
		).DoAndReturn(func(ctx context.Context, main *MockIdentifiableResource) (*secaBundleType, error) {
			if main.GetWorkspace() != goodResourceWorkspace {
				return nil, errInvalidWorkspace
			}

			return &secaBundleType{main: main}, nil
		}).Times(1)

		//
		// And a converter which will inject a bad tag value
		converter := NewMockConverter[*secaBundleType, *arubaBundleType](ctrl)

		converter.EXPECT().FromSECAToAruba(gomock.AssignableToTypeOf(&secaBundleType{})).DoAndReturn(
			func(from *secaBundleType) (*arubaBundleType, error) {
				if from.main.GetName() != goodResourceName {
					return nil, errInvalidName
				}

				return &arubaBundleType{main: map[string]any{
					"name":         from.main.GetName(),
					resourceTagKey: badResourceTagValue,
				}}, nil
			},
		).Times(1)

		//
		// And a Aruba dependencies resolver which will return an error when detect a bad tag value
		arubaResolver := NewMockDependenciesResolver[*arubaBundleType, *arubaBundleType](ctrl)

		arubaResolver.EXPECT().ResolveDependencies(
			gomock.AssignableToTypeOf(t.Context()),
			gomock.AssignableToTypeOf(&arubaBundleType{}),
		).DoAndReturn(func(ctx context.Context, main *arubaBundleType) (*arubaBundleType, error) {
			if main.main[resourceTagKey] != goodResourceTagValue {
				return nil, errInvalidTag
			}

			main.dependency = map[string]any{resourceTagKey: goodResourceTagValue}

			return main, nil
		}).Times(1)

		//
		// And a delegated which uses these above mentioned elements
		delegated := GenericDelegated[*MockIdentifiableResource, *secaBundleType, *arubaBundleType]{
			resolveSECAFunc: secaResolver.ResolveDependencies,
			convertFunc:     converter.FromSECAToAruba,
			resolvArubaFunc: arubaResolver.ResolveDependencies,
		}

		//
		// When we try to perform the delegated action
		err := delegated.Do(t.Context(), resource)

		//
		// Then it should return the dependency resolution error properly
		// wrapped
		require.ErrorIs(t, err, errInvalidTag)
	})

	t.Run("should report an error when mutation fails", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		//
		// Given a SECA identifiable resource
		resource := NewMockIdentifiableResource(ctrl)

		resource.EXPECT().GetWorkspace().Return(goodResourceWorkspace).Times(1)
		resource.EXPECT().GetName().Return(goodResourceName).Times(2)

		//
		// And a SECA dependencies resolver
		secaResolver := NewMockDependenciesResolver[*MockIdentifiableResource, *secaBundleType](ctrl)

		secaResolver.EXPECT().ResolveDependencies(
			gomock.AssignableToTypeOf(t.Context()),
			gomock.AssignableToTypeOf(resource),
		).DoAndReturn(func(ctx context.Context, main *MockIdentifiableResource) (*secaBundleType, error) {
			if main.GetWorkspace() != goodResourceWorkspace {
				return nil, errInvalidWorkspace
			}

			return &secaBundleType{main: main}, nil
		}).Times(1)

		//
		// And a converter
		converter := NewMockConverter[*secaBundleType, *arubaBundleType](ctrl)

		converter.EXPECT().FromSECAToAruba(gomock.AssignableToTypeOf(&secaBundleType{})).DoAndReturn(
			func(from *secaBundleType) (*arubaBundleType, error) {
				if from.main.GetName() != goodResourceName {
					return nil, errInvalidName
				}

				return &arubaBundleType{main: map[string]any{
					"name":         from.main.GetName(),
					resourceTagKey: goodResourceTagValue,
				}}, nil
			},
		).Times(1)

		//
		// And a Aruba dependencies resolver which will inject a bad tag value
		arubaResolver := NewMockDependenciesResolver[*arubaBundleType, *arubaBundleType](ctrl)

		arubaResolver.EXPECT().ResolveDependencies(
			gomock.AssignableToTypeOf(t.Context()),
			gomock.AssignableToTypeOf(&arubaBundleType{}),
		).DoAndReturn(func(ctx context.Context, main *arubaBundleType) (*arubaBundleType, error) {
			if main.main[resourceTagKey] != goodResourceTagValue {
				return nil, errInvalidTag
			}

			main.dependency = map[string]any{resourceTagKey: badResourceTagValue}

			return main, nil
		})

		//
		// And a mutator which will return an error when detect a bad tag value
		mutator := NewMockMutator[*arubaBundleType, *secaBundleType](ctrl)

		mutator.EXPECT().Mutate(
			gomock.AssignableToTypeOf(&arubaBundleType{}),
			gomock.AssignableToTypeOf(&secaBundleType{}),
		).DoAndReturn(
			func(mutable *arubaBundleType, params *secaBundleType) error {
				if mutable.dependency[resourceTagKey] != goodResourceTagValue {
					return errInvalidTag
				}

				return nil
			},
		).Times(1)

		//
		// And a delegated which uses these above mentioned elements
		delegated := GenericDelegated[*MockIdentifiableResource, *secaBundleType, *arubaBundleType]{
			resolveSECAFunc: secaResolver.ResolveDependencies,
			convertFunc:     converter.FromSECAToAruba,
			resolvArubaFunc: arubaResolver.ResolveDependencies,
			mutateFunc:      mutator.Mutate,
		}

		//
		// When we try to perform the delegated action
		err := delegated.Do(t.Context(), resource)

		//
		// Then it should return the mutation error properly wrapped
		require.ErrorIs(t, err, errInvalidTag)
	})

	t.Run("should report an error when propagation fails", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		//
		// Given a SECA identifiable resource
		resource := NewMockIdentifiableResource(ctrl)

		resource.EXPECT().GetWorkspace().Return(goodResourceWorkspace).Times(1)
		resource.EXPECT().GetName().Return(goodResourceName).Times(2)

		//
		// And a SECA dependencies resolver
		secaResolver := NewMockDependenciesResolver[*MockIdentifiableResource, *secaBundleType](ctrl)

		secaResolver.EXPECT().ResolveDependencies(
			gomock.AssignableToTypeOf(t.Context()),
			gomock.AssignableToTypeOf(resource),
		).DoAndReturn(func(ctx context.Context, main *MockIdentifiableResource) (*secaBundleType, error) {
			if main.GetWorkspace() != goodResourceWorkspace {
				return nil, errInvalidWorkspace
			}

			return &secaBundleType{main: main}, nil
		}).Times(1)

		//
		// And a converter
		converter := NewMockConverter[*secaBundleType, *arubaBundleType](ctrl)

		converter.EXPECT().FromSECAToAruba(gomock.AssignableToTypeOf(&secaBundleType{})).DoAndReturn(
			func(from *secaBundleType) (*arubaBundleType, error) {
				if from.main.GetName() != goodResourceName {
					return nil, errInvalidName
				}

				return &arubaBundleType{main: map[string]any{
					"name":         from.main.GetName(),
					resourceTagKey: goodResourceTagValue,
				}}, nil
			},
		).Times(1)

		//
		// And a Aruba dependencies resolver
		arubaResolver := NewMockDependenciesResolver[*arubaBundleType, *arubaBundleType](ctrl)

		arubaResolver.EXPECT().ResolveDependencies(
			gomock.AssignableToTypeOf(t.Context()),
			gomock.AssignableToTypeOf(&arubaBundleType{}),
		).DoAndReturn(func(ctx context.Context, main *arubaBundleType) (*arubaBundleType, error) {
			if main.main[resourceTagKey] != goodResourceTagValue {
				return nil, errInvalidTag
			}

			main.dependency = map[string]any{resourceTagKey: goodResourceTagValue}

			return main, nil
		})

		//
		// And a mutator  which will inject a bad tag value
		mutator := NewMockMutator[*arubaBundleType, *secaBundleType](ctrl)

		mutator.EXPECT().Mutate(
			gomock.AssignableToTypeOf(&arubaBundleType{}),
			gomock.AssignableToTypeOf(&secaBundleType{}),
		).DoAndReturn(
			func(mutable *arubaBundleType, params *secaBundleType) error {
				if mutable.dependency[resourceTagKey] != goodResourceTagValue {
					return errInvalidTag
				}

				mutable.dependency[resourceTagKey] = badResourceTagValue

				return nil
			},
		).Times(1)

		//
		// And a repository writer which will return an error when detect a bad tag value
		writer := NewMockWriter[*arubaBundleType](ctrl)

		writer.EXPECT().Update(
			gomock.AssignableToTypeOf(t.Context()),
			gomock.AssignableToTypeOf(&arubaBundleType{}),
		).DoAndReturn(
			func(ctx context.Context, resource *arubaBundleType) error {
				if resource.dependency[resourceTagKey] != goodResourceTagValue {
					return errInvalidTag
				}

				return nil
			},
		).Times(1)

		//
		// And a delegated which uses these above mentioned elements
		delegated := GenericDelegated[*MockIdentifiableResource, *secaBundleType, *arubaBundleType]{
			resolveSECAFunc: secaResolver.ResolveDependencies,
			convertFunc:     converter.FromSECAToAruba,
			resolvArubaFunc: arubaResolver.ResolveDependencies,
			mutateFunc:      mutator.Mutate,
			propagateFunc:   writer.Update,
		}

		//
		// When we try to perform the delegated action
		err := delegated.Do(t.Context(), resource)

		//
		// Then it should return the propagation error properly wrapped
		require.ErrorIs(t, err, errInvalidTag)
	})
}
