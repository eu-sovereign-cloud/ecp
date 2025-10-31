package secatest

const (
	// Network Sku
	networkSkusResponseTemplateV1 = `
	{
		"items": [
        	{
				"metadata": {
					"name": "{{.Metadata.Name}}",
					"tenant": "{{.Metadata.Tenant}}"
				},
				"labels": {
					"tier": "{{.Tier}}"
				},
				"spec": {
					"bandwidth": {{.Bandwidth}},
					"packets": {{.Packets}}
				}
			}
    	]
	}`
	networkSkuResponseTemplateV1 = `
	{
		"metadata": {
			"name": "{{.Metadata.Name}}",
			"tenant": "{{.Metadata.Tenant}}"
		},
		"labels": {
					"tier": "{{.Tier}}"
		},
		"spec": {
			"bandwidth": {{.Bandwidth}},
			"packets": {{.Packets}}
		}
	}`

	// Network
	networksResponseTemplateV1 = `
	{
		"items": [
        	{
				"metadata": {
					"name": "{{.Metadata.Name}}",
					"tenant": "{{.Metadata.Tenant}}",
					"workspace": "{{.Metadata.Workspace}}"
				},
				"spec": {
        			"routeTableRef": "{{.RouteTableRef}}"
				},
				"status": {
					"state": "{{.Status.State}}"
				}
			}
    	]
	}`
	networkResponseTemplateV1 = `
	{
		"metadata": {
			"name": "{{.Metadata.Name}}",
			"tenant": "{{.Metadata.Tenant}}",
			"workspace": "{{.Metadata.Workspace}}"
		},
		"spec": {
			"routeTableRef": "{{.RouteTableRef}}"
		},
		"status": {
			"state": "{{.Status.State}}"
		}
	}`

	// Subnet
	subnetsResponseTemplateV1 = `
	{
		"items": [
        	{
				"metadata": {
					"name": "{{.Metadata.Name}}",
					"tenant": "{{.Metadata.Tenant}}",
					"workspace": "{{.Metadata.Workspace}}",
					"network": "{{.Metadata.Network}}"
				},
				"spec": {
					"skuRef": "{{.SkuRef}}"
				},
				"status": {
					"state": "{{.Status.State}}"
				}
			}
    	]
	}`
	subnetResponseTemplateV1 = `
	{
		"metadata": {
			"name": "{{.Metadata.Name}}",
			"tenant": "{{.Metadata.Tenant}}",
			"workspace": "{{.Metadata.Workspace}}",
			"network": "{{.Metadata.Network}}"
		},
		"spec": {
			"skuRef": "{{.SkuRef}}"
		},
		"status": {
			"state": "{{.Status.State}}"
		}
	}`

	// Route Table
	routeTablesResponseTemplateV1 = `
	{
		"items": [
        	{
				"metadata": {
					"name": "{{.Metadata.Name}}",
					"tenant": "{{.Metadata.Tenant}}",
					"workspace": "{{.Metadata.Workspace}}",
					"network": "{{.Metadata.Network}}"
				},
				"spec": {
					"routes": [
			  			{
			    			"destinationCidrBlock": "{{.RouteCidrBlock}}",
			    			"targetRef": "{{.RouteTargetRef}}"
						}
					]
				},
				"status": {
					"state": "{{.Status.State}}"
				}
			}
    	]
	}`
	routeTableResponseTemplateV1 = `
	{
		"metadata": {
			"name": "{{.Metadata.Name}}",
			"tenant": "{{.Metadata.Tenant}}",
			"workspace": "{{.Metadata.Workspace}}",
			"network": "{{.Metadata.Network}}"
		},
		"spec": {
		  "routes": [
            {
              "destinationCidrBlock": "{{.RouteCidrBlock}}",
              "targetRef": "{{.RouteTargetRef}}"
            }
          ]
		},
		"status": {
			"state": "{{.Status.State}}"
		}
	}`

	// Internet Gateway
	internetGatewaysResponseTemplateV1 = `
	{
		"items": [
        	{
				"metadata": {
					"name": "{{.Metadata.Name}}",
					"tenant": "{{.Metadata.Tenant}}",
					"workspace": "{{.Metadata.Workspace}}"
				},
				"spec": {
					"egressOnly": {{.EgressOnly}}
				},
				"status": {
					"state": "{{.Status.State}}"
				}
			}
    	]
	}`
	internetGatewayResponseTemplateV1 = `
	{
		"metadata": {
			"name": "{{.Metadata.Name}}",
			"tenant": "{{.Metadata.Tenant}}",
			"workspace": "{{.Metadata.Workspace}}"
		},
		"spec": {
			"egressOnly": {{.EgressOnly}}
		},
		"status": {
			"state": "{{.Status.State}}"
		}
	}`

	// Security Group
	securityGroupsResponseTemplateV1 = `
	{
		"items": [
        	{
				"metadata": {
					"name": "{{.Metadata.Name}}",
					"tenant": "{{.Metadata.Tenant}}",
					"workspace": "{{.Metadata.Workspace}}"
				},
				"spec": {
					"rules": [
          				{
							"direction": "{{.RuleDirection}}"
          				}
        			]
				},
				"status": {
					"state": "{{.Status.State}}"
				}
			}
    	]
	}`
	securityGroupResponseTemplateV1 = `
	{
		"metadata": {
			"name": "{{.Metadata.Name}}",
			"tenant": "{{.Metadata.Tenant}}",
			"workspace": "{{.Metadata.Workspace}}"
		},
		"spec": {
			"rules": [
				{
					"direction": "{{.RuleDirection}}"
				}
			]
		},
		"status": {
			"state": "{{.Status.State}}"
		}
	}`

	// Nic
	nicsResponseTemplateV1 = `
	{
		"items": [
        	{
				"metadata": {
					"name": "{{.Metadata.Name}}",
					"tenant": "{{.Metadata.Tenant}}",
					"workspace": "{{.Metadata.Workspace}}"
				},
				"spec": {
					"subnetRef": "{{.SubnetRef}}"
				},
				"status": {
					"state": "{{.Status.State}}"
				}
			}
    	]
	}`
	nicResponseTemplateV1 = `
	{
		"metadata": {
			"name": "{{.Metadata.Name}}",
			"tenant": "{{.Metadata.Tenant}}",
			"workspace": "{{.Metadata.Workspace}}"
		},
		"spec": {
			"subnetRef": "{{.SubnetRef}}"
		},
		"status": {
			"state": "{{.Status.State}}"
		}
	}`

	// Public Ip
	publicIpsResponseTemplateV1 = `
	{
		"items": [
        	{
				"metadata": {
					"name": "{{.Metadata.Name}}",
					"tenant": "{{.Metadata.Tenant}}",
					"workspace": "{{.Metadata.Workspace}}"
				},
				"spec": {
					"address": "{{.Address}}"
				},
				"status": {
					"state": "{{.Status.State}}"
				}
			}
    	]
	}`
	publicIpResponseTemplateV1 = `
	{
		"metadata": {
			"name": "{{.Metadata.Name}}",
			"tenant": "{{.Metadata.Tenant}}",
			"workspace": "{{.Metadata.Workspace}}"
		},
		"spec": {
			"address": "{{.Address}}"
		},
		"status": {
			"state": "{{.Status.State}}"
		}
	}`
)
