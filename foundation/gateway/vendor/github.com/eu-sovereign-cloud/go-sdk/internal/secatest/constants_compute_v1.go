package secatest

const (
	// Instance Sku
	instanceSkusResponseTemplateV1 = `
	{
		"items": [
			{
				"metadata": {
					"name": "{{.Metadata.Name}}"
				},
				"labels": {
					"tier": "{{.Tier}}"
				},
				"spec": {
					"vCPU": {{.VCPU}},
					"ram": {{.Ram}}
				}
			}
		]
	}`
	instanceSkuResponseTemplateV1 = `
	{
		"metadata": {
			"name": "{{.Metadata.Name}}"
		},
		"labels": {
			"tier": "{{.Tier}}"
		},
		"spec": {
			"vCPU": {{.VCPU}},
			"ram": {{.Ram}}
		} 
	}`

	// Instance
	instancesResponseTemplateV1 = `
	{
		"items": [
			{
				"metadata": {
					"name": "{{.Metadata.Name}}",
					"tenant": "{{.Metadata.Tenant}}",
					"workspace": "{{.Metadata.Workspace}}"
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
	instanceResponseTemplateV1 = `
	{
		"metadata": {
			"name": "{{.Metadata.Name}}",
			"tenant": "{{.Metadata.Tenant}}",
			"workspace": "{{.Metadata.Workspace}}"
		},
		"spec": {
			"skuRef": "{{.SkuRef}}"
		},
		"status": {
			"state": "{{.Status.State}}"
		}
	}`
)
