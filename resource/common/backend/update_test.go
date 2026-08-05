package backend

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	backendport "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/backend"
	"github.com/eu-sovereign-cloud/ecp/resource/common/domain"
)

// updatable is the smallest thing HandleUpdate accepts: an identifiable resource carrying a status.
type updatable struct {
	name   string
	status domain.Status
}

func (u *updatable) GetName() string      { return u.name }
func (u *updatable) GetVersion() string   { return "" }
func (u *updatable) GetTenant() string    { return "t1" }
func (u *updatable) GetWorkspace() string { return "w1" }

// countingRepo records how many times status was persisted, which is the half of HandleUpdate's
// contract that assertions on the conditions alone cannot see: the controller reconciles on its own
// writes, so a write per pass never settles.
type countingRepo struct{ statusWrites int }

func (r *countingRepo) Delete(context.Context, *updatable) error { return nil }
func (r *countingRepo) Create(_ context.Context, u *updatable) (**updatable, error) {
	return &u, nil
}

func (r *countingRepo) Update(_ context.Context, u *updatable) (**updatable, error) {
	return &u, nil
}

func (r *countingRepo) UpdateStatus(_ context.Context, u *updatable) (**updatable, error) {
	r.statusWrites++
	return &u, nil
}

func succeeds(context.Context, *updatable) error { return nil }

func fails(context.Context, *updatable) error { return errors.New("provider refused") }

const maxConditions = 10

// TestHandleUpdate_ClearsBuriedFailure is the regression test for a recovered update that stayed
// reported as failed. The failure is only the head condition until something else is pushed over
// it - a block-storage resize stepping through "updating", an instance's power transition - and a
// clear that only inspected the head left the buried one in place forever.
func TestHandleUpdate_ClearsBuriedFailure(t *testing.T) {
	resource := &updatable{name: "r1"}
	repo := &countingRepo{}

	// A failed update, then a lifecycle transition on top of it, exactly as a resize would leave it.
	requeue, err := HandleUpdate(context.Background(), resource, &resource.status, fails, repo, maxConditions)
	require.NoError(t, err)
	require.True(t, requeue, "a transient failure must be retried")
	require.Equal(t, updateFailedConditionType, resource.status.Conditions[0].Type)

	resource.status.PushCondition(ConditionFromState(domain.ResourceStateUpdating))
	resource.status.PushCondition(ConditionFromState(domain.ResourceStateActive))
	require.NotEqual(t, updateFailedConditionType, resource.status.Conditions[0].Type,
		"the failure must be buried for this test to mean anything")

	_, err = HandleUpdate(context.Background(), resource, &resource.status, succeeds, repo, maxConditions)
	require.NoError(t, err)

	for _, c := range resource.status.Conditions {
		require.NotEqual(t, updateFailedConditionType, c.Type,
			"a buried failure must be retracted once the update succeeds")
	}
	require.Equal(t, domain.ResourceStateActive, resource.status.State)
}

// TestHandleUpdate_SuccessIsQuiet pins the other half: clearing must be idempotent. Removing the
// stale condition and pushing a fresh active one is only correct if the next pass finds nothing
// left to do - otherwise every reconcile of a healthy resource writes status and re-triggers itself.
func TestHandleUpdate_SuccessIsQuiet(t *testing.T) {
	resource := &updatable{name: "r1"}
	repo := &countingRepo{}

	_, err := HandleUpdate(context.Background(), resource, &resource.status, fails, repo, maxConditions)
	require.NoError(t, err)
	require.Equal(t, 1, repo.statusWrites, "the failure is reported once")

	// First success retracts it and writes; every later one must be silent.
	for range 5 {
		_, err = HandleUpdate(context.Background(), resource, &resource.status, succeeds, repo, maxConditions)
		require.NoError(t, err)
	}

	require.Equal(t, 2, repo.statusWrites,
		"one write to retract the failure, and nothing after it")
}

// TestHandleUpdate_StableFailureReportedOnce pins that a provider failing the same way on every
// pass is reported once rather than re-written each reconcile.
func TestHandleUpdate_StableFailureReportedOnce(t *testing.T) {
	resource := &updatable{name: "r1"}
	repo := &countingRepo{}

	for range 5 {
		requeue, err := HandleUpdate(context.Background(), resource, &resource.status, fails, repo, maxConditions)
		require.NoError(t, err)
		require.True(t, requeue)
	}

	require.Equal(t, 1, repo.statusWrites, "an unchanged failure must not be re-written")
}

// TestHandleUpdate_NotSupportedIsNotRetried pins that a provider which will never accept a change
// is not asked again, unlike a transient failure.
func TestHandleUpdate_NotSupportedIsNotRetried(t *testing.T) {
	resource := &updatable{name: "r1"}
	repo := &countingRepo{}

	requeue, err := HandleUpdate(context.Background(), resource, &resource.status,
		func(context.Context, *updatable) error {
			return errors.Join(backendport.ErrNotSupported, errors.New("region is immutable"))
		}, repo, maxConditions)

	require.NoError(t, err)
	require.False(t, requeue, "a refused change must not be re-issued")
	require.Equal(t, updateFailedConditionType, resource.status.Conditions[0].Type)
	require.Equal(t, domain.ResourceStateActive, resource.status.State,
		"a failed update leaves the resource active so a corrected spec is still picked up")
}

// TestHandleUpdate_StillProcessingLeavesStatusAlone pins that an in-flight update reports nothing:
// it has not failed, and writing status would re-trigger the reconcile it is waiting on.
func TestHandleUpdate_StillProcessingLeavesStatusAlone(t *testing.T) {
	resource := &updatable{name: "r1"}
	repo := &countingRepo{}

	requeue, err := HandleUpdate(context.Background(), resource, &resource.status,
		func(context.Context, *updatable) error { return backendport.ErrStillProcessing },
		repo, maxConditions)

	require.NoError(t, err)
	require.True(t, requeue)
	require.Empty(t, resource.status.Conditions)
	require.Zero(t, repo.statusWrites)
}
