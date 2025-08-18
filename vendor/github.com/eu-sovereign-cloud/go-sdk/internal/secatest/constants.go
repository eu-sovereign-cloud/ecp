package secatest

const (
	// Providers
	ProviderRegionName     = "seca.region/v1"
	ProviderRegionEndpoint = "/providers/seca.regions"

	ProviderAuthorizationName     = "seca.authorization"
	ProviderAuthorizationEndpoint = "/providers/seca.authorization"

	ProviderWorkspaceName     = "seca.workspace"
	ProviderWorkspaceEndpoint = "/providers/seca.workspace"

	ProviderNetworkName     = "seca.network"
	ProviderNetworkEndpoint = "/providers/seca.network"

	ProviderComputeName     = "seca.compute"
	ProviderComputeEndpoint = "/providers/seca.compute"

	ProviderStorageName     = "seca.storage"
	ProviderStorageEndpoint = "/providers/seca.storage"

	ProviderVersion1 = "v1"

	// Test Data

	AuthToken = "test-token"

	/// Metadata
	Tenant1Name    = "tenant-1"
	Workspace1Name = "workspace-1"
	Region1Name    = "region-1"
	ZoneA          = "a"

	// Labels
	LabelKeyTier = "tier"

	/// Network
	NetworkSku1Name      = "sku-1"
	NetworkSku1Tier      = "N1K"
	NetworkSku1Bandwidth = 1000
	NetworkSku1Packets   = 100
	NetworkSku1Ref       = "skus/sku-1"

	Network1Name = "network-1"
	Network1Ref  = "networks/network-1"

	Subnet1Name = "subnet-1"
	Subnet1Ref  = "subnets/subnet-1"

	RouteTable1Name = "route-table-1"
	RouteTable1Ref  = "route-tables/route-table-1"

	InternetGateway1Name = "internet-gateway-1"

	SecurityGroup1Name     = "security-group-1"
	SecurityGroup1PortFrom = 80
	SecurityGroup1PortTo   = 80

	Nic1Name = "nic-1"

	PublicIp1Name = "public-ip-1"

	CidrIpv4 = "0.0.0.0/16"

	Address1 = "0.0.0.0"

	SecurityGroupRuleDirectionIngress = "ingress"

	/// Compute
	InstanceSku1Ref  = "skus/sku-1"
	InstanceSku1Name = "sku-1"
	InstanceSku1Tier = "D2XS"
	InstanceSku1VCPU = 16
	InstanceSku1RAM  = 32

	Instance1Ref  = "instances/instance-1"
	Instance1Name = "instance-1"

	/// Authorization
	Role1Name               = "role-1"
	Role1PermissionProvider = "seca.compute"
	Role1PermissionResource = "instances/*"
	Role1PermissionVerb     = "get"

	RoleAssignment1Name    = "role-assignment-1"
	RoleAssignment1Subject = "sub@secapi.com"

	/// Storage
	StorageSku1Ref  = "storage/skus-1"
	StorageSku1Name = "sku-1"
	StorageSku1Tier = "DXS"
	StorageSku1Iops = 100

	BlockStorage1Ref    = "storages/storage-1"
	BlockStorage1Name   = "storage-1"
	BlockStorage1SizeGB = 10

	Image1Name    = "image-1"
	Image1CpuArch = "amd64"

	/// Status
	StatusStateActive   = "active"
	StatusStateCreating = "creating"
)
