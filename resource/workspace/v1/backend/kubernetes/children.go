package kubernetes

import (
	"k8s.io/apimachinery/pkg/runtime/schema"

	instancek8s "github.com/eu-sovereign-cloud/ecp/resource/compute/v1/instance/backend/kubernetes"
	internetgatewayk8s "github.com/eu-sovereign-cloud/ecp/resource/network/v1/internet-gateway/backend/kubernetes"
	netk8s "github.com/eu-sovereign-cloud/ecp/resource/network/v1/network/backend/kubernetes"
	nick8s "github.com/eu-sovereign-cloud/ecp/resource/network/v1/nic/backend/kubernetes"
	publicipk8s "github.com/eu-sovereign-cloud/ecp/resource/network/v1/public-ip/backend/kubernetes"
	securitygrouprulek8s "github.com/eu-sovereign-cloud/ecp/resource/network/v1/security-group-rule/backend/kubernetes"
	securitygroupk8s "github.com/eu-sovereign-cloud/ecp/resource/network/v1/security-group/backend/kubernetes"
	bsk8s "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/block-storage/backend/kubernetes"
	imgk8s "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/image/backend/kubernetes"
)

// ChildResourceGVRs is the closed set of SECA types that live in the namespace a Workspace owns
// for its children. It gates two things: the gateway's refusal to delete a non-empty Workspace
// (409), and the controller's re-check before it deletes that namespace.
//
// It lives next to the owner rather than at each call site because a type missing from this list
// makes the namespace look empty when it is not, and the namespace delete cascades. Adding a
// workspace-scoped resource means adding it here.
//
// Subnet and RouteTable are absent on purpose: they are network-scoped and live in the Network's
// own child namespace, not the workspace one. See network's ChildResourceGVRs.
var ChildResourceGVRs = []schema.GroupVersionResource{
	bsk8s.BlockStorageGVR,
	imgk8s.ImageGVR,
	netk8s.NetworkGVR,
	nick8s.NICGVR,
	publicipk8s.PublicIPGVR,
	internetgatewayk8s.InternetGatewayGVR,
	securitygroupk8s.SecurityGroupGVR,
	securitygrouprulek8s.SecurityGroupRuleGVR,
	instancek8s.InstanceGVR,
}
