{{/*
Expand the name of the chart.
*/}}
{{- define "ecp.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name, truncated to leave room for the
"-gateway-regional" component suffix (63 - 17).
*/}}
{{- define "ecp.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 46 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 46 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 46 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{- define "ecp.gatewayGlobal.fullname" -}}
{{- printf "%s-gateway-global" (include "ecp.fullname" .) }}
{{- end }}

{{- define "ecp.gatewayRegional.fullname" -}}
{{- printf "%s-gateway-regional" (include "ecp.fullname" .) }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "ecp.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels.
*/}}
{{- define "ecp.labels" -}}
helm.sh/chart: {{ include "ecp.chart" . }}
{{ include "ecp.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels, shared by both components; the deployments add
app.kubernetes.io/component on top.
*/}}
{{- define "ecp.selectorLabels" -}}
app.kubernetes.io/name: {{ include "ecp.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{- define "ecp.gatewayGlobal.serviceAccountName" -}}
{{- if .Values.gatewayGlobal.serviceAccount.create }}
{{- default (include "ecp.gatewayGlobal.fullname" .) .Values.gatewayGlobal.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.gatewayGlobal.serviceAccount.name }}
{{- end }}
{{- end }}

{{- define "ecp.gatewayRegional.serviceAccountName" -}}
{{- if .Values.gatewayRegional.serviceAccount.create }}
{{- default (include "ecp.gatewayRegional.fullname" .) .Values.gatewayRegional.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.gatewayRegional.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Name of the Secret holding the dummy-authenticator users.json.
*/}}
{{- define "ecp.dummyUsersSecretName" -}}
{{- default (printf "%s-dummy-users" (include "ecp.fullname" .)) .Values.auth.dummyUsers.existingSecret }}
{{- end }}

{{/*
Name of the Secret holding the JWT verification key (jwt.pub).
*/}}
{{- define "ecp.jwtKeySecretName" -}}
{{- default (printf "%s-jwt-key" (include "ecp.fullname" .)) .Values.auth.jwt.existingSecret }}
{{- end }}

{{/*
Validated auth plugin name. A typo must not silently fall back to dummy.
*/}}
{{- define "ecp.authPlugin" -}}
{{- if not (has .Values.auth.plugin (list "dummy" "jwt")) }}
{{- fail (printf "unsupported auth.plugin %q — must be \"dummy\" or \"jwt\"" .Values.auth.plugin) }}
{{- end }}
{{- .Values.auth.plugin }}
{{- end }}

{{/*
Auth environment variables, consumed identically by both gateways' start
scripts: the plugin (dummy or jwt) and its config, plus the authz toggles.
*/}}
{{- define "ecp.authEnv" -}}
- name: AUTH_ENABLED
  value: {{ .Values.auth.enabled | quote }}
{{- if .Values.auth.enabled }}
- name: AUTH_PLUGIN
  value: {{ include "ecp.authPlugin" . | quote }}
{{- if eq .Values.auth.plugin "jwt" }}
- name: JWT_SIGNING_METHOD
  value: {{ .Values.auth.jwt.signingMethod | quote }}
- name: JWT_SECRET
  value: /etc/ecp/jwt/jwt.pub
{{- else }}
- name: DUMMY_AUTH_USERS
  value: /etc/ecp/auth/users.json
{{- end }}
- name: AUTHZ_ENABLED
  value: {{ .Values.auth.authz.enabled | quote }}
- name: AUTHZ_IMPL
  value: {{ .Values.auth.authz.impl | quote }}
{{- with .Values.auth.authz.skipProviders }}
- name: AUTHZ_SKIP_PROVIDERS
  value: {{ . | quote }}
{{- end }}
{{- end }}
{{- end }}
