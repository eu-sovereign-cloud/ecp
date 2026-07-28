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
Auth command-line arguments, identical for both gateways: the plugin (dummy or
jwt) and its config, plus the authz toggles.

Flags, not environment variables: the gateway images' ENTRYPOINT is the binary
itself, and the binary reads nothing but APP_ENV from the environment
(gateway/internal/auth/config.go RegisterFlags). Anything the chart configures
has to arrive as an argument or it is silently ignored.

Flags left at their binary default are omitted — --authz-skip-providers already
defaults to seca.region.
*/}}
{{- define "ecp.authArgs" -}}
{{- if .Values.auth.enabled }}
- --auth-enabled
- --auth-plugin={{ include "ecp.authPlugin" . }}
{{- if eq .Values.auth.plugin "jwt" }}
- --jwt-signing-method={{ .Values.auth.jwt.signingMethod }}
- --jwt-secret=/etc/ecp/jwt/jwt.pub
{{- else }}
- --dummy-auth-users=/etc/ecp/auth/users.json
{{- end }}
- --authz-enabled={{ .Values.auth.authz.enabled }}
{{- if and .Values.auth.authz.enabled (eq .Values.auth.authz.impl "cached") }}
- --authz-cache
{{- end }}
{{- with .Values.auth.authz.skipProviders }}
- --authz-skip-providers={{ . }}
{{- end }}
{{- end }}
{{- end }}
