package kubernetes

import (
	"k8s.io/apimachinery/pkg/runtime/schema"

	routetablek8s "github.com/eu-sovereign-cloud/ecp/resource/network/v1/route-table/backend/kubernetes"
	subnetk8s "github.com/eu-sovereign-cloud/ecp/resource/network/v1/subnet/backend/kubernetes"
)

// ChildResourceGVRs is the closed set of SECA types that live in the namespace a Network owns
// for its children. It gates two things: the gateway's refusal to delete a non-empty Network
// (409), and the controller's re-check before it deletes that namespace.
//
// It lives next to the owner rather than at each call site because a type missing from this list
// makes the namespace look empty when it is not, and the namespace delete cascades. Adding a
// network-scoped resource means adding it here.
var ChildResourceGVRs = []schema.GroupVersionResource{
	routetablek8s.RouteTableGVR,
	subnetk8s.SubnetGVR,
}
