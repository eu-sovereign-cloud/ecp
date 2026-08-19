package backend

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	schemav1 "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes/schema/v1"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel"

	"github.com/eu-sovereign-cloud/ecp/resource/common/domain"
)

// The enum mappers used to answer "" for anything they did not recognise, which made a typo in a
// hand-edited CR indistinguishable from a status nobody had written yet. These tests pin the two
// halves of the replacement: every state the CRD allows still round-trips, and anything else is
// reported as a kernel.Error a caller can errors.As across the layer boundary.

func TestResourceStateRoundTripsEveryState(t *testing.T) {
	states := []domain.ResourceState{
		domain.ResourceStatePending,
		domain.ResourceStateCreating,
		domain.ResourceStateActive,
		domain.ResourceStateUpdating,
		domain.ResourceStateDeleting,
		domain.ResourceStateError,
	}

	for _, state := range states {
		t.Run(string(state), func(t *testing.T) {
			cr, err := ResourceStateToCR(state)
			require.NoError(t, err)

			back, err := ResourceStateFromCR(cr)
			require.NoError(t, err)
			assert.Equal(t, state, back)
		})
	}
}

func TestResourceStateFromCRRejectsUnknownState(t *testing.T) {
	state, err := ResourceStateFromCR(schemav1.ResourceState("halfway"))

	require.Error(t, err)
	assert.Empty(t, state)

	// The whole point of the change: the failure survives errors.As at any depth, carries a kind
	// the REST layer can map, and names the value that caused it.
	var domErr *kernel.Error
	require.ErrorAs(t, err, &domErr)
	assert.Equal(t, kernel.KindValidation, domErr.Kind)
	require.ErrorIs(t, err, kernel.ErrValidation)
	assert.Equal(t, []kernel.ErrorSource{{Name: "status.state", Value: "halfway"}}, domErr.Sources)
	assert.Contains(t, err.Error(), `"halfway"`)
}

// An unwritten status is not a corrupt one: the CR simply has no state yet, and flattening that to
// empty is the correct answer rather than a swallowed failure.
func TestResourceStateFromCRCarriesEmptyThrough(t *testing.T) {
	state, err := ResourceStateFromCR("")

	require.NoError(t, err)
	assert.Empty(t, state)
}

// The outbound direction keeps its old contract - the CRD requires the field, so there is no
// empty state to write - but reports it as a kernel.Error instead of a nil sentinel the caller
// had to translate into an error string of its own.
func TestResourceStateToCRRejectsUnmappableState(t *testing.T) {
	for _, state := range []domain.ResourceState{"", "halfway"} {
		t.Run("state="+string(state), func(t *testing.T) {
			cr, err := ResourceStateToCR(state)

			require.Error(t, err)
			assert.Empty(t, cr)

			var domErr *kernel.Error
			require.ErrorAs(t, err, &domErr)
			assert.Equal(t, kernel.KindValidation, domErr.Kind)
		})
	}
}

func TestConditionsFromCRPropagatesTheChain(t *testing.T) {
	conds := []schemav1.StatusCondition{
		{Type: "Reconcile", State: schemav1.ResourceStateActive},
		{Type: "Broken", State: schemav1.ResourceState("halfway")},
	}

	got, err := ConditionsFromCR(conds)

	require.Error(t, err)
	assert.Nil(t, got)
	// The condition that failed is named, and the kind survives the extra wrapping layer.
	assert.Contains(t, err.Error(), `condition "Broken"`)

	var domErr *kernel.Error
	require.ErrorAs(t, err, &domErr)
	assert.Equal(t, kernel.KindValidation, domErr.Kind)
}

func TestConditionsToCRPropagatesTheChain(t *testing.T) {
	conds := []domain.StatusCondition{
		{Type: "Reconcile", State: domain.ResourceStateActive},
		{Type: "Broken", State: domain.ResourceState("halfway")},
	}

	got, err := ConditionsToCR(conds)

	require.Error(t, err)
	assert.Nil(t, got)
	assert.Contains(t, err.Error(), `condition "Broken"`)

	var domErr *kernel.Error
	require.ErrorAs(t, err, &domErr)
	assert.Equal(t, kernel.KindValidation, domErr.Kind)
}

func TestConditionsRoundTrip(t *testing.T) {
	conds := []domain.StatusCondition{
		{Type: "Reconcile", State: domain.ResourceStateActive, Reason: "Active", Message: "ok", Occurrences: 2},
	}

	cr, err := ConditionsToCR(conds)
	require.NoError(t, err)

	back, err := ConditionsFromCR(cr)
	require.NoError(t, err)
	require.Len(t, back, 1)
	assert.Equal(t, conds[0].Type, back[0].Type)
	assert.Equal(t, conds[0].State, back[0].State)
	assert.Equal(t, conds[0].Reason, back[0].Reason)
	assert.Equal(t, conds[0].Message, back[0].Message)
	assert.Equal(t, conds[0].Occurrences, back[0].Occurrences)
}

func TestIPVersionFromCR(t *testing.T) {
	testCases := []struct {
		name    string
		in      schemav1.IPVersion
		want    domain.IPVersion
		wantErr bool
	}{
		{name: "ipv4", in: schemav1.IPVersionIPv4, want: domain.IPVersionIPv4},
		{name: "ipv6", in: schemav1.IPVersionIPv6, want: domain.IPVersionIPv6},
		{name: "unset is optional on some specs", in: ""},
		{name: "anything else is corrupt", in: schemav1.IPVersion("IPv9"), wantErr: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := IPVersionFromCR(tc.in)
			if tc.wantErr {
				require.Error(t, err)
				var domErr *kernel.Error
				require.ErrorAs(t, err, &domErr)
				assert.Equal(t, kernel.KindValidation, domErr.Kind)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}
