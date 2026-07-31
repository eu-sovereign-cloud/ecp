package rest

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	roledom "github.com/eu-sovereign-cloud/ecp/resource/authorization/v1/role"
	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
)

// fakeRoleWriter records which write methods were called and lets Create fail.
type fakeRoleWriter struct {
	createErr        error
	createCalled     bool
	updateStatusArg  *roledom.Role
	updateStatusDone bool
}

func (f *fakeRoleWriter) Delete(context.Context, *roledom.Role) error { return nil }

func (f *fakeRoleWriter) Create(_ context.Context, m *roledom.Role) (**roledom.Role, error) {
	f.createCalled = true
	if f.createErr != nil {
		return nil, f.createErr
	}
	return &m, nil
}

func (f *fakeRoleWriter) Update(_ context.Context, m *roledom.Role) (**roledom.Role, error) {
	return &m, nil
}

func (f *fakeRoleWriter) UpdateStatus(_ context.Context, m *roledom.Role) (**roledom.Role, error) {
	f.updateStatusDone = true
	f.updateStatusArg = m
	return &m, nil
}

func TestActiveCreator_MarksActiveAfterPersist(t *testing.T) {
	f := &fakeRoleWriter{}

	out, err := activeCreator[*roledom.Role](f, markRoleActive).Do(context.Background(), &roledom.Role{})

	require.NoError(t, err)
	require.True(t, f.createCalled)
	require.True(t, f.updateStatusDone, "status subresource must be written after the spec write")
	require.Equal(t, commondomain.ResourceStateActive, f.updateStatusArg.Status.State)
	require.Equal(t, commondomain.ResourceStateActive, out.Status.State)
}

func TestActiveCreator_PersistErrorSkipsStatusWrite(t *testing.T) {
	f := &fakeRoleWriter{createErr: errors.New("boom")}

	_, err := activeCreator[*roledom.Role](f, markRoleActive).Do(context.Background(), &roledom.Role{})

	require.Error(t, err)
	require.False(t, f.updateStatusDone, "status must not be written when the spec write fails")
}
