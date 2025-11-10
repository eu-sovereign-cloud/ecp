package regionalhandler

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"

	sdkcompute "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.compute.v1"
)

type ComputeTestSuite struct {
	Handler *Compute
}

func NewComputeTestSuite(t *testing.T) *ComputeTestSuite {
	t.Helper()

	return &ComputeTestSuite{
		Handler: NewCompute(slog.Default(), nil),
	}

}

func TestComputeHandler_ListSkus(t *testing.T) {
	t.Run("should panic because is not implemented", func(t *testing.T) {
		suite := NewComputeTestSuite(t)

		require.PanicsWithValue(t, "implement me", func() {
			suite.Handler.ListSkus(nil, nil, "", sdkcompute.ListSkusParams{})
		})
	})
}

func TestComputeHandler_GetSku(t *testing.T) {
	t.Run("should panic because is not implemented", func(t *testing.T) {
		suite := NewComputeTestSuite(t)

		require.PanicsWithValue(t, "implement me", func() {
			suite.Handler.GetSku(nil, nil, "", "")
		})
	})
}

func TestComputeHandler_ListInstances(t *testing.T) {
	t.Run("should panic because is not implemented", func(t *testing.T) {
		suite := NewComputeTestSuite(t)

		require.PanicsWithValue(t, "implement me", func() {
			suite.Handler.ListInstances(nil, nil, "", "", sdkcompute.ListInstancesParams{})
		})
	})
}

func TestComputeHandler_DeleteInstance(t *testing.T) {
	t.Run("should panic because is not implemented", func(t *testing.T) {
		suite := NewComputeTestSuite(t)

		require.PanicsWithValue(t, "implement me", func() {
			suite.Handler.DeleteInstance(nil, nil, "", "", "", sdkcompute.DeleteInstanceParams{})
		})
	})
}

func TestComputeHandler_GetInstance(t *testing.T) {
	t.Run("should panic because is not implemented", func(t *testing.T) {
		suite := NewComputeTestSuite(t)

		require.PanicsWithValue(t, "implement me", func() {
			suite.Handler.GetInstance(nil, nil, "", "", "")
		})
	})
}

func TestComputeHandler_CreateOrUpdateInstance(t *testing.T) {
	t.Run("should panic because is not implemented", func(t *testing.T) {
		suite := NewComputeTestSuite(t)

		require.PanicsWithValue(t, "implement me", func() {
			suite.Handler.CreateOrUpdateInstance(nil, nil, "", "", "", sdkcompute.CreateOrUpdateInstanceParams{})
		})
	})
}

func TestComputeHandler_RestartInstance(t *testing.T) {
	t.Run("should panic because is not implemented", func(t *testing.T) {
		suite := NewComputeTestSuite(t)

		require.PanicsWithValue(t, "implement me", func() {
			suite.Handler.RestartInstance(nil, nil, "", "", "", sdkcompute.RestartInstanceParams{})
		})
	})
}

func TestComputeHandler_StartInstance(t *testing.T) {
	t.Run("should panic because is not implemented", func(t *testing.T) {
		suite := NewComputeTestSuite(t)

		require.PanicsWithValue(t, "implement me", func() {
			suite.Handler.StartInstance(nil, nil, "", "", "", sdkcompute.StartInstanceParams{})
		})
	})
}

func TestComputeHandler_StopInstance(t *testing.T) {
	t.Run("should panic because is not implemented", func(t *testing.T) {
		suite := NewComputeTestSuite(t)

		require.PanicsWithValue(t, "implement me", func() {
			suite.Handler.StopInstance(nil, nil, "", "", "", sdkcompute.StopInstanceParams{})
		})
	})
}
