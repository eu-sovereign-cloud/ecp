package handler

import (
	"context"
	"fmt"
	"testing"

	"github.com/Arubacloud/arubacloud-resource-operator/api/v1alpha1"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestBlockStorage_create(t *testing.T) {
	tests := []struct {
		name        string
		setupMocks  func(*MockRepository[*v1alpha1.BlockStorage, *v1alpha1.BlockStorageList], *MockConverter[*regional.BlockStorageDomain, *v1alpha1.BlockStorage], *regional.BlockStorageDomain)
		wantErr     bool
		errContains string
	}{
		{
			name: "conversion error",
			setupMocks: func(mockRepo *MockRepository[*v1alpha1.BlockStorage, *v1alpha1.BlockStorageList], mockConv *MockConverter[*regional.BlockStorageDomain, *v1alpha1.BlockStorage], blockStorageDomain *regional.BlockStorageDomain) {
				mockConv.
					EXPECT().
					FromSECAToAruba(blockStorageDomain).
					Return(nil, fmt.Errorf("conversion error"))
			},
			wantErr:     true,
			errContains: "conversion error",
		},
		{
			name: "create error",
			setupMocks: func(mockRepo *MockRepository[*v1alpha1.BlockStorage, *v1alpha1.BlockStorageList], mockConv *MockConverter[*regional.BlockStorageDomain, *v1alpha1.BlockStorage], blockStorageDomain *regional.BlockStorageDomain) {
				prj := &v1alpha1.BlockStorage{
					Spec: v1alpha1.BlockStorageSpec{
						Tenant: uuid.NewString(),
						SizeGb: 100,
						Tags:   []string{"tag1", "tag2"},
					},
				}

				mockConv.
					EXPECT().
					FromSECAToAruba(blockStorageDomain).
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
			setupMocks: func(mockRepo *MockRepository[*v1alpha1.BlockStorage, *v1alpha1.BlockStorageList], mockConv *MockConverter[*regional.BlockStorageDomain, *v1alpha1.BlockStorage], blockStorageDomain *regional.BlockStorageDomain) {
				bs := &v1alpha1.BlockStorage{
					Spec: v1alpha1.BlockStorageSpec{
						Tenant: uuid.NewString(),
						SizeGb: 100,
						Tags:   []string{"tag1", "tag2"},
					},
					Status: v1alpha1.BlockStorageStatus{
						ResourceStatus: v1alpha1.ResourceStatus{
							Phase: v1alpha1.ResourcePhaseCreating,
						},
					},
				}

				mockConv.
					EXPECT().
					FromSECAToAruba(blockStorageDomain).
					Return(bs, nil).MaxTimes(1)

				mockRepo.
					EXPECT().
					Create(context.Background(), bs).
					Return(nil).AnyTimes()

				mockRepo.
					EXPECT().
					WaitUntil(context.Background(), bs, gomock.Any()).
					DoAndReturn(func(_ context.Context, _ *v1alpha1.BlockStorage, condition func(*v1alpha1.BlockStorage) bool) (*v1alpha1.BlockStorage, error) {
						if condition(bs) {
							return bs, nil
						}
						return nil, fmt.Errorf("condition not met")
					})
			},
			wantErr:     true,
			errContains: "condition not met",
		},
		{
			name: "success create",
			setupMocks: func(mockRepo *MockRepository[*v1alpha1.BlockStorage, *v1alpha1.BlockStorageList], mockConv *MockConverter[*regional.BlockStorageDomain, *v1alpha1.BlockStorage], blockStorageDomain *regional.BlockStorageDomain) {
				bs := &v1alpha1.BlockStorage{
					Spec: v1alpha1.BlockStorageSpec{
						Tenant: uuid.NewString(),
						SizeGb: 100,
						Tags:   []string{"tag1", "tag2"},
					},
					Status: v1alpha1.BlockStorageStatus{
						ResourceStatus: v1alpha1.ResourceStatus{
							Phase: v1alpha1.ResourcePhaseCreated,
						},
					},
				}

				mockConv.
					EXPECT().
					FromSECAToAruba(blockStorageDomain).
					Return(bs, nil).MaxTimes(1)

				mockRepo.
					EXPECT().
					Create(context.Background(), bs).
					Return(nil).AnyTimes()

				mockRepo.
					EXPECT().
					WaitUntil(context.Background(), bs, gomock.Any()).
					DoAndReturn(func(_ context.Context, _ *v1alpha1.BlockStorage, condition func(*v1alpha1.BlockStorage) bool) (*v1alpha1.BlockStorage, error) {
						if condition(bs) {
							return bs, nil
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

			mockRepo := NewMockRepository[*v1alpha1.BlockStorage, *v1alpha1.BlockStorageList](ctrl)
			mockConv := NewMockConverter[*regional.BlockStorageDomain, *v1alpha1.BlockStorage](ctrl)

			bd := &regional.BlockStorageDomain{
				Spec: regional.BlockStorageSpec{
					SizeGB: 100,
					SkuRef: regional.ReferenceObject{
						Tenant: "sku-id",
					},
				},
			}

			tt.setupMocks(mockRepo, mockConv, bd)

			op := NewBlockStorageHandler(mockRepo, mockConv)

			err := op.Create(context.Background(), bd)
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

func TestBlockStorage_delete(t *testing.T) {
	tests := []struct {
		name        string
		setupMocks  func(*MockRepository[*v1alpha1.BlockStorage, *v1alpha1.BlockStorageList], *MockConverter[*regional.BlockStorageDomain, *v1alpha1.BlockStorage], *regional.BlockStorageDomain)
		wantErr     bool
		errContains string
	}{
		{
			name: "conversion error",
			setupMocks: func(mockRepo *MockRepository[*v1alpha1.BlockStorage, *v1alpha1.BlockStorageList], mockConv *MockConverter[*regional.BlockStorageDomain, *v1alpha1.BlockStorage], blockStorageDomain *regional.BlockStorageDomain) {
				mockConv.
					EXPECT().
					FromSECAToAruba(blockStorageDomain).
					Return(nil, fmt.Errorf("conversion error"))
			},
			wantErr:     true,
			errContains: "conversion error",
		},
		{
			name: "delete error",
			setupMocks: func(mockRepo *MockRepository[*v1alpha1.BlockStorage, *v1alpha1.BlockStorageList], mockConv *MockConverter[*regional.BlockStorageDomain, *v1alpha1.BlockStorage], blockStorageDomain *regional.BlockStorageDomain) {
				bs := &v1alpha1.BlockStorage{
					Spec: v1alpha1.BlockStorageSpec{
						SizeGb: 100,
						Location: v1alpha1.Location{
							Value: "ITBG-1",
						},
					},
				}

				mockConv.
					EXPECT().
					FromSECAToAruba(blockStorageDomain).
					Return(bs, nil)

				mockRepo.
					EXPECT().
					Delete(context.Background(), bs).
					Return(fmt.Errorf("deletion error"))
			},
			wantErr:     true,
			errContains: "deletion error",
		},
		{
			name: "pending deletion - condition not met",
			setupMocks: func(mockRepo *MockRepository[*v1alpha1.BlockStorage, *v1alpha1.BlockStorageList], mockConv *MockConverter[*regional.BlockStorageDomain, *v1alpha1.BlockStorage], blockStorageDomain *regional.BlockStorageDomain) {
				bs := &v1alpha1.BlockStorage{
					Spec: v1alpha1.BlockStorageSpec{
						SizeGb: 100,
						Location: v1alpha1.Location{
							Value: "ITBG-1",
						},
						Tenant: uuid.NewString(),
					},
					Status: v1alpha1.BlockStorageStatus{
						ResourceStatus: v1alpha1.ResourceStatus{
							Phase: v1alpha1.ResourcePhaseDeleting,
						},
					},
				}

				mockConv.
					EXPECT().
					FromSECAToAruba(blockStorageDomain).
					Return(bs, nil).MaxTimes(1)

				mockRepo.
					EXPECT().
					Delete(context.Background(), bs).
					Return(nil).AnyTimes()

				mockRepo.
					EXPECT().
					WaitUntil(context.Background(), bs, gomock.Any()).
					DoAndReturn(func(_ context.Context, _ *v1alpha1.BlockStorage, condition func(*v1alpha1.BlockStorage) bool) (*v1alpha1.BlockStorage, error) {
						if condition(bs) {
							return bs, nil
						}
						return nil, fmt.Errorf("condition not met")
					})
			},
			wantErr:     true,
			errContains: "condition not met",
		},
		{
			name: "success delete",
			setupMocks: func(mockRepo *MockRepository[*v1alpha1.BlockStorage, *v1alpha1.BlockStorageList], mockConv *MockConverter[*regional.BlockStorageDomain, *v1alpha1.BlockStorage], blockStorageDomain *regional.BlockStorageDomain) {
				bs := &v1alpha1.BlockStorage{
					Spec: v1alpha1.BlockStorageSpec{
						SizeGb: 100,
						Location: v1alpha1.Location{
							Value: "ITBG-1",
						},
						Tenant: uuid.NewString(),
					},
					Status: v1alpha1.BlockStorageStatus{
						ResourceStatus: v1alpha1.ResourceStatus{
							Phase: v1alpha1.ResourcePhaseDeleted,
						},
					},
				}

				mockConv.
					EXPECT().
					FromSECAToAruba(blockStorageDomain).
					Return(bs, nil).MaxTimes(1)

				mockRepo.
					EXPECT().
					Delete(context.Background(), bs).
					Return(nil).AnyTimes()

				mockRepo.
					EXPECT().
					WaitUntil(context.Background(), bs, gomock.Any()).
					DoAndReturn(func(_ context.Context, _ *v1alpha1.BlockStorage, condition func(*v1alpha1.BlockStorage) bool) (*v1alpha1.BlockStorage, error) {
						if condition(bs) {
							return bs, nil
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

			mockRepo := NewMockRepository[*v1alpha1.BlockStorage, *v1alpha1.BlockStorageList](ctrl)
			mockConv := NewMockConverter[*regional.BlockStorageDomain, *v1alpha1.BlockStorage](ctrl)

			bd := &regional.BlockStorageDomain{
				Spec: regional.BlockStorageSpec{
					SizeGB: 100,
					SkuRef: regional.ReferenceObject{
						Tenant: "sku-id",
					},
				},
			}

			tt.setupMocks(mockRepo, mockConv, bd)

			handler := NewBlockStorageHandler(mockRepo, mockConv)

			err := handler.Delete(context.Background(), bd)

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

func TestBlockStorage_increaseSize(t *testing.T) {
	tests := []struct {
		name        string
		setupMocks  func(*MockRepository[*v1alpha1.BlockStorage, *v1alpha1.BlockStorageList], *MockConverter[*regional.BlockStorageDomain, *v1alpha1.BlockStorage], *regional.BlockStorageDomain)
		wantErr     bool
		errContains string
	}{
		{
			name: "increase size success",
			setupMocks: func(mockRepo *MockRepository[*v1alpha1.BlockStorage, *v1alpha1.BlockStorageList], mockConv *MockConverter[*regional.BlockStorageDomain, *v1alpha1.BlockStorage], blockStorageDomain *regional.BlockStorageDomain) {
				bs := &v1alpha1.BlockStorage{
					Spec: v1alpha1.BlockStorageSpec{
						SizeGb: 200,
						Location: v1alpha1.Location{
							Value: "ITBG-1",
						},
						Tenant: uuid.NewString(),
					},
					Status: v1alpha1.BlockStorageStatus{
						ResourceStatus: v1alpha1.ResourceStatus{
							Phase: v1alpha1.ResourcePhaseCreated,
						},
					},
				}

				mockConv.
					EXPECT().
					FromSECAToAruba(blockStorageDomain).
					Return(bs, nil).MaxTimes(1)

				mockRepo.
					EXPECT().
					Update(context.Background(), bs).
					Return(nil).AnyTimes()

				mockRepo.
					EXPECT().
					WaitUntil(context.Background(), bs, gomock.Any()).
					DoAndReturn(func(_ context.Context, _ *v1alpha1.BlockStorage, condition func(*v1alpha1.BlockStorage) bool) (*v1alpha1.BlockStorage, error) {
						if condition(bs) {
							return bs, nil
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

			mockRepo := NewMockRepository[*v1alpha1.BlockStorage, *v1alpha1.BlockStorageList](ctrl)
			mockConv := NewMockConverter[*regional.BlockStorageDomain, *v1alpha1.BlockStorage](ctrl)

			bd := &regional.BlockStorageDomain{
				Spec: regional.BlockStorageSpec{
					SizeGB: 200, // New size for increase
					SkuRef: regional.ReferenceObject{
						Tenant: "sku-id",
					},
				},
			}

			tt.setupMocks(mockRepo, mockConv, bd)

			handler := NewBlockStorageHandler(mockRepo, mockConv)

			err := handler.IncreaseSize(context.Background(), bd)

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
