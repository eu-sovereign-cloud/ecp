package secatest

const (
	// Storage Sku
	storageSkusResponseTemplateV1 = `
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
					"iops": {{.Iops}}
				}
			}
		]
	}`
	storageSkuResponseTemplateV1 = `
	{
		"metadata": {
			"name": "{{.Metadata.Name}}"
		},
		"labels": {
			"tier": "{{.Tier}}"
		},
		"spec": {
			"iops": {{.Iops}}
		}
	}`

	// Block Storage
	blockStoragesResponseTemplateV1 = `
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
	blockStorageResponseTemplateV1 = `
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

	// Image
	imagesResponseTemplateV1 = `
	{
		"items": [
			{
				"metadata": {
					"name": "{{.Metadata.Name}}",
					"tenant": "{{.Metadata.Tenant}}"
				},
				"spec": {
					"blockStorageRef": "{{.BlockStorageRef}}"
				},
				"status": {
					"state": "{{.Status.State}}"
				}
			}
		]
	}`
	imageResponseTemplateV1 = `
	{
		"metadata": {
			"name": "{{.Metadata.Name}}",
			"tenant": "{{.Metadata.Tenant}}"
		},
		"spec": {
			"blockStorageRef": "{{.BlockStorageRef}}"
		},
		"status": {
			"state": "{{.Status.State}}"
		}
	}`
)
