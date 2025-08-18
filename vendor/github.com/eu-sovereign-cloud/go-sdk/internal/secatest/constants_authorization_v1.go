package secatest

const (
	// Role
	rolesResponseTemplateV1 = `
	{
		"items": [
			{
				"metadata": {
					"name": "{{.Metadata.Name}}",
					"tenant": "{{.Metadata.Tenant}}"
				},
				"spec": {
					"permissions": [
						{
							"verb": ["{{.PermissionVerb}}"]
						}
					]
				},
				"status": {
					"state": "{{.Status.State}}"
				}
			}
		]
	}`
	roleResponseTemplateV1 = `
	{
		"metadata": {
			"name": "{{.Metadata.Name}}",
			"tenant": "{{.Metadata.Tenant}}"
		},
		"spec": {
			"permissions": [
				{
					"verb": ["{{.PermissionVerb}}"]
				}
			]
		},
		"status": {
			"state": "{{.Status.State}}"
		}
	}`

	// Role Assignment
	roleAssignmentsResponseTemplateV1 = `
	{
		"items": [
			{
				"metadata": {
					"name": "{{.Metadata.Name}}",
					"tenant": "{{.Metadata.Tenant}}"
				},
				"spec": {
					"subs": [ "{{.Subject}}" ]
				},
				"status": {
					"state": "{{.Status.State}}"
				}
			}
		]
	}`
	roleAssignmentResponseTemplateV1 = `
	{
		"metadata": {
			"name": "{{.Metadata.Name}}",
			"tenant": "{{.Metadata.Tenant}}"
		},
		"spec": {
			"subs": [ "{{.Subject}}" ]
		},
		"status": {
			"state": "{{.Status.State}}"
		}
	}`
)
