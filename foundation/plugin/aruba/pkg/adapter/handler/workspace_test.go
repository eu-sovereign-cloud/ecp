package handler

import (
	"context"
	"fmt"
	"testing"

	"github.com/Arubacloud/arubacloud-resource-operator/api/v1alpha1"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestWorkspace_create(t *testing.T) {
	tests := []struct {
		name        string
		setupMocks  func(*MockRepository[*v1alpha1.Project, *v1alpha1.ProjectList], *MockConverter[*regional.WorkspaceDomain, *v1alpha1.Project], *regional.WorkspaceDomain)
		wantErr     bool
		errContains string
	}{
		{
			name: "conversion error",
			setupMocks: func(mockRepo *MockRepository[*v1alpha1.Project, *v1alpha1.ProjectList], mockConv *MockConverter[*regional.WorkspaceDomain, *v1alpha1.Project], workspaceDomain *regional.WorkspaceDomain) {
				mockConv.
					EXPECT().
					FromSECAToAruba(workspaceDomain).
					Return(nil, fmt.Errorf("conversion error"))
			},
			wantErr:     true,
			errContains: "conversion error",
		},
		{
			name: "create error",
			setupMocks: func(mockRepo *MockRepository[*v1alpha1.Project, *v1alpha1.ProjectList], mockConv *MockConverter[*regional.WorkspaceDomain, *v1alpha1.Project], workspaceDomain *regional.WorkspaceDomain) {
				prj := &v1alpha1.Project{
					Spec: v1alpha1.ProjectSpec{
						Description: "Test Workspace Description",
						Tags:        []string{"tag1", "tag2"},
						Default:     true,
					},
				}

				mockConv.
					EXPECT().
					FromSECAToAruba(workspaceDomain).
					Return(prj, nil)

				mockRepo.
					EXPECT().
					Create(context.Background(), prj).
					Return(fmt.Errorf("creation error"))
			},
			wantErr:     true,
			errContains: "creation error",
		},
		{
			name: "pending creation - condition not met",
			setupMocks: func(mockRepo *MockRepository[*v1alpha1.Project, *v1alpha1.ProjectList], mockConv *MockConverter[*regional.WorkspaceDomain, *v1alpha1.Project], workspaceDomain *regional.WorkspaceDomain) {
				prj := &v1alpha1.Project{
					Spec: v1alpha1.ProjectSpec{
						Description: "Test Workspace Description",
						Tags:        []string{"tag1", "tag2"},
						Default:     true,
					},
					Status: v1alpha1.ResourceStatus{
						Phase: v1alpha1.ResourcePhaseCreating,
					},
				}

				mockConv.
					EXPECT().
					FromSECAToAruba(workspaceDomain).
					Return(prj, nil).MaxTimes(1)

				mockRepo.
					EXPECT().
					Create(context.Background(), prj).
					Return(nil).AnyTimes()

				mockRepo.
					EXPECT().
					WaitUntil(context.Background(), prj, gomock.Any()).
					DoAndReturn(func(_ context.Context, _ *v1alpha1.Project, condition func(*v1alpha1.Project) bool) (*v1alpha1.Project, error) {
						if condition(prj) {
							return prj, nil
						}
						return nil, fmt.Errorf("condition not met")
					})
			},
			wantErr:     true,
			errContains: "condition not met",
		},
		{
			name: "success create",
			setupMocks: func(mockRepo *MockRepository[*v1alpha1.Project, *v1alpha1.ProjectList], mockConv *MockConverter[*regional.WorkspaceDomain, *v1alpha1.Project], workspaceDomain *regional.WorkspaceDomain) {
				prj := &v1alpha1.Project{
					Spec: v1alpha1.ProjectSpec{
						Description: "Test Workspace Description",
						Tags:        []string{"tag1", "tag2"},
						Default:     true,
					},
					Status: v1alpha1.ResourceStatus{
						Phase: v1alpha1.ResourcePhaseCreated,
					},
				}

				mockConv.
					EXPECT().
					FromSECAToAruba(workspaceDomain).
					Return(prj, nil).MaxTimes(1)

				mockRepo.
					EXPECT().
					Create(context.Background(), prj).
					Return(nil).AnyTimes()

				mockRepo.
					EXPECT().
					WaitUntil(context.Background(), prj, gomock.Any()).
					DoAndReturn(func(_ context.Context, _ *v1alpha1.Project, condition func(*v1alpha1.Project) bool) (*v1alpha1.Project, error) {
						if condition(prj) {
							return prj, nil
						}
						return nil, fmt.Errorf("condition not met")
					})
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRepo := NewMockRepository[*v1alpha1.Project, *v1alpha1.ProjectList](ctrl)
			mockConv := NewMockConverter[*regional.WorkspaceDomain, *v1alpha1.Project](ctrl)

			wd := &regional.WorkspaceDomain{
				Spec: regional.WorkspaceSpec{
					"description": "Test Workspace Description",
					"tags":        []string{"tag1", "tag2"},
					"default":     true,
				},
			}

			tt.setupMocks(mockRepo, mockConv, wd)

			op := NewWorkspaceHandler(mockRepo, mockConv)

			err := op.Create(context.Background(), wd)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestWorkspace_delete(t *testing.T) {
	tests := []struct {
		name        string
		setupMocks  func(*MockRepository[*v1alpha1.Project, *v1alpha1.ProjectList], *MockConverter[*regional.WorkspaceDomain, *v1alpha1.Project], *regional.WorkspaceDomain)
		wantErr     bool
		errContains string
	}{
		{
			name: "conversion error",
			setupMocks: func(mockRepo *MockRepository[*v1alpha1.Project, *v1alpha1.ProjectList], mockConv *MockConverter[*regional.WorkspaceDomain, *v1alpha1.Project], workspaceDomain *regional.WorkspaceDomain) {
				mockConv.
					EXPECT().
					FromSECAToAruba(workspaceDomain).
					Return(nil, fmt.Errorf("conversion error"))
			},
			wantErr:     true,
			errContains: "conversion error",
		},
		{
			name: "delete error",
			setupMocks: func(mockRepo *MockRepository[*v1alpha1.Project, *v1alpha1.ProjectList], mockConv *MockConverter[*regional.WorkspaceDomain, *v1alpha1.Project], workspaceDomain *regional.WorkspaceDomain) {
				prj := &v1alpha1.Project{
					Spec: v1alpha1.ProjectSpec{
						Description: "Test Workspace Description",
						Tags:        []string{"tag1", "tag2"},
						Default:     true,
					},
				}

				mockConv.
					EXPECT().
					FromSECAToAruba(workspaceDomain).
					Return(prj, nil)

				mockRepo.
					EXPECT().
					Delete(context.Background(), prj).
					Return(fmt.Errorf("deletion error"))
			},
			wantErr:     true,
			errContains: "deletion error",
		},
		{
			name: "pending deletion - condition not met",
			setupMocks: func(mockRepo *MockRepository[*v1alpha1.Project, *v1alpha1.ProjectList], mockConv *MockConverter[*regional.WorkspaceDomain, *v1alpha1.Project], workspaceDomain *regional.WorkspaceDomain) {
				prj := &v1alpha1.Project{
					Spec: v1alpha1.ProjectSpec{
						Description: "Test Workspace Description",
						Tags:        []string{"tag1", "tag2"},
						Default:     true,
					},
					Status: v1alpha1.ResourceStatus{
						Phase: v1alpha1.ResourcePhaseDeleting,
					},
				}

				mockConv.
					EXPECT().
					FromSECAToAruba(workspaceDomain).
					Return(prj, nil).MaxTimes(1)

				mockRepo.
					EXPECT().
					Delete(context.Background(), prj).
					Return(nil).AnyTimes()

				mockRepo.
					EXPECT().
					WaitUntil(context.Background(), prj, gomock.Any()).
					DoAndReturn(func(_ context.Context, _ *v1alpha1.Project, condition func(*v1alpha1.Project) bool) (*v1alpha1.Project, error) {
						if condition(prj) {
							return prj, nil
						}
						return nil, fmt.Errorf("condition not met")
					})
			},
			wantErr:     true,
			errContains: "condition not met",
		},
		{
			name: "success delete",
			setupMocks: func(mockRepo *MockRepository[*v1alpha1.Project, *v1alpha1.ProjectList], mockConv *MockConverter[*regional.WorkspaceDomain, *v1alpha1.Project], workspaceDomain *regional.WorkspaceDomain) {
				prj := &v1alpha1.Project{
					Spec: v1alpha1.ProjectSpec{
						Description: "Test Workspace Description",
						Tags:        []string{"tag1", "tag2"},
						Default:     true,
					},
					Status: v1alpha1.ResourceStatus{
						Phase: v1alpha1.ResourcePhaseDeleted,
					},
				}

				mockConv.
					EXPECT().
					FromSECAToAruba(workspaceDomain).
					Return(prj, nil).MaxTimes(1)

				mockRepo.
					EXPECT().
					Delete(context.Background(), prj).
					Return(nil).AnyTimes()

				mockRepo.
					EXPECT().
					WaitUntil(context.Background(), prj, gomock.Any()).
					DoAndReturn(func(_ context.Context, _ *v1alpha1.Project, condition func(*v1alpha1.Project) bool) (*v1alpha1.Project, error) {
						if condition(prj) {
							return prj, nil
						}
						return nil, fmt.Errorf("condition not met")
					})
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRepo := NewMockRepository[*v1alpha1.Project, *v1alpha1.ProjectList](ctrl)
			mockConv := NewMockConverter[*regional.WorkspaceDomain, *v1alpha1.Project](ctrl)

			wd := &regional.WorkspaceDomain{
				Spec: regional.WorkspaceSpec{
					"description": "Test Workspace Description",
					"tags":        []string{"tag1", "tag2"},
					"default":     true,
				},
			}

			tt.setupMocks(mockRepo, mockConv, wd)

			handler := NewWorkspaceHandler(mockRepo, mockConv)

			err := handler.Delete(context.Background(), wd)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
