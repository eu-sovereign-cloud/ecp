package regional

import "github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model"

type NetworkSKUDomain struct {
	Meta model.Metadata
	Spec NetworkSKUSpec
}

type NetworkSKUSpec struct {
	Bandwidth int
	Packets   int
}

func (n *NetworkSKUDomain) GetName() string        { return n.Meta.Name }
func (n *NetworkSKUDomain) GetNamespace() string   { return n.Meta.Namespace }
func (n *NetworkSKUDomain) SetName(name string)    { n.Meta.Name = name }
func (n *NetworkSKUDomain) SetNamespace(ns string) { n.Meta.Namespace = ns }
