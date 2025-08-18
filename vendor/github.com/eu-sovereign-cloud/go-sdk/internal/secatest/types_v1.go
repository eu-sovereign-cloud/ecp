package secatest

type MetadataResponseV1 struct {
	Name      string
	Tenant    string
	Workspace string
}
type StatusResponseV1 struct {
	State string
}

// Authorization
type RoleResponseV1 struct {
	Metadata       MetadataResponseV1
	PermissionVerb string
	Status         StatusResponseV1
}
type RoleAssignmentResponseV1 struct {
	Metadata MetadataResponseV1
	Subject  string
	Status   StatusResponseV1
}

// Compute
type InstanceSkuResponseV1 struct {
	Metadata MetadataResponseV1
	Tier     string
	VCPU     int
	Ram      int
}
type InstanceResponseV1 struct {
	Metadata MetadataResponseV1
	SkuRef   string
	Status   StatusResponseV1
}

// Network
type NetworkSkuResponseV1 struct {
	Metadata  MetadataResponseV1
	Tier      string
	Bandwidth int
	Packets   int
}
type NetworkResponseV1 struct {
	Metadata      MetadataResponseV1
	RouteTableRef string
	Status        StatusResponseV1
}
type SubnetResponseV1 struct {
	Metadata MetadataResponseV1
	SkuRef   string
	Status   StatusResponseV1
}
type RouteTableResponseV1 struct {
	Metadata MetadataResponseV1
	LocalRef string
	Status   StatusResponseV1
}
type InternetGatewayResponseV1 struct {
	Metadata   MetadataResponseV1
	EgressOnly bool
	Status     StatusResponseV1
}
type SecurityGroupResponseV1 struct {
	Metadata      MetadataResponseV1
	RuleDirection string
	Status        StatusResponseV1
}
type NicResponseV1 struct {
	Metadata  MetadataResponseV1
	SubnetRef string
	Status    StatusResponseV1
}
type PublicIpResponseV1 struct {
	Metadata MetadataResponseV1
	Address  string
	Status   StatusResponseV1
}

// Region
type RegionResponseV1 struct {
	Metadata  MetadataResponseV1
	Providers []RegionResponseProviderV1
}
type RegionResponseProviderV1 struct {
	Name    string
	URL     string
	Version string
}

// Storage
type StorageSkuResponseV1 struct {
	Metadata MetadataResponseV1
	Tier     string
	Iops     int
}
type BlockStorageResponseV1 struct {
	Metadata MetadataResponseV1
	SkuRef   string
	Status   StatusResponseV1
}
type ImageResponseV1 struct {
	Metadata        MetadataResponseV1
	BlockStorageRef string
	Status          StatusResponseV1
}

// Workspace
type WorkspaceTypeResponseV1 struct {
	Metadata MetadataResponseV1
	Status   StatusResponseV1
}
