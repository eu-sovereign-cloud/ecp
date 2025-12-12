package storage

import (
	"encoding/json"
	"fmt"
	"strings"

	sdkschema "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	genv1 "github.com/eu-sovereign-cloud/ecp/foundation/api/generated/types"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
)

// BlockStorageToAPI converts a BlockStorageDomain to its SDK representation.
func BlockStorageToAPI(domain *regional.BlockStorageDomain) *sdkschema.BlockStorage {
	bs := &sdkschema.BlockStorage{
		Spec: sdkschema.BlockStorageSpec{
			SizeGB:         domain.Spec.SizeGB,
			SkuRef:         convertReferenceToSDK(domain.Spec.SkuRef),
			SourceImageRef: convertReferencePointerToSDK(domain.Spec.SourceImageRef),
		},
	}

	if domain.Labels != nil {
		bs.Labels = domain.Labels
	}

	if domain.Status != nil {
		bs.Status = &sdkschema.BlockStorageStatus{
			SizeGB:     domain.Status.SizeGB,
			Conditions: convertStatusConditionsToSDK(domain.Status.Conditions),
			AttachedTo: convertReferencePointerToSDK(domain.Status.AttachedTo),
			State:      convertResourceStatePointerToSDK(domain.Status.State),
		}
	}

	return bs
}

// BlockStorageFromAPI converts an SDK BlockStorage to the domain model.
func BlockStorageFromAPI(api *sdkschema.BlockStorage, tenant, name string) *regional.BlockStorageDomain {
	// Sanitize inputs: trim whitespace/newlines from name and tenant
	cleanTenant := strings.TrimSpace(tenant)
	cleanName := strings.TrimSpace(name)
	refObj, _ := convertReferenceFromSDK(api.Spec.SkuRef)
	domain := &regional.BlockStorageDomain{
		Spec: regional.BlockStorageSpec{
			SizeGB:         api.Spec.SizeGB,
			SkuRef:         refObj,
			SourceImageRef: convertReferencePointerFromSDK(api.Spec.SourceImageRef),
		},
	}

	domain.SetName(cleanName)
	domain.SetNamespace(cleanTenant) // In K8s, tenant is the namespace

	// Handle labels and annotations if present
	if api.Labels != nil {
		domain.Labels = api.Labels
	}

	// Status is typically not provided in create/update requests
	if api.Status != nil {
		domain.Status = &regional.BlockStorageStatus{
			SizeGB:     api.Status.SizeGB,
			Conditions: convertStatusConditionsFromSDK(api.Status.Conditions),
			AttachedTo: convertReferencePointerFromSDK(api.Status.AttachedTo),
			State:      convertResourceStatePointerFromSDK(api.Status.State),
		}
	}

	return domain
}

// Helper functions to convert between SDK and generated types
func convertReferenceToSDK(ref genv1.Reference) sdkschema.Reference {
	// Both types have the same structure, so we can marshal/unmarshal
	var sdkRef sdkschema.Reference
	data, _ := json.Marshal(ref)
	_ = json.Unmarshal(data, &sdkRef)
	return sdkRef
}

func convertReferenceFromSDK(ref sdkschema.Reference) (genv1.Reference, error) {
	// Try to obtain the object variant via SDK helper
	if obj, err := ref.AsReferenceObject(); err == nil {
		// Convert SDK object to generated object via JSON
		var genRef genv1.Reference
		data, _ := json.Marshal(obj)
		_ = json.Unmarshal(data, &genRef)
		return genRef, nil
	}
	// Fallback: it’s the string union (URN). Wrap into object form.
	if urn, err := ref.AsReferenceURN(); err == nil {
		wrapped := genv1.ReferenceObject{
			Provider:  nil,
			Region:    nil,
			Resource:  urn,
			Tenant:    nil,
			Workspace: nil,
		}
		var genRef genv1.Reference
		data, _ := json.Marshal(wrapped)
		_ = json.Unmarshal(data, &genRef)
		return genRef, nil
	}
	// If neither variant works, return empty with error
	return genv1.Reference{}, fmt.Errorf("unsupported reference variant")
}

func convertReferencePointerToSDK(ref *genv1.Reference) *sdkschema.Reference {
	if ref == nil {
		return nil
	}
	result := convertReferenceToSDK(*ref)
	return &result
}

func convertReferencePointerFromSDK(ref *sdkschema.Reference) *genv1.Reference {
	if ref == nil {
		return nil
	}
	result, _ := convertReferenceFromSDK(*ref)
	return &result
}

func convertStatusConditionsToSDK(conds []genv1.StatusCondition) []sdkschema.StatusCondition {
	result := make([]sdkschema.StatusCondition, len(conds))
	for i, c := range conds {
		data, _ := json.Marshal(c)
		_ = json.Unmarshal(data, &result[i])
	}
	return result
}

func convertStatusConditionsFromSDK(conds []sdkschema.StatusCondition) []genv1.StatusCondition {
	result := make([]genv1.StatusCondition, len(conds))
	for i, c := range conds {
		data, _ := json.Marshal(c)
		_ = json.Unmarshal(data, &result[i])
	}
	return result
}

func convertResourceStatePointerToSDK(state *genv1.ResourceState) *sdkschema.ResourceState {
	if state == nil {
		return nil
	}
	result := sdkschema.ResourceState(*state)
	return &result
}

func convertResourceStatePointerFromSDK(state *sdkschema.ResourceState) *genv1.ResourceState {
	if state == nil {
		return nil
	}
	result := genv1.ResourceState(*state)
	return &result
}
