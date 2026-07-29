{{/*
Expand the name of the chart.
*/}}
{{- define "ecp-delegator.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "ecp-delegator.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "ecp-delegator.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels.
*/}}
{{- define "ecp-delegator.labels" -}}
helm.sh/chart: {{ include "ecp-delegator.chart" . }}
{{ include "ecp-delegator.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels.
*/}}
{{- define "ecp-delegator.selectorLabels" -}}
app.kubernetes.io/name: {{ include "ecp-delegator.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Validated plugin name. It selects both the image (delegator-<plugin>) and the
RBAC, so a typo or an empty value must fail loudly rather than deploy a mismatched
pair — reconciling the wrong resources with the wrong RBAC, or pulling a
nonexistent image.
*/}}
{{- define "ecp-delegator.plugin" -}}
{{- if not (has .Values.plugin (list "aruba" "dummy" "ionos")) }}
{{- fail (printf "unsupported plugin %q — set plugin to \"aruba\", \"dummy\" or \"ionos\"" .Values.plugin) }}
{{- end }}
{{- .Values.plugin }}
{{- end }}

{{/*
Create the name of the service account to use.
*/}}
{{- define "ecp-delegator.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "ecp-delegator.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}
