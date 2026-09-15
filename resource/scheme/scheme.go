// Package scheme registers the CR types of every resource slice with a runtime.Scheme in one
// call. A delegator passes AddToScheme to builder.NewDelegator, and a test client adds it next
// to the client-go scheme, instead of naming each slice's AddToScheme on its own line.
package scheme

import (
	"k8s.io/apimachinery/pkg/runtime"

	rak8s "github.com/eu-sovereign-cloud/ecp/resource/authorization/v1/role-assignment/backend/kubernetes"
	rolek8s "github.com/eu-sovereign-cloud/ecp/resource/authorization/v1/role/backend/kubernetes"
	instancek8s "github.com/eu-sovereign-cloud/ecp/resource/compute/v1/instance/backend/kubernetes"
	computeskuk8s "github.com/eu-sovereign-cloud/ecp/resource/compute/v1/sku/backend/kubernetes"
	igwk8s "github.com/eu-sovereign-cloud/ecp/resource/network/v1/internet-gateway/backend/kubernetes"
	netskuk8s "github.com/eu-sovereign-cloud/ecp/resource/network/v1/network-sku/backend/kubernetes"
	netk8s "github.com/eu-sovereign-cloud/ecp/resource/network/v1/network/backend/kubernetes"
	nick8s "github.com/eu-sovereign-cloud/ecp/resource/network/v1/nic/backend/kubernetes"
	publicipk8s "github.com/eu-sovereign-cloud/ecp/resource/network/v1/public-ip/backend/kubernetes"
	routetablek8s "github.com/eu-sovereign-cloud/ecp/resource/network/v1/route-table/backend/kubernetes"
	sgrk8s "github.com/eu-sovereign-cloud/ecp/resource/network/v1/security-group-rule/backend/kubernetes"
	sgk8s "github.com/eu-sovereign-cloud/ecp/resource/network/v1/security-group/backend/kubernetes"
	subnetk8s "github.com/eu-sovereign-cloud/ecp/resource/network/v1/subnet/backend/kubernetes"
	rk8s "github.com/eu-sovereign-cloud/ecp/resource/region/v1/backend/kubernetes"
	bsk8s "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/block-storage/backend/kubernetes"
	imgk8s "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/image/backend/kubernetes"
	ssk8s "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/storage-sku/backend/kubernetes"
	wsk8s "github.com/eu-sovereign-cloud/ecp/resource/workspace/v1/backend/kubernetes"
)

// AddToScheme registers every SECA CR type, whether or not a given plugin reconciles it:
// registering a type starts no informer, so a delegator pays nothing for the kinds it never
// touches. A new slice adds its AddToScheme here; TestAddToScheme_CoversEveryCRD fails until it
// does.
var AddToScheme = builder.AddToScheme

var builder = runtime.NewSchemeBuilder(
	rolek8s.AddToScheme,
	rak8s.AddToScheme,
	instancek8s.AddToScheme,
	computeskuk8s.AddToScheme,
	igwk8s.AddToScheme,
	netk8s.AddToScheme,
	netskuk8s.AddToScheme,
	nick8s.AddToScheme,
	publicipk8s.AddToScheme,
	routetablek8s.AddToScheme,
	sgk8s.AddToScheme,
	sgrk8s.AddToScheme,
	subnetk8s.AddToScheme,
	rk8s.AddToScheme,
	bsk8s.AddToScheme,
	imgk8s.AddToScheme,
	ssk8s.AddToScheme,
	wsk8s.AddToScheme,
)
