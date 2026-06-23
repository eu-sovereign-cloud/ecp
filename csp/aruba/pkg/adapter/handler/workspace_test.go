package handler

import (
	"context"
	"fmt"
	"testing"

	"github.com/Arubacloud/arubacloud-resource-operator/api/v1alpha1"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"

	wsdom "github.com/eu-sovereign-cloud/ecp/resource/workspace/v1"
)

func notFoundErr(name string) error {
	return errors.NewNotFound(schema.GroupResource{Group: "your.group", Resource: "your-resource"}, name)
}

func alreadyExistsErr(name string) error {
	return errors.NewAlreadyExists(schema.GroupResource{Group: "your.group", Resource: "your-resource"}, name)
}

func TestWorkspace_create(t *testing.T) {
	tests := []struct {
		name        string
		setupMocks  func(*MockRepository[*v1alpha1.Project, *v1alpha1.ProjectList], *MockConverter[*wsdom.Workspace, *v1alpha1.Project], *wsdom.Workspace)
		wantErr     bool
		errContains string
	}{
		{
			name: "conversion error",
			setupMocks: func(mockRepo *MockRepository[*v1alpha1.Project, *v1alpha1.ProjectList], mockConv *MockConverter[*wsdom.Workspace, *v1alpha1.Project], workspaceDomain *wsdom.Workspace) {
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
			setupMocks: func(mockRepo *MockRepository[*v1alpha1.Project, *v1alpha1.ProjectList], mockConv *MockConverter[*wsdom.Workspace, *v1alpha1.Project], workspaceDomain *wsdom.Workspace) {
				prj := &v1alpha1.Project{
					Spec: v1alpha1.ProjectSpec{
						Description: "Test Workspace Description",
						Tags:        []string{"tag1", "tag2"},
					},
				}

				mockConv.
					EXPECT().
					FromSECAToAruba(workspaceDomain).
					Return(prj, nil)

				// Not created yet, so the check reports "not present".
				mockRepo.
					EXPECT().
					Load(gomock.Any(), gomock.Any()).
					Return(notFoundErr(workspaceDomain.Name)).
					AnyTimes()

				mockRepo.
					EXPECT().
					Create(context.Background(), prj).
					Return(fmt.Errorf("creation error"))
			},
			wantErr:     true,
			errContains: "creation error",
		},
		{
			name: "pending creation - still processing",
			setupMocks: func(mockRepo *MockRepository[*v1alpha1.Project, *v1alpha1.ProjectList], mockConv *MockConverter[*wsdom.Workspace, *v1alpha1.Project], workspaceDomain *wsdom.Workspace) {
				prj := &v1alpha1.Project{
					Spec: v1alpha1.ProjectSpec{
						Description: "Test Workspace Description",
						Tags:        []string{"tag1", "tag2"},
					},
					Status: v1alpha1.ResourceStatus{
						Phase: v1alpha1.ResourcePhaseCreating,
					},
				}

				mockConv.
					EXPECT().
					FromSECAToAruba(workspaceDomain).
					Return(prj, nil).MaxTimes(1)

				// Present but not yet active, so the check reports "not done".
				mockRepo.
					EXPECT().
					Load(gomock.Any(), gomock.Any()).
					Return(nil).
					AnyTimes()

				mockRepo.
					EXPECT().
					Create(context.Background(), prj).
					Return(nil).AnyTimes()
			},
			wantErr:     true,
			errContains: "operation still in progress",
		},
		{
			name: "success create",
			setupMocks: func(mockRepo *MockRepository[*v1alpha1.Project, *v1alpha1.ProjectList], mockConv *MockConverter[*wsdom.Workspace, *v1alpha1.Project], workspaceDomain *wsdom.Workspace) {
				prj := &v1alpha1.Project{
					Spec: v1alpha1.ProjectSpec{
						Description: "Test Workspace Description",
						Tags:        []string{"tag1", "tag2"},
					},
					Status: v1alpha1.ResourceStatus{
						Phase: v1alpha1.ResourcePhaseActive,
					},
				}

				mockConv.
					EXPECT().
					FromSECAToAruba(workspaceDomain).
					Return(prj, nil).MaxTimes(1)

				// Present and active, so the check reports "done".
				mockRepo.
					EXPECT().
					Load(gomock.Any(), gomock.Any()).
					Return(nil).
					AnyTimes()

				mockRepo.
					EXPECT().
					Create(context.Background(), prj).
					Return(nil).AnyTimes()
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRepo := NewMockRepository[*v1alpha1.Project, *v1alpha1.ProjectList](ctrl)
			mockConv := NewMockConverter[*wsdom.Workspace, *v1alpha1.Project](ctrl)

			wd := &wsdom.Workspace{
				Spec: wsdom.WorkspaceSpec{
					"description": "Test Workspace Description",
					"tags":        []string{"tag1", "tag2"},
					"default":     true,
				},
			}

			tt.setupMocks(mockRepo, mockConv, wd)

			op := NewWorkspaceHandler(mockRepo, mockConv)

			err := op.Create(context.Background(), wd)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					require.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestWorkspace_delete(t *testing.T) {
	tests := []struct {
		name        string
		setupMocks  func(*MockRepository[*v1alpha1.Project, *v1alpha1.ProjectList], *MockConverter[*wsdom.Workspace, *v1alpha1.Project], *wsdom.Workspace)
		wantErr     bool
		errContains string
	}{
		{
			name: "conversion error",
			setupMocks: func(mockRepo *MockRepository[*v1alpha1.Project, *v1alpha1.ProjectList], mockConv *MockConverter[*wsdom.Workspace, *v1alpha1.Project], workspaceDomain *wsdom.Workspace) {
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
			setupMocks: func(mockRepo *MockRepository[*v1alpha1.Project, *v1alpha1.ProjectList], mockConv *MockConverter[*wsdom.Workspace, *v1alpha1.Project], workspaceDomain *wsdom.Workspace) {
				prj := &v1alpha1.Project{
					Spec: v1alpha1.ProjectSpec{
						Description: "Test Workspace Description",
						Tags:        []string{"tag1", "tag2"},
					},
				}

				mockConv.
					EXPECT().
					FromSECAToAruba(workspaceDomain).
					Return(prj, nil)

				// Still present, so the check reports "not done".
				mockRepo.EXPECT().
					Load(gomock.Any(), gomock.Any()).
					Return(nil).
					AnyTimes()

				mockRepo.
					EXPECT().
					Delete(context.Background(), prj).
					Return(fmt.Errorf("deletion error"))
			},
			wantErr:     true,
			errContains: "deletion error",
		},
		{
			name: "pending deletion - still processing",
			setupMocks: func(mockRepo *MockRepository[*v1alpha1.Project, *v1alpha1.ProjectList], mockConv *MockConverter[*wsdom.Workspace, *v1alpha1.Project], workspaceDomain *wsdom.Workspace) {
				prj := &v1alpha1.Project{
					Spec: v1alpha1.ProjectSpec{
						Description: "Test Workspace Description",
						Tags:        []string{"tag1", "tag2"},
					},
					Status: v1alpha1.ResourceStatus{
						Phase: v1alpha1.ResourcePhaseDeleting,
					},
				}

				mockConv.
					EXPECT().
					FromSECAToAruba(workspaceDomain).
					Return(prj, nil).MaxTimes(1)

				// Still present, so the check reports "not done".
				mockRepo.EXPECT().
					Load(gomock.Any(), gomock.Any()).
					Return(nil).
					AnyTimes()

				mockRepo.
					EXPECT().
					Delete(context.Background(), prj).
					Return(nil).AnyTimes()
			},
			wantErr:     true,
			errContains: "operation still in progress",
		},
		{
			name: "success delete",
			setupMocks: func(mockRepo *MockRepository[*v1alpha1.Project, *v1alpha1.ProjectList], mockConv *MockConverter[*wsdom.Workspace, *v1alpha1.Project], workspaceDomain *wsdom.Workspace) {
				prj := &v1alpha1.Project{
					Spec: v1alpha1.ProjectSpec{
						Description: "Test Workspace Description",
						Tags:        []string{"tag1", "tag2"},
					},
					Status: v1alpha1.ResourceStatus{
						Phase: v1alpha1.ResourcePhaseDeleted,
					},
				}

				mockConv.
					EXPECT().
					FromSECAToAruba(workspaceDomain).
					Return(prj, nil).MaxTimes(1)

				// Gone, so the check reports "done".
				mockRepo.EXPECT().
					Load(gomock.Any(), gomock.Any()).
					Return(notFoundErr(workspaceDomain.Name)).
					AnyTimes()

				mockRepo.
					EXPECT().
					Delete(context.Background(), prj).
					Return(nil).AnyTimes()
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRepo := NewMockRepository[*v1alpha1.Project, *v1alpha1.ProjectList](ctrl)
			mockConv := NewMockConverter[*wsdom.Workspace, *v1alpha1.Project](ctrl)

			wd := &wsdom.Workspace{
				Spec: wsdom.WorkspaceSpec{
					"description": "Test Workspace Description",
					"tags":        []string{"tag1", "tag2"},
					"default":     true,
				},
			}

			tt.setupMocks(mockRepo, mockConv, wd)

			handler := NewWorkspaceHandler(mockRepo, mockConv)

			err := handler.Delete(context.Background(), wd)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					require.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}
