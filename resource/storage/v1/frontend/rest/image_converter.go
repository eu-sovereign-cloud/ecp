package rest

import (
	"fmt"
	"strconv"

	sdkstorage "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.storage.v1"
	sdkschema "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	"github.com/eu-sovereign-cloud/ecp/framework/kernel/port/persistence"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel/validation"
	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	commonfrontend "github.com/eu-sovereign-cloud/ecp/resource/common/frontend"
	imgdom "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/image"
)

const (
	// ImageAPIVersion is the API version string used in response metadata.
	ImageAPIVersion = imgdom.Version
	// ImageResource is the resource name.
	ImageResource = imgdom.Resource
)

// ImageIdentity carries identity for a single image resource.
type ImageIdentity struct {
	name            string
	tenant          string
	resourceVersion string
}

func (i *ImageIdentity) GetName() string      { return i.name }
func (i *ImageIdentity) GetVersion() string   { return i.resourceVersion }
func (i *ImageIdentity) GetTenant() string    { return i.tenant }
func (i *ImageIdentity) GetWorkspace() string { return "" }

var _ persistence.IdentifiableResource = (*ImageIdentity)(nil)

// newImageWithIdentity returns a *imgdom.Image populated with identity fields from ir.
func newImageWithIdentity(ir persistence.IdentifiableResource) *imgdom.Image {
	img := &imgdom.Image{}
	img.Name = ir.GetName()
	img.Tenant = ir.GetTenant()
	img.ResourceVersion = ir.GetVersion()
	return img
}

// imageListParamsFromAPI converts SDK ListImagesParams to resource.ListParams.
func imageListParamsFromAPI(params sdkstorage.ListImagesParams, tenant string) resource.ListParams {
	limit := validation.GetLimit(params.Limit)

	var skipToken string
	if params.SkipToken != nil {
		skipToken = *params.SkipToken
	}

	var selector string
	if params.Labels != nil {
		selector = *params.Labels
	}

	return resource.ListParams{
		Scope: resource.Scope{
			Tenant: tenant,
		},
		Limit:     limit,
		SkipToken: skipToken,
		Selector:  selector,
	}
}

// ImageToAPIWithVerb returns a func that converts an Image to its SDK representation with the given verb.
func ImageToAPIWithVerb(verb string) func(img *imgdom.Image) *sdkschema.Image {
	return func(img *imgdom.Image) *sdkschema.Image {
		sdk := imageToAPI(img)
		sdk.Metadata.Verb = verb
		return sdk
	}
}

// imageToAPI converts an Image to its SDK representation.
func imageToAPI(img *imgdom.Image) *sdkschema.Image {
	resourceVersion := int64(0)
	if parsed, err := strconv.ParseInt(img.ResourceVersion, 10, 64); err == nil {
		resourceVersion = parsed
	}

	out := &sdkschema.Image{
		Metadata: &sdkschema.RegionalResourceMetadata{
			ApiVersion:     ImageAPIVersion,
			CreatedAt:      img.CreatedAt,
			LastModifiedAt: img.UpdatedAt,
			Kind:           sdkschema.RegionalResourceMetadataKindResourceKindImage,
			Name:           img.Name,
			Tenant:         img.Tenant,
			Provider:       img.Provider,
			Region:         img.Region,
			Resource:       fmt.Sprintf(commondomain.RegionalResourceFormat, sdkschema.RegionalResourceMetadataKindResourceKindImage, img.Name),
			Ref: fmt.Sprintf(
				img.Provider+"/"+commondomain.RegionalTenantScopedResourceFormat,
				img.Tenant,
				sdkschema.RegionalResourceMetadataKindResourceKindImage,
				img.Name,
			),
			ResourceVersion: resourceVersion,
		},
		Labels:      img.Labels,
		Annotations: img.Annotations,
		Extensions:  img.Extensions,
		Spec: sdkschema.ImageSpec{
			BlockStorageRef: commonfrontend.ReferenceToAPI(img.Spec.BlockStorageRef),
			CpuArchitecture: sdkschema.ImageSpecCpuArchitecture(img.Spec.CpuArchitecture),
			Boot:            sdkschema.ImageSpecBoot(img.Spec.Boot),
			Initializer:     sdkschema.ImageSpecInitializer(img.Spec.Initializer),
		},
	}

	if out.Labels == nil {
		out.Labels = make(sdkschema.Labels)
	}

	if img.Status != nil {
		out.Status = &sdkschema.ImageStatus{
			SizeMB:     img.Status.SizeMB,
			Conditions: commonfrontend.ConditionsToAPI(img.Status.Conditions),
			State:      commonfrontend.ResourceStateToAPI(img.Status.State),
		}
	}
	if img.DeletedAt != nil {
		out.Metadata.DeletedAt = img.DeletedAt
	}
	return out
}

// ImageIteratorToAPI converts a list of Image to an SDK ImageIterator.
func ImageIteratorToAPI(imgs []*imgdom.Image, nextSkipToken *string) *sdkstorage.ImageIterator {
	items := make([]sdkschema.Image, len(imgs))
	for i := range imgs {
		items[i] = *imageToAPI(imgs[i])
	}

	iterator := &sdkstorage.ImageIterator{
		Items: items,
		Metadata: sdkschema.ResponseMetadata{
			Provider: imgdom.ProviderID,
			Resource: ImageResource,
			Verb:     "list",
		},
	}

	if nextSkipToken != nil {
		iterator.Metadata.SkipToken = nextSkipToken
	}

	return iterator
}

// ImageFromAPI converts an SDK Image to an Image.
func ImageFromAPI(sdk sdkschema.Image, id *ImageIdentity, region string) *imgdom.Image {
	img := &imgdom.Image{
		Spec: imgdom.ImageSpec{
			BlockStorageRef: commonfrontend.ReferenceFromAPI(sdk.Spec.BlockStorageRef),
			CpuArchitecture: string(sdk.Spec.CpuArchitecture),
			Boot:            string(sdk.Spec.Boot),
			Initializer:     string(sdk.Spec.Initializer),
		},
	}
	img.Name = id.GetName()
	img.ResourceVersion = id.GetVersion()
	img.Provider = imgdom.ProviderID
	img.Tenant = id.GetTenant()
	img.Region = region
	img.Labels = sdk.Labels
	img.Annotations = sdk.Annotations
	img.Extensions = sdk.Extensions

	return img
}
